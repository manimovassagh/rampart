package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/oauth"
)

const (
	testState             = "teststate"
	testAdminVerifier     = "testverifier1234567890abcdefghijk"
	testAdminPKCEVerifier = "test-verifier-for-pkce-challenge-must-be-43-chars-long"
)

// mockAdminLoginStore implements AdminLoginStore for testing.
type mockAdminLoginStore struct {
	oauthClient    *model.OAuthClient
	oauthClientErr error
	authCode       *model.AuthorizationCode
	authCodeErr    error
	userByID       *model.User
	userByIDErr    error
	orgSettings    *model.OrgSettings
	orgSettingsErr error
	roles          []string
	rolesErr       error
}

func (m *mockAdminLoginStore) GetOAuthClient(_ context.Context, _ string) (*model.OAuthClient, error) {
	return m.oauthClient, m.oauthClientErr
}

func (m *mockAdminLoginStore) ConsumeAuthorizationCode(_ context.Context, _ string) (*model.AuthorizationCode, error) {
	return m.authCode, m.authCodeErr
}

func (m *mockAdminLoginStore) GetUserByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return m.userByID, m.userByIDErr
}

func (m *mockAdminLoginStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.orgSettings, m.orgSettingsErr
}

func (m *mockAdminLoginStore) GetEffectiveUserRoles(_ context.Context, _ uuid.UUID) ([]string, error) {
	return m.roles, m.rolesErr
}

func newTestAdminLoginHandler(store *mockAdminLoginStore, sessions *mockSessionStore) *AdminLoginHandler {
	hmacKey := []byte("test-hmac-key-for-admin-sessions")
	return NewAdminLoginHandler(
		store, sessions, noopLogger(), nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
		hmacKey,
	)
}

func TestAdminLoginRedirect(t *testing.T) {
	store := &mockAdminLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	req := httptest.NewRequest(http.MethodGet, "/admin/login", http.NoBody)
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if location == "" {
		t.Fatal("expected Location header to be set")
	}

	// Verify redirect contains required OAuth parameters
	if !contains(location, "client_id=rampart-admin") {
		t.Errorf("redirect missing client_id, got %s", location)
	}
	if !contains(location, "response_type=code") {
		t.Errorf("redirect missing response_type, got %s", location)
	}
	if !contains(location, "code_challenge=") {
		t.Errorf("redirect missing code_challenge, got %s", location)
	}
	if !contains(location, "code_challenge_method=S256") {
		t.Errorf("redirect missing code_challenge_method, got %s", location)
	}
	if !contains(location, "state=") {
		t.Errorf("redirect missing state, got %s", location)
	}

	// Verify OAuth cookie was set
	cookies := w.Result().Cookies()
	var foundCookie bool
	for _, c := range cookies {
		if c.Name == adminOAuthCookieName {
			foundCookie = true
			if !c.HttpOnly {
				t.Error("oauth cookie should be HttpOnly")
			}
			break
		}
	}
	if !foundCookie {
		t.Error("expected admin OAuth cookie to be set")
	}
}

func TestAdminCallbackMissingCode(t *testing.T) {
	store := &mockAdminLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	req := httptest.NewRequest(http.MethodGet, "/admin/callback?state=abc", http.NoBody)
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != middleware.AdminLoginPath {
		t.Errorf("location = %q, want %q", loc, middleware.AdminLoginPath)
	}
}

func TestAdminCallbackMissingState(t *testing.T) {
	store := &mockAdminLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	req := httptest.NewRequest(http.MethodGet, "/admin/callback?code=abc", http.NoBody)
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminCallbackMissingCookie(t *testing.T) {
	store := &mockAdminLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	req := httptest.NewRequest(http.MethodGet, "/admin/callback?code=abc&state=xyz", http.NoBody)
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != middleware.AdminLoginPath {
		t.Errorf("location = %q, want %q", loc, middleware.AdminLoginPath)
	}
}

func TestAdminCallbackStateMismatch(t *testing.T) {
	store := &mockAdminLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	req := httptest.NewRequest(http.MethodGet, "/admin/callback?code=abc&state=wrong", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  adminOAuthCookieName,
		Value: "correct.verifier123",
	})
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != middleware.AdminLoginPath {
		t.Errorf("location = %q, want %q", loc, middleware.AdminLoginPath)
	}
}

func TestAdminCallbackInvalidCookieFormat(t *testing.T) {
	store := &mockAdminLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	req := httptest.NewRequest(http.MethodGet, "/admin/callback?code=abc&state=test", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  adminOAuthCookieName,
		Value: "nodot",
	})
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminCallbackConsumeCodeError(t *testing.T) {
	store := &mockAdminLoginStore{
		authCodeErr: fmt.Errorf("db error"),
	}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	state := testState
	verifier := testAdminVerifier
	req := httptest.NewRequest(http.MethodGet, "/admin/callback?code=abc&state="+state, http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  adminOAuthCookieName,
		Value: state + "." + verifier,
	})
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminCallbackNilAuthCode(t *testing.T) {
	store := &mockAdminLoginStore{
		authCode: nil,
	}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	state := testState
	verifier := testAdminVerifier
	req := httptest.NewRequest(http.MethodGet, "/admin/callback?code=abc&state="+state, http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  adminOAuthCookieName,
		Value: state + "." + verifier,
	})
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminCallbackClientIDMismatch(t *testing.T) {
	store := &mockAdminLoginStore{
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      "wrong-client",
			UserID:        uuid.New(),
			OrgID:         uuid.New(),
			RedirectURI:   testIssuer + "/admin/callback",
			CodeChallenge: "test-challenge",
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
	}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	state := testState
	verifier := testAdminVerifier
	req := httptest.NewRequest(http.MethodGet, "/admin/callback?code=abc&state="+state, http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  adminOAuthCookieName,
		Value: state + "." + verifier,
	})
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminCallbackRedirectURIMismatch(t *testing.T) {
	store := &mockAdminLoginStore{
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      adminClientID,
			UserID:        uuid.New(),
			OrgID:         uuid.New(),
			RedirectURI:   "http://wrong.example.com/callback",
			CodeChallenge: "test-challenge",
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
	}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	state := testState
	verifier := testAdminVerifier
	req := httptest.NewRequest(http.MethodGet, "/admin/callback?code=abc&state="+state, http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  adminOAuthCookieName,
		Value: state + "." + verifier,
	})
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminCallbackPKCEFailure(t *testing.T) {
	// Create a valid challenge from a different verifier
	realVerifier := "real-verifier-that-doesnt-match-and-is-at-least-43-chars"
	challenge := oauth.ComputeS256Challenge(realVerifier)

	store := &mockAdminLoginStore{
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      adminClientID,
			UserID:        uuid.New(),
			OrgID:         uuid.New(),
			RedirectURI:   testIssuer + "/admin/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
	}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	state := testState
	wrongVerifier := "wrong-verifier-doesnt-match-and-is-at-least-43-chars"
	req := httptest.NewRequest(http.MethodGet, "/admin/callback?code=abc&state="+state, http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  adminOAuthCookieName,
		Value: state + "." + wrongVerifier,
	})
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminCallbackUserNotFound(t *testing.T) {
	verifier := testAdminPKCEVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	store := &mockAdminLoginStore{
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      adminClientID,
			UserID:        uuid.New(),
			OrgID:         uuid.New(),
			RedirectURI:   testIssuer + "/admin/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
		userByID: nil,
	}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	state := testState
	req := httptest.NewRequest(http.MethodGet, "/admin/callback?code=abc&state="+state, http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  adminOAuthCookieName,
		Value: state + "." + verifier,
	})
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminCallbackDisabledUser(t *testing.T) {
	user := newTestUser()
	user.Enabled = false

	verifier := testAdminPKCEVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	store := &mockAdminLoginStore{
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      adminClientID,
			UserID:        user.ID,
			OrgID:         user.OrgID,
			RedirectURI:   testIssuer + "/admin/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
		userByID: user,
	}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	state := testState
	req := httptest.NewRequest(http.MethodGet, "/admin/callback?code=abc&state="+state, http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  adminOAuthCookieName,
		Value: state + "." + verifier,
	})
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminCallbackSuccess(t *testing.T) {
	user := newTestUser()

	verifier := testAdminPKCEVerifier
	challenge := oauth.ComputeS256Challenge(verifier)

	store := &mockAdminLoginStore{
		authCode: &model.AuthorizationCode{
			ID:            uuid.New(),
			ClientID:      adminClientID,
			UserID:        user.ID,
			OrgID:         user.OrgID,
			RedirectURI:   testIssuer + "/admin/callback",
			CodeChallenge: challenge,
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		},
		userByID: user,
	}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	state := testState
	req := httptest.NewRequest(http.MethodGet, "/admin/callback?code=abc&state="+state, http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  adminOAuthCookieName,
		Value: state + "." + verifier,
	})
	w := httptest.NewRecorder()

	h.Callback(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != "/admin/" {
		t.Errorf("location = %q, want /admin/", loc)
	}

	// Verify session cookie was set
	cookies := w.Result().Cookies()
	var foundSession bool
	for _, c := range cookies {
		if c.Name == "rampart_admin_session" {
			foundSession = true
			if !c.HttpOnly {
				t.Error("session cookie should be HttpOnly")
			}
			break
		}
	}
	if !foundSession {
		t.Error("expected admin session cookie to be set")
	}
}

func TestAdminLogout(t *testing.T) {
	store := &mockAdminLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	req := httptest.NewRequest(http.MethodPost, "/admin/logout", http.NoBody)
	w := httptest.NewRecorder()

	h.Logout(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != middleware.AdminLoginPath {
		t.Errorf("location = %q, want %q", loc, middleware.AdminLoginPath)
	}
}

func TestAdminLogoutWithAuthenticatedUser(t *testing.T) {
	store := &mockAdminLoginStore{}
	sessions := &mockSessionStore{}
	h := newTestAdminLoginHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{
		UserID:            uuid.New(),
		OrgID:             uuid.New(),
		PreferredUsername: "admin",
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/logout", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Logout(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestSplitFirst(t *testing.T) {
	tests := []struct {
		input    string
		sep      byte
		expected []string
	}{
		{"a.b", '.', []string{"a", "b"}},
		{"a.b.c", '.', []string{"a", "b.c"}},
		{"nodot", '.', []string{"nodot"}},
		{"", '.', []string{""}},
		{".leading", '.', []string{"", "leading"}},
	}

	for _, tt := range tests {
		result := splitFirst(tt.input, tt.sep)
		if len(result) != len(tt.expected) {
			t.Errorf("splitFirst(%q, %q) = %v, want %v", tt.input, tt.sep, result, tt.expected)
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("splitFirst(%q, %q)[%d] = %q, want %q", tt.input, tt.sep, i, v, tt.expected[i])
			}
		}
	}
}

// contains checks if s contains substr (helper for test assertions).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
