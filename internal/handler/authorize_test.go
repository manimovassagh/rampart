package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/model"
)

type mockAuthorizeStore struct {
	oauthClient   *model.OAuthClient
	oauthErr      error
	defaultOrgID  uuid.UUID
	defaultOrgErr error
	emailUser     *model.User
	emailErr      error
	usernameUser  *model.User
	usernameErr   error
	storedCode    bool
	storeErr      error
	updateErr     error
	orgSettings   *model.OrgSettings
	orgSettErr    error
}

func (m *mockAuthorizeStore) GetOAuthClient(_ context.Context, _ string) (*model.OAuthClient, error) {
	return m.oauthClient, m.oauthErr
}

func (m *mockAuthorizeStore) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return m.defaultOrgID, m.defaultOrgErr
}

func (m *mockAuthorizeStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return m.emailUser, m.emailErr
}

func (m *mockAuthorizeStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return m.usernameUser, m.usernameErr
}

func (m *mockAuthorizeStore) StoreAuthorizationCode(_ context.Context, _, _ string, _, _ uuid.UUID, _, _, _ string, _ time.Time) error {
	m.storedCode = true
	return m.storeErr
}

func (m *mockAuthorizeStore) UpdateLastLoginAt(_ context.Context, _ uuid.UUID) error {
	return m.updateErr
}

func (m *mockAuthorizeStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.orgSettings, m.orgSettErr
}

func newTestOAuthClient(orgID uuid.UUID) *model.OAuthClient {
	return &model.OAuthClient{
		ID:           "test-client",
		OrgID:        orgID,
		Name:         "Test Client",
		ClientType:   "public",
		RedirectURIs: []string{"http://localhost:3002/callback"},
	}
}

func TestAuthorizeGetMissingParams(t *testing.T) {
	store := &mockAuthorizeStore{}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize", http.NoBody)
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "client_id") {
		t.Error("expected error mentioning client_id")
	}
}

func TestAuthorizeGetUnknownClient(t *testing.T) {
	store := &mockAuthorizeStore{oauthClient: nil}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=unknown&redirect_uri=http://evil.com/cb&response_type=code&state=abc&code_challenge=xyz&code_challenge_method=S256", http.NoBody)
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Unknown client_id") {
		t.Error("expected unknown client error")
	}
}

func TestAuthorizeGetInvalidRedirectURI(t *testing.T) {
	orgID := uuid.New()
	store := &mockAuthorizeStore{
		oauthClient: newTestOAuthClient(orgID),
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=test-client&redirect_uri=http://evil.com/cb&response_type=code&state=abc&code_challenge=xyz&code_challenge_method=S256", http.NoBody)
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Invalid redirect_uri") {
		t.Error("expected redirect_uri error")
	}
}

func TestAuthorizeGetMissingPKCE(t *testing.T) {
	orgID := uuid.New()
	store := &mockAuthorizeStore{
		oauthClient: newTestOAuthClient(orgID),
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=test-client&redirect_uri=http://localhost:3002/callback&response_type=code&state=abc", http.NoBody)
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "PKCE") {
		t.Error("expected PKCE error")
	}
}

func TestAuthorizeGetRendersLoginPage(t *testing.T) {
	orgID := uuid.New()
	store := &mockAuthorizeStore{
		oauthClient: newTestOAuthClient(orgID),
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=test-client&redirect_uri=http://localhost:3002/callback&response_type=code&state=abc123&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&code_challenge_method=S256", http.NoBody)
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test Client") {
		t.Error("expected client name in login page")
	}
	if !strings.Contains(body, "test-client") {
		t.Error("expected client_id in hidden field")
	}
	if !strings.Contains(body, "abc123") {
		t.Error("expected state in hidden field")
	}
}

func TestAuthorizePostBadCredentials(t *testing.T) {
	orgID := uuid.New()
	store := &mockAuthorizeStore{
		oauthClient:  newTestOAuthClient(orgID),
		emailUser:    nil,
		usernameUser: nil,
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	form := url.Values{
		"client_id":             {"test-client"},
		"redirect_uri":          {"http://localhost:3002/callback"},
		"response_type":         {"code"},
		"scope":                 {"openid"},
		"state":                 {"abc"},
		"code_challenge":        {"xyz"},
		"code_challenge_method": {"S256"},
		"identifier":            {"baduser"},
		"password":              {"badpass"},
	}

	req := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (re-render login)", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "Invalid username/email or password") {
		t.Error("expected error message in re-rendered page")
	}
}

func TestAuthorizePostValidCredentialsRedirects(t *testing.T) {
	orgID := uuid.New()
	hash, _ := auth.HashPassword("Str0ng!Pass")
	user := &model.User{
		ID:           uuid.New(),
		OrgID:        orgID,
		Username:     "admin",
		Email:        "admin@test.com",
		PasswordHash: []byte(hash),
		Enabled:      true,
	}

	store := &mockAuthorizeStore{
		oauthClient: newTestOAuthClient(orgID),
		emailUser:   user,
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	form := url.Values{
		"client_id":             {"test-client"},
		"redirect_uri":          {"http://localhost:3002/callback"},
		"response_type":         {"code"},
		"scope":                 {"openid"},
		"state":                 {"mystate123"},
		"code_challenge":        {"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"},
		"code_challenge_method": {"S256"},
		"identifier":            {"admin@test.com"},
		"password":              {"Str0ng!Pass"},
	}

	req := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if !strings.HasPrefix(location, "http://localhost:3002/callback?code=") {
		t.Errorf("Location = %q, want redirect to callback with code", location)
	}
	if !strings.Contains(location, "state=mystate123") {
		t.Error("expected state in redirect URL")
	}

	if !store.storedCode {
		t.Error("expected authorization code to be stored")
	}
}

func TestAuthorizeGetMissingState(t *testing.T) {
	orgID := uuid.New()
	store := &mockAuthorizeStore{
		oauthClient: newTestOAuthClient(orgID),
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=test-client&redirect_uri=http://localhost:3002/callback&response_type=code&code_challenge=xyz&code_challenge_method=S256", http.NoBody)
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "state") {
		t.Error("expected error mentioning state")
	}
}
