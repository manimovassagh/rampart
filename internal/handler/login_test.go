package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
)

const testJWTSecret = "this-is-a-test-secret-that-is-at-least-32-bytes-long"

// mockLoginStore implements LoginStore for testing.
type mockLoginStore struct {
	defaultOrgID  uuid.UUID
	defaultOrgErr error
	emailUser     *model.User
	emailErr      error
	usernameUser  *model.User
	usernameErr   error
	userByID      *model.User
	userByIDErr   error
	updateErr     error
}

func (m *mockLoginStore) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return m.defaultOrgID, m.defaultOrgErr
}

func (m *mockLoginStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return m.emailUser, m.emailErr
}

func (m *mockLoginStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return m.usernameUser, m.usernameErr
}

func (m *mockLoginStore) GetUserByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return m.userByID, m.userByIDErr
}

func (m *mockLoginStore) UpdateLastLoginAt(_ context.Context, _ uuid.UUID) error {
	return m.updateErr
}

// mockSessionStore implements session.Store for testing.
type mockSessionStore struct {
	created   *session.Session
	createErr error
	found     *session.Session
	findErr   error
	deleteErr error
}

func (m *mockSessionStore) Create(_ context.Context, userID uuid.UUID, _ string, expiresAt time.Time) (*session.Session, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.created != nil {
		return m.created, nil
	}
	return &session.Session{
		ID:        uuid.New(),
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockSessionStore) FindByRefreshToken(_ context.Context, _ string) (*session.Session, error) {
	return m.found, m.findErr
}

func (m *mockSessionStore) Delete(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockSessionStore) DeleteByUserID(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func newTestUser() *model.User {
	hash, _ := auth.HashPassword("Str0ng!Pass")
	return &model.User{
		ID:           uuid.New(),
		OrgID:        uuid.New(),
		Username:     "admin",
		Email:        "admin@rampart.local",
		PasswordHash: []byte(hash),
		Enabled:      true,
		GivenName:    "Admin",
		FamilyName:   "User",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func newTestLoginHandler(store *mockLoginStore, sessions *mockSessionStore) *LoginHandler {
	return NewLoginHandler(store, sessions, noopLogger(), testJWTSecret, 15*time.Minute, 7*24*time.Hour)
}

func TestLoginSuccess(t *testing.T) {
	user := newTestUser()
	store := &mockLoginStore{
		defaultOrgID: user.OrgID,
		emailUser:    user,
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "admin@rampart.local", "password": "Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp LoginResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("token_type = %q, want Bearer", resp.TokenType)
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("expires_in = %d, want 900", resp.ExpiresIn)
	}
	if resp.User == nil {
		t.Fatal("expected user in response")
	}
	if resp.User.Username != "admin" {
		t.Errorf("user.username = %q, want admin", resp.User.Username)
	}
}

func TestLoginByUsername(t *testing.T) {
	user := newTestUser()
	store := &mockLoginStore{
		defaultOrgID: user.OrgID,
		usernameUser: user,
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "admin", "password": "Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	user := newTestUser()
	store := &mockLoginStore{
		defaultOrgID: user.OrgID,
		emailUser:    user,
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "admin@rampart.local", "password": "WrongPass1!"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLoginUserNotFound(t *testing.T) {
	store := &mockLoginStore{defaultOrgID: uuid.New()}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "nobody@test.com", "password": "Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLoginEmptyCredentials(t *testing.T) {
	store := &mockLoginStore{defaultOrgID: uuid.New()}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "", "password": ""}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLoginDisabledUser(t *testing.T) {
	user := newTestUser()
	user.Enabled = false
	store := &mockLoginStore{
		defaultOrgID: user.OrgID,
		emailUser:    user,
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "admin@rampart.local", "password": "Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLoginInvalidJSON(t *testing.T) {
	store := &mockLoginStore{defaultOrgID: uuid.New()}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginResponseNeverContainsPasswordHash(t *testing.T) {
	user := newTestUser()
	store := &mockLoginStore{
		defaultOrgID: user.OrgID,
		emailUser:    user,
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "admin@rampart.local", "password": "Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	respBody := w.Body.String()
	if bytes.Contains([]byte(respBody), []byte("password_hash")) {
		t.Error("response body contains password_hash")
	}
	if bytes.Contains([]byte(respBody), []byte("argon2id")) {
		t.Error("response body contains argon2 hash data")
	}
}

func TestRefreshSuccess(t *testing.T) {
	user := newTestUser()
	sess := &session.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	store := &mockLoginStore{userByID: user}
	sessions := &mockSessionStore{found: sess}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": "some-refresh-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/token/refresh", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp RefreshResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestRefreshInvalidToken(t *testing.T) {
	store := &mockLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": "invalid-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/token/refresh", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRefreshEmptyToken(t *testing.T) {
	store := &mockLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": ""}`)
	req := httptest.NewRequest(http.MethodPost, "/token/refresh", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLogoutSuccess(t *testing.T) {
	sess := &session.Session{ID: uuid.New(), UserID: uuid.New()}
	store := &mockLoginStore{}
	sessions := &mockSessionStore{found: sess}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": "some-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Logout(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestLogoutNoToken(t *testing.T) {
	store := &mockLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": ""}`)
	req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Logout(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}
