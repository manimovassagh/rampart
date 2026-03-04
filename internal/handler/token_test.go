package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/oauth"
)

type mockTokenStore struct {
	oauthClient   *model.OAuthClient
	oauthErr      error
	authCode      *model.AuthorizationCode
	consumeErr    error
	userByID      *model.User
	userByIDErr   error
	orgSettings   *model.OrgSettings
	orgSettingsErr error
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

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", resp["error"])
	}
}

func TestTokenWrongClientID(t *testing.T) {
	orgID := uuid.New()
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
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
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
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
	correctVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
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

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "invalid_grant" {
		t.Errorf("error = %q, want invalid_grant", resp["error"])
	}
}

func TestTokenValidExchange(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
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
