package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/oauth"
	"github.com/manimovassagh/rampart/internal/session"
)

const (
	testVerifier    = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	errInvalidGrant = "invalid_grant"
)

type mockTokenStore struct {
	oauthClient    *model.OAuthClient
	oauthErr       error
	authCode       *model.AuthorizationCode
	consumeErr     error
	userByID       *model.User
	userByIDErr    error
	orgSettings    *model.OrgSettings
	orgSettingsErr error
	roles          []string
	rolesErr       error
}

func (m *mockTokenStore) GetOAuthClient(_ context.Context, _ string) (*model.OAuthClient, error) {
	return m.oauthClient, m.oauthErr
}

func (m *mockTokenStore) ConsumeAuthorizationCode(_ context.Context, _ string) (*model.AuthorizationCode, error) {
	return m.authCode, m.consumeErr
}

func (m *mockTokenStore) GetUserByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return m.userByID, m.userByIDErr
}

func (m *mockTokenStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.orgSettings, m.orgSettingsErr
}

func (m *mockTokenStore) GetEffectiveUserRoles(_ context.Context, _ uuid.UUID) ([]string, error) {
	return m.roles, m.rolesErr
}

// ── stub methods to satisfy store.AuthCodeStore ──

func (m *mockTokenStore) StoreAuthorizationCode(_ context.Context, _, _ string, _, _ uuid.UUID, _, _, _, _ string, _ time.Time) error {
	return nil
}
func (m *mockTokenStore) DeleteExpiredAuthorizationCodes(_ context.Context) (int64, error) {
	return 0, nil
}

// ── stub methods to satisfy store.UserReader ──

func (m *mockTokenStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockTokenStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockTokenStore) FindUserByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}

// ── stub methods to satisfy store.OrgSettingsReadWriter ──

func (m *mockTokenStore) UpdateOrgSettings(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	return nil, nil
}

// ── stub methods to satisfy store.GroupReader ──

func (m *mockTokenStore) GetGroupByID(_ context.Context, _ uuid.UUID) (*model.Group, error) {
	return nil, nil
}
func (m *mockTokenStore) GetGroupMembers(_ context.Context, _ uuid.UUID) ([]*model.GroupMember, error) {
	return nil, nil
}
func (m *mockTokenStore) GetGroupRoles(_ context.Context, _ uuid.UUID) ([]*model.GroupRoleAssignment, error) {
	return nil, nil
}
func (m *mockTokenStore) GetUserGroups(_ context.Context, _ uuid.UUID) ([]*model.Group, error) {
	return nil, nil
}
func (m *mockTokenStore) CountGroupMembers(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockTokenStore) CountGroupRoles(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockTokenStore) CountGroups(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }

func TestTokenMissingGrantType(t *testing.T) {
	store := &mockTokenStore{}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader("code=abc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTokenMissingParams(t *testing.T) {
	store := &mockTokenStore{}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader("grant_type=authorization_code"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTokenInvalidCode(t *testing.T) {
	orgID := uuid.New()
	store := &mockTokenStore{
		oauthClient: newTestOAuthClient(orgID),
		authCode:    nil, // no code found
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=badcode&client_id=test-client&redirect_uri=http://localhost:3002/callback&code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != errInvalidGrant {
		t.Errorf("error = %q, want invalid_grant", resp["error"])
	}
}

func TestTokenWrongClientID(t *testing.T) {
	orgID := uuid.New()
	verifier := testVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	store := &mockTokenStore{
		oauthClient: newTestOAuthClient(orgID),
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      "different-client", // mismatch
			UserID:        uuid.New(),
			OrgID:         orgID,
			RedirectURI:   "http://localhost:3002/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=test-client&redirect_uri=http://localhost:3002/callback&code_verifier=" + verifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTokenWrongRedirectURI(t *testing.T) {
	orgID := uuid.New()
	verifier := testVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	store := &mockTokenStore{
		oauthClient: newTestOAuthClient(orgID),
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      "test-client",
			UserID:        uuid.New(),
			OrgID:         orgID,
			RedirectURI:   "http://localhost:3002/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=test-client&redirect_uri=http://evil.com/callback&code_verifier=" + verifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTokenWrongCodeVerifier(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	correctVerifier := testVerifier
	challenge := oauth.ComputeS256Challenge(correctVerifier)

	store := &mockTokenStore{
		oauthClient: newTestOAuthClient(orgID),
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      "test-client",
			UserID:        userID,
			OrgID:         orgID,
			RedirectURI:   "http://localhost:3002/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	wrongVerifier := "WRONG-VERIFIER-that-is-at-least-43-characters-long!!!"
	body := "grant_type=authorization_code&code=validcode&client_id=test-client&redirect_uri=http://localhost:3002/callback&code_verifier=" + wrongVerifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != errInvalidGrant {
		t.Errorf("error = %q, want invalid_grant", resp["error"])
	}
}

func TestTokenValidExchange(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	verifier := testVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	user := &model.User{
		ID:       userID,
		OrgID:    orgID,
		Username: "admin",
		Email:    "admin@test.com",
		Enabled:  true,
	}

	store := &mockTokenStore{
		oauthClient: newTestOAuthClient(orgID),
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      "test-client",
			UserID:        userID,
			OrgID:         orgID,
			RedirectURI:   "http://localhost:3002/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
		userByID: user,
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=test-client&redirect_uri=http://localhost:3002/callback&code_verifier=" + verifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp TokenResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh_token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("token_type = %q, want Bearer", resp.TokenType)
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("expires_in = %d, want 900", resp.ExpiresIn)
	}

	// Verify Cache-Control is no-store
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
}

func TestTokenMethodNotAllowed(t *testing.T) {
	store := &mockTokenStore{}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/oauth/token", http.NoBody)
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestTokenUnknownClientID(t *testing.T) {
	store := &mockTokenStore{
		oauthClient: nil, // client not found
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=unknown-client&redirect_uri=http://localhost:3002/callback&code_verifier=" + testVerifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "invalid_client" {
		t.Errorf("error = %q, want invalid_client", resp["error"])
	}
}

func TestTokenClientFetchError(t *testing.T) {
	store := &mockTokenStore{
		oauthErr: fmt.Errorf("db connection failed"),
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=test-client&redirect_uri=http://localhost:3002/callback&code_verifier=" + testVerifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestTokenConsumeCodeError(t *testing.T) {
	orgID := uuid.New()
	store := &mockTokenStore{
		oauthClient: newTestOAuthClient(orgID),
		consumeErr:  fmt.Errorf("db error consuming code"),
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=test-client&redirect_uri=http://localhost:3002/callback&code_verifier=" + testVerifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestTokenUserDisabledAfterCodeExchange(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	verifier := testVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	disabledUser := &model.User{
		ID:       userID,
		OrgID:    orgID,
		Username: "disabled",
		Email:    "disabled@test.com",
		Enabled:  false,
	}

	store := &mockTokenStore{
		oauthClient: newTestOAuthClient(orgID),
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      "test-client",
			UserID:        userID,
			OrgID:         orgID,
			RedirectURI:   "http://localhost:3002/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
		userByID: disabledUser,
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=test-client&redirect_uri=http://localhost:3002/callback&code_verifier=" + verifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != errInvalidGrant {
		t.Errorf("error = %q, want invalid_grant", resp["error"])
	}
}

func TestTokenUserNotFoundAfterCodeExchange(t *testing.T) {
	orgID := uuid.New()
	verifier := testVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	store := &mockTokenStore{
		oauthClient: newTestOAuthClient(orgID),
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      "test-client",
			UserID:        uuid.New(),
			OrgID:         orgID,
			RedirectURI:   "http://localhost:3002/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
		userByID: nil, // user deleted between auth and token exchange
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=test-client&redirect_uri=http://localhost:3002/callback&code_verifier=" + verifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTokenUserFetchError(t *testing.T) {
	orgID := uuid.New()
	verifier := testVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	store := &mockTokenStore{
		oauthClient: newTestOAuthClient(orgID),
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      "test-client",
			UserID:        uuid.New(),
			OrgID:         orgID,
			RedirectURI:   "http://localhost:3002/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
		userByIDErr: fmt.Errorf("db error"),
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=test-client&redirect_uri=http://localhost:3002/callback&code_verifier=" + verifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestTokenSessionCreateError(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	verifier := testVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	user := &model.User{
		ID:       userID,
		OrgID:    orgID,
		Username: "testuser",
		Email:    "test@test.com",
		Enabled:  true,
	}

	store := &mockTokenStore{
		oauthClient: newTestOAuthClient(orgID),
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      "test-client",
			UserID:        userID,
			OrgID:         orgID,
			RedirectURI:   "http://localhost:3002/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
		userByID: user,
	}
	sessions := &mockSessionStore{createErr: fmt.Errorf("session store down")}
	h := NewTokenHandler(store, sessions, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=test-client&redirect_uri=http://localhost:3002/callback&code_verifier=" + verifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestTokenWithOrgSettings(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	verifier := testVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	user := &model.User{
		ID:       userID,
		OrgID:    orgID,
		Username: "admin",
		Email:    "admin@test.com",
		Enabled:  true,
	}

	store := &mockTokenStore{
		oauthClient: newTestOAuthClient(orgID),
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      "test-client",
			UserID:        userID,
			OrgID:         orgID,
			RedirectURI:   "http://localhost:3002/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
		userByID: user,
		orgSettings: &model.OrgSettings{
			AccessTokenTTL:  30 * time.Minute,
			RefreshTokenTTL: 14 * 24 * time.Hour,
		},
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=test-client&redirect_uri=http://localhost:3002/callback&code_verifier=" + verifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp TokenResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ExpiresIn != 1800 {
		t.Errorf("expires_in = %d, want 1800 (30 min org setting)", resp.ExpiresIn)
	}
}

func TestTokenWithAdminClientIncludesAdminRole(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	verifier := testVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	user := &model.User{
		ID:       userID,
		OrgID:    orgID,
		Username: "adminuser",
		Email:    "adminuser@test.com",
		Enabled:  true,
	}

	store := &mockTokenStore{
		oauthClient: &model.OAuthClient{
			ID:           adminClientID,
			OrgID:        orgID,
			Name:         "Rampart Admin",
			ClientType:   "public",
			RedirectURIs: []string{"http://localhost:3002/callback"},
		},
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      adminClientID,
			UserID:        userID,
			OrgID:         orgID,
			RedirectURI:   "http://localhost:3002/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
		userByID: user,
		roles:    []string{"admin", "user"},
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=" + adminClientID + "&redirect_uri=http://localhost:3002/callback&code_verifier=" + verifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRevokeValidToken(t *testing.T) {
	sessID := uuid.New()
	userID := uuid.New()
	sessions := &mockSessionStore{
		found: &session.Session{
			ID:     sessID,
			UserID: userID,
		},
	}
	h := NewTokenHandler(&mockTokenStore{}, sessions, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "token=valid-refresh-token"
	req := httptest.NewRequest(http.MethodPost, "/oauth/revoke", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Revoke(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRevokeInvalidToken(t *testing.T) {
	sessions := &mockSessionStore{
		found: nil, // token not found
	}
	h := NewTokenHandler(&mockTokenStore{}, sessions, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "token=unknown-refresh-token"
	req := httptest.NewRequest(http.MethodPost, "/oauth/revoke", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Revoke(w, req)

	// Per RFC 7009: invalid tokens do not cause an error response
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRevokeEmptyToken(t *testing.T) {
	h := NewTokenHandler(&mockTokenStore{}, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "token="
	req := httptest.NewRequest(http.MethodPost, "/oauth/revoke", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Revoke(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRevokeMethodNotAllowed(t *testing.T) {
	h := NewTokenHandler(&mockTokenStore{}, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/oauth/revoke", http.NoBody)
	w := httptest.NewRecorder()

	h.Revoke(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestTokenValidExchangeReturnsIDToken(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	verifier := testVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	user := &model.User{
		ID:       userID,
		OrgID:    orgID,
		Username: "admin",
		Email:    "admin@test.com",
		Enabled:  true,
	}

	store := &mockTokenStore{
		oauthClient: newTestOAuthClient(orgID),
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      "test-client",
			UserID:        userID,
			OrgID:         orgID,
			RedirectURI:   "http://localhost:3002/callback",
			CodeChallenge: challenge,
			Scope:         "openid",
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
		userByID: user,
	}
	h := NewTokenHandler(store, &mockSessionStore{}, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)

	body := "grant_type=authorization_code&code=validcode&client_id=test-client&redirect_uri=http://localhost:3002/callback&code_verifier=" + verifier
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	idToken, ok := resp["id_token"].(string)
	if !ok || idToken == "" {
		t.Error("expected non-empty id_token in response when scope includes openid")
	}
}
