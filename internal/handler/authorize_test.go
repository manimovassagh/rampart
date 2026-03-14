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
	"github.com/manimovassagh/rampart/internal/middleware"
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

func (m *mockAuthorizeStore) StoreAuthorizationCode(_ context.Context, _, _ string, _, _ uuid.UUID, _, _, _, _ string, _ time.Time) error {
	m.storedCode = true
	return m.storeErr
}

func (m *mockAuthorizeStore) UpdateLastLoginAt(_ context.Context, _ uuid.UUID) error {
	return m.updateErr
}

func (m *mockAuthorizeStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.orgSettings, m.orgSettErr
}

func (m *mockAuthorizeStore) IncrementFailedLogins(_ context.Context, _ uuid.UUID, _ int, _ time.Duration) error {
	return nil
}

func (m *mockAuthorizeStore) ResetFailedLogins(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockAuthorizeStore) GetUserByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return m.emailUser, m.emailErr
}

func (m *mockAuthorizeStore) HasConsent(_ context.Context, _ uuid.UUID, _, _ string) (bool, error) {
	return true, nil // default to consented for existing tests
}

func (m *mockAuthorizeStore) GrantConsent(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}

// ── stub methods to satisfy store.OrgReader ──

func (m *mockAuthorizeStore) GetOrganizationByID(_ context.Context, _ uuid.UUID) (*model.Organization, error) {
	return nil, nil
}
func (m *mockAuthorizeStore) GetOrganizationIDBySlug(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// ── stub methods to satisfy store.UserReader ──

func (m *mockAuthorizeStore) FindUserByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}

// ── stub methods to satisfy store.UserWriter ──

func (m *mockAuthorizeStore) CreateUser(_ context.Context, _ *model.User) (*model.User, error) {
	return nil, nil
}
func (m *mockAuthorizeStore) UpdateUser(_ context.Context, _, _ uuid.UUID, _ *model.UpdateUserRequest) (*model.User, error) {
	return nil, nil
}
func (m *mockAuthorizeStore) DeleteUser(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockAuthorizeStore) UpdatePassword(_ context.Context, _, _ uuid.UUID, _ []byte) error {
	return nil
}

// ── stub methods to satisfy store.AuthCodeStore ──

func (m *mockAuthorizeStore) ConsumeAuthorizationCode(_ context.Context, _ string) (*model.AuthorizationCode, error) {
	return nil, nil
}
func (m *mockAuthorizeStore) DeleteExpiredAuthorizationCodes(_ context.Context) (int64, error) {
	return 0, nil
}

// ── stub methods to satisfy store.OrgSettingsReadWriter ──

func (m *mockAuthorizeStore) UpdateOrgSettings(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	return nil, nil
}

func newTestOAuthClient(orgID uuid.UUID) *model.OAuthClient {
	return &model.OAuthClient{
		ID:           "test-client",
		OrgID:        orgID,
		Name:         "Test Client",
		ClientType:   "public",
		RedirectURIs: []string{"http://localhost:3002/callback"},
		Enabled:      true,
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

func TestAuthorizeGetDisabledClient(t *testing.T) {
	orgID := uuid.New()
	disabledClient := newTestOAuthClient(orgID)
	disabledClient.Enabled = false
	store := &mockAuthorizeStore{
		oauthClient: disabledClient,
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=test-client&redirect_uri=http://localhost:3002/callback&response_type=code&state=abc&code_challenge=xyz&code_challenge_method=S256", http.NoBody)
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if !strings.Contains(w.Body.String(), "disabled") {
		t.Error("expected error mentioning disabled client")
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

// newPostRequestWithCSRF creates a POST request with a matching CSRF cookie and form token.
func newPostRequestWithCSRF(t *testing.T, target string, form url.Values) *http.Request {
	t.Helper()
	csrfToken, err := middleware.GenerateCSRFToken()
	if err != nil {
		t.Fatalf("failed to generate CSRF token: %v", err)
	}
	form.Set("csrf_token", csrfToken)
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: middleware.OAuthCSRFCookieName, Value: csrfToken})
	return req
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

	req := newPostRequestWithCSRF(t, "/oauth/authorize", form)
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

	req := newPostRequestWithCSRF(t, "/oauth/authorize", form)
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

func TestAuthorizeGetRendersCSRFToken(t *testing.T) {
	orgID := uuid.New()
	store := &mockAuthorizeStore{
		oauthClient: newTestOAuthClient(orgID),
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=test-client&redirect_uri=http://localhost:3002/callback&response_type=code&state=abc123&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&code_challenge_method=S256", http.NoBody)
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "csrf_token") {
		t.Error("expected csrf_token hidden field in login page")
	}

	// Verify the CSRF cookie is set
	cookies := w.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == middleware.OAuthCSRFCookieName {
			csrfCookie = c
			break
		}
	}
	if csrfCookie == nil {
		t.Fatal("expected OAuth CSRF cookie to be set")
	}
	if !csrfCookie.HttpOnly {
		t.Error("expected CSRF cookie to be HttpOnly")
	}
	if csrfCookie.Value == "" {
		t.Error("expected non-empty CSRF cookie value")
	}
}

func TestAuthorizePostMissingCSRFToken(t *testing.T) {
	orgID := uuid.New()
	store := &mockAuthorizeStore{
		oauthClient: newTestOAuthClient(orgID),
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	form := url.Values{
		"client_id":             {"test-client"},
		"redirect_uri":          {"http://localhost:3002/callback"},
		"scope":                 {"openid"},
		"state":                 {"abc"},
		"code_challenge":        {"xyz"},
		"code_challenge_method": {"S256"},
		"identifier":            {"admin"},
		"password":              {"secret"},
	}

	// POST without CSRF cookie or token
	req := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if !strings.Contains(w.Body.String(), "CSRF") {
		t.Error("expected CSRF error message")
	}
}

func TestAuthorizePostMismatchedCSRFToken(t *testing.T) {
	orgID := uuid.New()
	store := &mockAuthorizeStore{
		oauthClient: newTestOAuthClient(orgID),
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	form := url.Values{
		"client_id":             {"test-client"},
		"redirect_uri":          {"http://localhost:3002/callback"},
		"scope":                 {"openid"},
		"state":                 {"abc"},
		"code_challenge":        {"xyz"},
		"code_challenge_method": {"S256"},
		"identifier":            {"admin"},
		"password":              {"secret"},
		"csrf_token":            {"wrong-token"},
	}

	req := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: middleware.OAuthCSRFCookieName, Value: "correct-token"})
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestAuthorizePostReRenderIncludesCSRFToken(t *testing.T) {
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
		"scope":                 {"openid"},
		"state":                 {"abc"},
		"code_challenge":        {"xyz"},
		"code_challenge_method": {"S256"},
		"identifier":            {"baduser"},
		"password":              {"badpass"},
	}

	req := newPostRequestWithCSRF(t, "/oauth/authorize", form)
	w := httptest.NewRecorder()

	h.Authorize(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// The re-rendered page should contain a fresh CSRF token
	body := w.Body.String()
	if !strings.Contains(body, "csrf_token") {
		t.Error("expected csrf_token in re-rendered login page")
	}

	// Check that a new CSRF cookie was set
	cookies := w.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == middleware.OAuthCSRFCookieName {
			csrfCookie = c
			break
		}
	}
	if csrfCookie == nil {
		t.Fatal("expected new OAuth CSRF cookie on re-render")
	}
	if csrfCookie.Value == "" {
		t.Error("expected non-empty CSRF cookie value on re-render")
	}
}

func TestConsentRejectsWithoutConsentCookie(t *testing.T) {
	orgID := uuid.New()
	store := &mockAuthorizeStore{
		oauthClient: newTestOAuthClient(orgID),
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	form := url.Values{
		"client_id":    {"test-client"},
		"redirect_uri": {"http://localhost:3002/callback"},
		"scope":        {"openid"},
		"state":        {"abc"},
		"consent":      {"allow"},
	}

	req := newPostRequestWithCSRF(t, "/oauth/consent", form)
	w := httptest.NewRecorder()

	h.Consent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Invalid or expired consent session") {
		t.Error("expected consent session error when no cookie is present")
	}
}

func TestConsentRejectsForgedUserID(t *testing.T) {
	orgID := uuid.New()
	store := &mockAuthorizeStore{
		oauthClient: newTestOAuthClient(orgID),
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	form := url.Values{
		"client_id":    {"test-client"},
		"redirect_uri": {"http://localhost:3002/callback"},
		"scope":        {"openid"},
		"state":        {"abc"},
		"consent":      {"allow"},
		"user_id":      {uuid.New().String()}, // attacker tries to inject user_id via form
	}

	req := newPostRequestWithCSRF(t, "/oauth/consent", form)
	w := httptest.NewRecorder()

	h.Consent(w, req)

	// Without the consent cookie, the handler must reject — even with user_id in form
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Invalid or expired consent session") {
		t.Error("expected consent session error; form-based user_id must be ignored")
	}
}

func TestConsentAcceptsValidCookie(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	hash, _ := auth.HashPassword("Str0ng!Pass")
	user := &model.User{
		ID:           userID,
		OrgID:        orgID,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: []byte(hash),
		Enabled:      true,
	}
	store := &mockAuthorizeStore{
		oauthClient: newTestOAuthClient(orgID),
		emailUser:   user,
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	form := url.Values{
		"client_id":      {"test-client"},
		"redirect_uri":   {"http://localhost:3002/callback"},
		"scope":          {"openid"},
		"state":          {"mystate"},
		"code_challenge": {"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"},
		"consent":        {"allow"},
	}

	req := newPostRequestWithCSRF(t, "/oauth/consent", form)
	// Add the consent user cookie (simulating server-side flow)
	req.AddCookie(&http.Cookie{Name: "rampart_consent_uid", Value: userID.String()})
	w := httptest.NewRecorder()

	h.Consent(w, req)

	// Should redirect to callback with an authorization code
	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
	location := w.Header().Get("Location")
	if !strings.HasPrefix(location, "http://localhost:3002/callback?code=") {
		t.Errorf("Location = %q, want redirect to callback with code", location)
	}
	if !strings.Contains(location, "state=mystate") {
		t.Error("expected state in redirect URL")
	}
}

func TestConsentDenialWithInvalidRedirectURI(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	store := &mockAuthorizeStore{
		oauthClient: newTestOAuthClient(orgID),
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	form := url.Values{
		"client_id":    {"test-client"},
		"redirect_uri": {"http://evil.example.com/phish"}, // not in registered URIs
		"scope":        {"openid"},
		"state":        {"abc"},
		"consent":      {"deny"},
	}

	req := newPostRequestWithCSRF(t, "/oauth/consent", form)
	req.AddCookie(&http.Cookie{Name: "rampart_consent_uid", Value: userID.String()})
	w := httptest.NewRecorder()

	h.Consent(w, req)

	// Must NOT redirect to the unvalidated URI — should render an error page instead
	if w.Code == http.StatusFound {
		t.Errorf("expected error page, got redirect to %s", w.Header().Get("Location"))
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Invalid redirect_uri") {
		t.Error("expected 'Invalid redirect_uri' error message")
	}
}

func TestConsentDenialWithUnknownClient(t *testing.T) {
	store := &mockAuthorizeStore{
		oauthClient: nil, // client not found
	}
	h := NewAuthorizeHandler(store, noopLogger(), nil, nil)

	form := url.Values{
		"client_id":    {"unknown-client"},
		"redirect_uri": {"http://evil.example.com/phish"},
		"scope":        {"openid"},
		"state":        {"abc"},
		"consent":      {"deny"},
	}

	req := newPostRequestWithCSRF(t, "/oauth/consent", form)
	req.AddCookie(&http.Cookie{Name: "rampart_consent_uid", Value: uuid.New().String()})
	w := httptest.NewRecorder()

	h.Consent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Unknown client") {
		t.Error("expected 'Unknown client' error message")
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
