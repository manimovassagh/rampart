package handler

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
)

// testRSAKeyPEM is a 2048-bit RSA key used only in tests.
const testRSAKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAvN8Ex780DiM6xO5PgniD7BbnTEGx1IkX1LE0EbrGrZJHcVbX
IiUbxBcnAMl/PqPtpS02pf0IgaGPM1DgO10eNcGxRvUcw/H0hbOEgMFIch71egvD
d/Ag8m18vO0MaoSh7xBlJSIfgRLCpyoWwghurFuMViwMcst6Cg35W8+IOCL7KOkj
OdFWIT7baffJ2w7LGq0i3/TlSmoUNVF+sZzM2vj4QMC6T7bUI9ISx9KP1wvxAz99
c1PoSi1bu2e/Yz3CyXeg00Z1BVWNGEcM98iaajTMGP1QmqsavNMjO72Ub+1XpyQ/
ve36uznlaqHhBZqrtTI+YugLtYIRq3etuI2HmwIDAQABAoIBAB9gzeKBmZxfrfvZ
u8vpScGHbJX2tByjShpD9mqbpTZg/w2NZ+B8WciSMCCpWUKG6YxvnoylJSykMq5L
2XUDW2mC7HjlcAn9wKoV0QWzFt4e1pmYKrlaY57jIb4hg9aOgni9OJCawrEm9L/g
9jb2P6zS6NXIK6lGtNfGyo6+Q9tPa2nMF86xrzscKuT8hLq2B4YN3jdL5tCIsfO/
IcnDwL6eCw+sjjeKfsEXful+9JZSAoKr1sukeW+xSE26hZwhxmwyHDPST/D5QkJo
NvbikjKpRRWwuUTondPbpJ3d2C6vIYVXtJjdzE9RdfKDPeuTpRRbgr/bf9LLO9E6
9k6gEYECgYEA2hHOT3Wm+WnVTILaCh4Z0/qeMrpBVcUgNNfaedXdYS4MND2TY2Wr
cd2IUGYvjFJs2neattXYijR7dC9i3j40wYTE3ak1S8rjVdTjF9eoV73zfWjGLUoT
xKTidmxhaWixJJxOQXoYVxwumsebCoRJLQqs8DNauaA3HzHpRSKIm5MCgYEA3bkR
YndjBzUrWFPZnhY89DUvwTCPrbMyHSUCpRiPlcDkQwgTnqvEgutKUtEKUvWDOu0d
4DVjK/P2LwwL8tuA82WaTOZjFTZ3yGNPC+gJeamKS4PgQviEdldLU53VKQpy7kgg
bxwW+aY+hKpVo+9RerdMJRn2SUhiHaHWdedQuNkCgYEApU9UN5Y3wuEAyiRzx7Gz
4Kce39OkDbIGzShIvY1raezvYXbAUVxUUFggqtob92LQk/iRN0L7CSHp6FS3vUQo
1/6fAm3wMgmWto1QrdVVD1a2y33upYx/WdWoux9D5RVxHBDFngtBgl+h0MG5/Yn0
swlhuiEkCI2025gJfthD+LMCgYAD6u85tC5VxES9zM19k5sEHaR4X2lKgm4SQcMo
M6Tl2oCuBoiCNzrDrXCkwfjSum/VLLdobMkRz7+72RSk9+fxZQwy66c4irvXGJoe
9bylH6/H4c6moEmG5cf49EL99KdPOosIK5DkXGGiangU63efGXoI9cp6RQMmzuNB
NhMhEQKBgQDF+33l7x8rxERo1x0E/nwaPr4kISE2RbQSxXL7UARAg457+ol/bs36
O4bU15NoXuwZ1ShWIKMWxLVcEysw7pDHs3xaiokx1SIrn9nVkaOb9iK8B8qgCQ2o
W/xlOTOB+SaEdBMOGPE8JlFPydshVuTUj5wMzN4GejazNxtEXbbGaQ==
-----END RSA PRIVATE KEY-----`

var testPrivKey *rsa.PrivateKey

func init() {
	block, _ := pem.Decode([]byte(testRSAKeyPEM))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic("failed to parse test RSA key: " + err.Error())
	}
	testPrivKey = key
}

const (
	testKID    = "test-kid-123"
	testIssuer = "http://localhost:8080"
)

// mockLoginStore implements LoginStore for testing.
type mockLoginStore struct {
	defaultOrgID   uuid.UUID
	defaultOrgErr  error
	slugOrgID      uuid.UUID
	slugOrgErr     error
	emailUser      *model.User
	emailErr       error
	usernameUser   *model.User
	usernameErr    error
	userByID       *model.User
	userByIDErr    error
	updateErr      error
	orgSettings    *model.OrgSettings
	orgSettingsErr error
}

func (m *mockLoginStore) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return m.defaultOrgID, m.defaultOrgErr
}

func (m *mockLoginStore) GetOrganizationIDBySlug(_ context.Context, _ string) (uuid.UUID, error) {
	if m.slugOrgID != uuid.Nil {
		return m.slugOrgID, m.slugOrgErr
	}
	return m.defaultOrgID, m.slugOrgErr
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

func (m *mockLoginStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.orgSettings, m.orgSettingsErr
}

func (m *mockLoginStore) GetEffectiveUserRoles(_ context.Context, _ uuid.UUID) ([]string, error) {
	return nil, nil
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
	return NewLoginHandler(store, sessions, noopLogger(), nil, testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)
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

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginEmptyBody(t *testing.T) {
	store := &mockLoginStore{defaultOrgID: uuid.New()}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	cases := []struct {
		name string
		body string
	}{
		{"emptyJSON", `{}`},
		{"missingPassword", `{"identifier":"admin@rampart.local"}`},
		{"missingIdentifier", `{"password":"Str0ng!Pass"}`},
		{"whitespaceIdentifier", `{"identifier":"  ","password":"Str0ng!Pass"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte(tc.body)))
			w := httptest.NewRecorder()

			h.Login(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
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

func TestLogoutInvalidJSON(t *testing.T) {
	store := &mockLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	h.Logout(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLogoutSessionNotFound(t *testing.T) {
	store := &mockLoginStore{}
	sessions := &mockSessionStore{found: nil}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": "nonexistent-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Logout(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestLogoutDeleteSessionError(t *testing.T) {
	sess := &session.Session{ID: uuid.New(), UserID: uuid.New()}
	store := &mockLoginStore{}
	sessions := &mockSessionStore{found: sess, deleteErr: fmt.Errorf("delete failed")}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": "some-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Logout(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestLogoutFindSessionError(t *testing.T) {
	store := &mockLoginStore{}
	sessions := &mockSessionStore{findErr: fmt.Errorf("redis connection failed")}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": "some-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Logout(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRefreshInvalidJSON(t *testing.T) {
	store := &mockLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	req := httptest.NewRequest(http.MethodPost, "/token/refresh", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRefreshFindSessionError(t *testing.T) {
	store := &mockLoginStore{}
	sessions := &mockSessionStore{findErr: fmt.Errorf("redis down")}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": "some-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/token/refresh", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRefreshUserNotFound(t *testing.T) {
	sess := &session.Session{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	store := &mockLoginStore{userByID: nil}
	sessions := &mockSessionStore{found: sess}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": "some-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/token/refresh", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRefreshDisabledUser(t *testing.T) {
	user := newTestUser()
	user.Enabled = false
	sess := &session.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	store := &mockLoginStore{userByID: user}
	sessions := &mockSessionStore{found: sess}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": "some-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/token/refresh", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRefreshGetUserError(t *testing.T) {
	sess := &session.Session{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	store := &mockLoginStore{userByIDErr: fmt.Errorf("db error")}
	sessions := &mockSessionStore{found: sess}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": "some-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/token/refresh", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestLoginWithOrgSlug(t *testing.T) {
	user := newTestUser()
	orgID := uuid.New()
	store := &mockLoginStore{
		slugOrgID: orgID,
		emailUser: user,
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "admin@rampart.local", "password": "Str0ng!Pass", "org_slug": "my-org"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestLoginOrgSlugError(t *testing.T) {
	store := &mockLoginStore{
		slugOrgID:  uuid.New(),
		slugOrgErr: fmt.Errorf("org not found"),
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "admin@rampart.local", "password": "Str0ng!Pass", "org_slug": "nonexistent"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLoginDefaultOrgError(t *testing.T) {
	store := &mockLoginStore{defaultOrgErr: fmt.Errorf("db down")}
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

func TestLoginEmailLookupError(t *testing.T) {
	store := &mockLoginStore{
		defaultOrgID: uuid.New(),
		emailErr:     fmt.Errorf("db error"),
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "admin@rampart.local", "password": "Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestLoginUsernameLookupError(t *testing.T) {
	store := &mockLoginStore{
		defaultOrgID: uuid.New(),
		usernameErr:  fmt.Errorf("db error"),
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "admin", "password": "Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestLoginSessionCreateError(t *testing.T) {
	user := newTestUser()
	store := &mockLoginStore{
		defaultOrgID: user.OrgID,
		emailUser:    user,
	}
	sessions := &mockSessionStore{createErr: fmt.Errorf("redis down")}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "admin@rampart.local", "password": "Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestLoginOrgSettingsError(t *testing.T) {
	user := newTestUser()
	store := &mockLoginStore{
		defaultOrgID:   user.OrgID,
		emailUser:      user,
		orgSettingsErr: fmt.Errorf("db error fetching settings"),
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "admin@rampart.local", "password": "Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	// Should succeed even if org settings fail (falls back to defaults)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp LoginResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	// Default accessTTL is 15 minutes = 900 seconds
	if resp.ExpiresIn != 900 {
		t.Errorf("expires_in = %d, want 900 (default)", resp.ExpiresIn)
	}
}

func TestLoginWhitespaceTrimmingOnIdentifier(t *testing.T) {
	user := newTestUser()
	store := &mockLoginStore{
		defaultOrgID: user.OrgID,
		emailUser:    user,
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "  admin@rampart.local  ", "password": "Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestLoginOnlyPasswordEmpty(t *testing.T) {
	store := &mockLoginStore{defaultOrgID: uuid.New()}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "admin@rampart.local", "password": ""}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLoginOnlyIdentifierEmpty(t *testing.T) {
	store := &mockLoginStore{defaultOrgID: uuid.New()}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier": "", "password": "Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRefreshDisabledUserAccount(t *testing.T) {
	user := newTestUser()
	user.Enabled = false
	sess := &session.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	store := &mockLoginStore{userByID: user}
	sessions := &mockSessionStore{found: sess}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token": "some-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/token/refresh", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLogoutWithUserAudit(t *testing.T) {
	user := newTestUser()
	sess := &session.Session{ID: uuid.New(), UserID: user.ID}
	store := &mockLoginStore{userByID: user}
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

func TestLoginWithOrgSettings(t *testing.T) {
	user := newTestUser()
	store := &mockLoginStore{
		defaultOrgID: user.OrgID,
		emailUser:    user,
		orgSettings: &model.OrgSettings{
			AccessTokenTTL:  30 * time.Minute,
			RefreshTokenTTL: 14 * 24 * time.Hour,
		},
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
	// With 30 min org setting, expires_in should be 1800
	if resp.ExpiresIn != 1800 {
		t.Errorf("expires_in = %d, want 1800", resp.ExpiresIn)
	}
}
