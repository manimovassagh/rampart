package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/model"
)

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// mockUserStore implements UserStore for testing.
type mockUserStore struct {
	defaultOrgID    uuid.UUID
	defaultOrgErr   error
	slugOrgID       uuid.UUID
	slugOrgErr      error
	emailUser       *model.User
	emailErr        error
	usernameUser    *model.User
	usernameErr     error
	createdUser     *model.User
	createErr       error
	orgSettings     *model.OrgSettings
	orgSettingsErr  error
}

func (m *mockUserStore) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return m.defaultOrgID, m.defaultOrgErr
}

func (m *mockUserStore) GetOrganizationIDBySlug(_ context.Context, _ string) (uuid.UUID, error) {
	if m.slugOrgID != uuid.Nil {
		return m.slugOrgID, m.slugOrgErr
	}
	return m.defaultOrgID, m.slugOrgErr
}

func (m *mockUserStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return m.emailUser, m.emailErr
}

func (m *mockUserStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return m.usernameUser, m.usernameErr
}

func (m *mockUserStore) CreateUser(_ context.Context, user *model.User) (*model.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createdUser != nil {
		return m.createdUser, nil
	}
	now := time.Now()
	return &model.User{
		ID:         uuid.New(),
		OrgID:      user.OrgID,
		Username:   user.Username,
		Email:      user.Email,
		GivenName:  user.GivenName,
		FamilyName: user.FamilyName,
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (m *mockUserStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.orgSettings, m.orgSettingsErr
}

func newTestRegisterHandler(store *mockUserStore) *RegisterHandler {
	return NewRegisterHandler(store, noopLogger())
}

func validRegistrationJSON() []byte {
	return []byte(`{
		"username": "johndoe",
		"email": "john@example.com",
		"password": "Str0ng!Pass",
		"given_name": "John",
		"family_name": "Doe"
	}`)
}

func TestRegisterSuccess(t *testing.T) {
	orgID := uuid.New()
	store := &mockUserStore{defaultOrgID: orgID}
	h := newTestRegisterHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(validRegistrationJSON()))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp model.UserResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Username != "johndoe" {
		t.Errorf("username = %q, want johndoe", resp.Username)
	}
	if resp.Email != "john@example.com" {
		t.Errorf("email = %q, want john@example.com", resp.Email)
	}
}

func TestRegisterResponseNeverContainsPasswordHash(t *testing.T) {
	store := &mockUserStore{defaultOrgID: uuid.New()}
	h := newTestRegisterHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(validRegistrationJSON()))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	body := w.Body.String()
	if bytes.Contains([]byte(body), []byte("password_hash")) {
		t.Error("response body contains password_hash, which must never be exposed")
	}
	if bytes.Contains([]byte(body), []byte("argon2id")) {
		t.Error("response body contains argon2 hash data")
	}
}

func TestRegisterInvalidJSON(t *testing.T) {
	store := &mockUserStore{defaultOrgID: uuid.New()}
	h := newTestRegisterHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRegisterValidationErrors(t *testing.T) {
	store := &mockUserStore{defaultOrgID: uuid.New()}
	h := newTestRegisterHandler(store)

	body := []byte(`{"username": "", "email": "", "password": "weak"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var ve apierror.ValidationError
	if err := json.NewDecoder(w.Body).Decode(&ve); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if ve.Code != "validation_error" {
		t.Errorf("error = %q, want validation_error", ve.Code)
	}
	if len(ve.Fields) < 2 {
		t.Errorf("expected at least 2 field errors, got %d", len(ve.Fields))
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	store := &mockUserStore{
		defaultOrgID: uuid.New(),
		emailUser:    &model.User{Email: "john@example.com"},
	}
	h := newTestRegisterHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(validRegistrationJSON()))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestRegisterDuplicateUsername(t *testing.T) {
	store := &mockUserStore{
		defaultOrgID: uuid.New(),
		usernameUser: &model.User{Username: "johndoe"},
	}
	h := newTestRegisterHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(validRegistrationJSON()))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestRegisterDefaultOrgError(t *testing.T) {
	store := &mockUserStore{
		defaultOrgErr: fmt.Errorf("db connection failed"),
	}
	h := newTestRegisterHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(validRegistrationJSON()))
	w := httptest.NewRecorder()

	h.Register(w, req)

	// Org resolution failures return 400 ("Organization not found") to avoid leaking internal details.
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRegisterCreateUserError(t *testing.T) {
	store := &mockUserStore{
		defaultOrgID: uuid.New(),
		createErr:    fmt.Errorf("unique constraint violation"),
	}
	h := newTestRegisterHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(validRegistrationJSON()))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRegisterEmailNormalization(t *testing.T) {
	store := &mockUserStore{defaultOrgID: uuid.New()}
	h := newTestRegisterHandler(store)

	body := []byte(`{
		"username": "johndoe",
		"email": "  John@Example.COM  ",
		"password": "Str0ng!Pass",
		"given_name": "John",
		"family_name": "Doe"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp model.UserResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Email != "john@example.com" {
		t.Errorf("email = %q, want john@example.com (normalized)", resp.Email)
	}
}
