package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/social"
)

type mockSocialStore struct {
	oauthClient      *model.OAuthClient
	oauthErr         error
	defaultOrgID     uuid.UUID
	defaultOrgErr    error
	emailUser        *model.User
	emailErr         error
	createdUser      *model.User
	createErr        error
	updateErr        error
	socialAccount    *model.SocialAccount
	socialAccountErr error
	createdSocial    *model.SocialAccount
	createSocialErr  error
	storedCode       bool
	storeErr         error
	orgSettings      *model.OrgSettings
	orgSettErr       error

	// capturedUser records the user passed to CreateUser for assertions.
	capturedUser *model.User
}

func (m *mockSocialStore) GetOAuthClient(_ context.Context, _ string) (*model.OAuthClient, error) {
	return m.oauthClient, m.oauthErr
}

func (m *mockSocialStore) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return m.defaultOrgID, m.defaultOrgErr
}

func (m *mockSocialStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return m.emailUser, m.emailErr
}

func (m *mockSocialStore) CreateUser(_ context.Context, u *model.User) (*model.User, error) {
	m.capturedUser = u
	if m.createdUser != nil {
		return m.createdUser, m.createErr
	}
	u.ID = uuid.New()
	return u, m.createErr
}

func (m *mockSocialStore) UpdateLastLoginAt(_ context.Context, _ uuid.UUID) error {
	return m.updateErr
}

func (m *mockSocialStore) CreateSocialAccount(_ context.Context, a *model.SocialAccount) (*model.SocialAccount, error) {
	if m.createdSocial != nil {
		return m.createdSocial, m.createSocialErr
	}
	a.ID = uuid.New()
	return a, m.createSocialErr
}

func (m *mockSocialStore) GetSocialAccount(_ context.Context, _, _ string) (*model.SocialAccount, error) {
	return m.socialAccount, m.socialAccountErr
}

func (m *mockSocialStore) StoreAuthorizationCode(_ context.Context, _, _ string, _, _ uuid.UUID, _, _, _, _ string, _ time.Time) error {
	m.storedCode = true
	return m.storeErr
}

func (m *mockSocialStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.orgSettings, m.orgSettErr
}

// ── stub methods to satisfy store.OrgReader ──

func (m *mockSocialStore) GetOrganizationByID(_ context.Context, _ uuid.UUID) (*model.Organization, error) {
	return nil, nil
}
func (m *mockSocialStore) GetOrganizationIDBySlug(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// ── stub methods to satisfy store.UserReader ──

func (m *mockSocialStore) GetUserByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockSocialStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockSocialStore) FindUserByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}

// ── stub methods to satisfy store.UserWriter ──

func (m *mockSocialStore) UpdateUser(_ context.Context, _ uuid.UUID, _ *model.UpdateUserRequest) (*model.User, error) {
	return nil, nil
}
func (m *mockSocialStore) DeleteUser(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockSocialStore) UpdatePassword(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockSocialStore) IncrementFailedLogins(_ context.Context, _ uuid.UUID, _ int, _ time.Duration) error {
	return nil
}
func (m *mockSocialStore) ResetFailedLogins(_ context.Context, _ uuid.UUID) error { return nil }

// ── stub methods to satisfy store.SocialAccountStore ──

func (m *mockSocialStore) GetSocialAccountsByUserID(_ context.Context, _ uuid.UUID) ([]*model.SocialAccount, error) {
	return nil, nil
}
func (m *mockSocialStore) UpdateSocialAccountTokens(_ context.Context, _ uuid.UUID, _, _ string, _ *time.Time) error {
	return nil
}
func (m *mockSocialStore) DeleteSocialAccount(_ context.Context, _ uuid.UUID) error { return nil }

// ── stub methods to satisfy store.AuthCodeStore ──

func (m *mockSocialStore) ConsumeAuthorizationCode(_ context.Context, _ string) (*model.AuthorizationCode, error) {
	return nil, nil
}
func (m *mockSocialStore) DeleteExpiredAuthorizationCodes(_ context.Context) (int64, error) {
	return 0, nil
}

// ── stub methods to satisfy store.OrgSettingsReadWriter ──

func (m *mockSocialStore) UpdateOrgSettings(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	return nil, nil
}

// mockProvider implements social.Provider for testing.
type mockProvider struct {
	name    string
	authURL string
}

func (p *mockProvider) Name() string { return p.name }

func (p *mockProvider) AuthURL(state, redirectURL string) string {
	return p.authURL + "?state=" + state + "&redirect_uri=" + redirectURL
}

func (p *mockProvider) Exchange(_ context.Context, _, _ string) (*social.UserInfo, error) {
	return &social.UserInfo{
		ProviderUserID: "provider-123",
		Email:          "test@example.com",
		EmailVerified:  true,
		GivenName:      "Test",
		FamilyName:     "User",
	}, nil
}

func newTestSocialRegistry(providers ...social.Provider) *social.Registry {
	reg := social.NewRegistry()
	for _, p := range providers {
		reg.Register(p)
	}
	return reg
}

func newSocialTestOAuthClient(orgID uuid.UUID) *model.OAuthClient {
	return &model.OAuthClient{
		ID:           "test-client",
		OrgID:        orgID,
		Name:         "Test Client",
		ClientType:   "public",
		RedirectURIs: []string{"http://localhost:3002/callback"},
	}
}

func TestSocialInitiateLoginMissingParams(t *testing.T) {
	store := &mockSocialStore{}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")

	// Create a chi router context with the provider param
	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}", h.InitiateLogin)

	// Missing client_id and redirect_uri
	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "client_id") {
		t.Error("expected error mentioning client_id")
	}
}

func TestSocialInitiateLoginUnknownProvider(t *testing.T) {
	store := &mockSocialStore{}
	reg := newTestSocialRegistry() // empty registry
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}", h.InitiateLogin)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/unknown?client_id=test&redirect_uri=http://localhost/cb&state=abc", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Unknown social provider") {
		t.Error("expected unknown provider error")
	}
}

func TestSocialInitiateLoginMissingState(t *testing.T) {
	store := &mockSocialStore{}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}", h.InitiateLogin)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google?client_id=test&redirect_uri=http://localhost/cb", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "state") {
		t.Error("expected error mentioning state")
	}
}

func TestSocialInitiateLoginRedirects(t *testing.T) {
	orgID := uuid.New()
	store := &mockSocialStore{
		oauthClient: newSocialTestOAuthClient(orgID),
	}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}", h.InitiateLogin)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google?client_id=test-client&redirect_uri=http://localhost:3002/callback&state=abc123&scope=openid&code_challenge=xyz", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if !strings.HasPrefix(location, "https://accounts.google.com/auth?") {
		t.Errorf("Location = %q, want redirect to Google auth", location)
	}

	// Check that the social cookie was set
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == socialCookieName {
			found = true
			if c.Path != socialCookiePath {
				t.Errorf("cookie path = %q, want %q", c.Path, socialCookiePath)
			}
			if !c.HttpOnly {
				t.Error("expected HttpOnly cookie")
			}
		}
	}
	if !found {
		t.Error("expected social login state cookie to be set")
	}
}

func TestSocialCallbackInvalidState(t *testing.T) {
	store := &mockSocialStore{}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}/callback", h.Callback)

	// Request with no cookie at all
	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google/callback?code=authcode&state=badstate", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "cookie") {
		t.Error("expected error mentioning missing cookie")
	}
}

func TestSocialCallbackTamperedCookie(t *testing.T) {
	store := &mockSocialStore{}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}/callback", h.Callback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google/callback?code=authcode&state=somestate", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  socialCookieName,
		Value: "dGFtcGVyZWQ.0000000000000000000000000000000000000000000000000000000000000000",
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Invalid") {
		t.Error("expected invalid state error")
	}
}

func TestSocialCallbackStateMismatch(t *testing.T) {
	store := &mockSocialStore{}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	hmacKey := []byte("test-hmac-key")
	h := NewSocialHandler(store, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")

	// Create a validly signed cookie with a specific provider state
	payload := socialCookiePayload{
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:3002/callback",
		Scope:         "openid",
		State:         "original-state",
		CodeChallenge: "xyz",
		ProviderState: "correct-provider-state",
	}
	cookieValue, err := h.signCookiePayload(&payload)
	if err != nil {
		t.Fatalf("failed to sign cookie: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}/callback", h.Callback)

	// Pass a different state than what's in the cookie
	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google/callback?code=authcode&state=wrong-state", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  socialCookieName,
		Value: cookieValue,
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Invalid social login state") {
		t.Error("expected state mismatch error")
	}
}

func TestSocialCallbackSuccessCreatesUser(t *testing.T) {
	orgID := uuid.New()
	store := &mockSocialStore{
		oauthClient:  newSocialTestOAuthClient(orgID),
		defaultOrgID: orgID,
		emailUser:    nil, // no existing user
	}
	hmacKey := []byte("test-hmac-key")
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")

	// Create a validly signed cookie
	payload := socialCookiePayload{
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:3002/callback",
		Scope:         "openid",
		State:         "original-state",
		CodeChallenge: "xyz",
		ProviderState: "provider-state-123",
	}
	cookieValue, err := h.signCookiePayload(&payload)
	if err != nil {
		t.Fatalf("failed to sign cookie: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}/callback", h.Callback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google/callback?code=authcode&state=provider-state-123", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  socialCookieName,
		Value: cookieValue,
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if !strings.HasPrefix(location, "http://localhost:3002/callback?code=") {
		t.Errorf("Location = %q, want redirect to client callback with code", location)
	}
	if !strings.Contains(location, "state=original-state") {
		t.Error("expected original state in redirect URL")
	}

	if !store.storedCode {
		t.Error("expected authorization code to be stored")
	}
}

func TestSocialCallbackMissingCode(t *testing.T) {
	store := &mockSocialStore{}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}/callback", h.Callback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google/callback?state=somestate", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Missing code") {
		t.Error("expected error mentioning missing code")
	}
}

func TestDeriveUsername(t *testing.T) {
	tests := []struct {
		email    string
		expected string
	}{
		{"user@example.com", "user"},
		{"hello.world@test.org", "hello.world"},
		{"nope", "nope"},
	}

	for _, tc := range tests {
		got := deriveUsername(tc.email)
		if got != tc.expected {
			t.Errorf("deriveUsername(%q) = %q, want %q", tc.email, got, tc.expected)
		}
	}
}

func TestCookieSignAndVerify(t *testing.T) {
	h := NewSocialHandler(nil, nil, noopLogger(), nil, []byte("secret-key"), "http://localhost:8080")

	payload := socialCookiePayload{
		ClientID:      "client-1",
		RedirectURI:   "http://localhost/cb",
		Scope:         "openid",
		State:         "state-abc",
		CodeChallenge: "challenge",
		ProviderState: "pstate",
	}

	signed, err := h.signCookiePayload(&payload)
	if err != nil {
		t.Fatalf("signCookiePayload failed: %v", err)
	}

	verified, err := h.verifyCookiePayload(signed)
	if err != nil {
		t.Fatalf("verifyCookiePayload failed: %v", err)
	}

	if verified.ClientID != payload.ClientID {
		t.Errorf("ClientID = %q, want %q", verified.ClientID, payload.ClientID)
	}
	if verified.ProviderState != payload.ProviderState {
		t.Errorf("ProviderState = %q, want %q", verified.ProviderState, payload.ProviderState)
	}
}

func TestCookieVerifyRejectsTampered(t *testing.T) {
	h := NewSocialHandler(nil, nil, noopLogger(), nil, []byte("secret-key"), "http://localhost:8080")

	_, err := h.verifyCookiePayload("tampered-value.badhex")
	if err == nil {
		t.Error("expected error for tampered cookie")
	}
}

func TestSocialInitiateLoginUnknownClient(t *testing.T) {
	store := &mockSocialStore{
		oauthClient: nil, // client not found
	}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}", h.InitiateLogin)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google?client_id=unknown&redirect_uri=http://localhost/cb&state=abc", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Unknown client_id") {
		t.Error("expected unknown client error")
	}
}

func TestSocialCallbackExistingSocialAccountLinksUser(t *testing.T) {
	orgID := uuid.New()
	existingUser := &model.User{
		ID:       uuid.New(),
		OrgID:    orgID,
		Username: "existinguser",
		Email:    "test@example.com",
		Enabled:  true,
	}
	store := &mockSocialStore{
		oauthClient:  newSocialTestOAuthClient(orgID),
		defaultOrgID: orgID,
		emailUser:    existingUser,
		socialAccount: &model.SocialAccount{
			ID:             uuid.New(),
			UserID:         existingUser.ID,
			Provider:       "google",
			ProviderUserID: "provider-123",
		},
	}
	hmacKey := []byte("test-hmac-key")
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")

	payload := socialCookiePayload{
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:3002/callback",
		Scope:         "openid",
		State:         "original-state",
		CodeChallenge: "xyz",
		ProviderState: "provider-state-existing",
	}
	cookieValue, err := h.signCookiePayload(&payload)
	if err != nil {
		t.Fatalf("failed to sign cookie: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}/callback", h.Callback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google/callback?code=authcode&state=provider-state-existing", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  socialCookieName,
		Value: cookieValue,
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusFound, w.Body.String())
	}

	location := w.Header().Get("Location")
	if !strings.HasPrefix(location, "http://localhost:3002/callback?code=") {
		t.Errorf("Location = %q, want redirect to client callback", location)
	}
}

func TestSocialCallbackStoreCodeError(t *testing.T) {
	orgID := uuid.New()
	store := &mockSocialStore{
		oauthClient:  newSocialTestOAuthClient(orgID),
		defaultOrgID: orgID,
		emailUser:    nil, // new user
		storeErr:     fmt.Errorf("db error storing code"),
	}
	hmacKey := []byte("test-hmac-key")
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")

	payload := socialCookiePayload{
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:3002/callback",
		Scope:         "openid",
		State:         "original-state",
		CodeChallenge: "xyz",
		ProviderState: "provider-state-err",
	}
	cookieValue, err := h.signCookiePayload(&payload)
	if err != nil {
		t.Fatalf("failed to sign cookie: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}/callback", h.Callback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google/callback?code=authcode&state=provider-state-err", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  socialCookieName,
		Value: cookieValue,
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestSocialInitiateClientFetchError(t *testing.T) {
	store := &mockSocialStore{
		oauthErr: fmt.Errorf("db error"),
	}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}", h.InitiateLogin)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google?client_id=test-client&redirect_uri=http://localhost:3002/callback&state=abc", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestCookieVerifyMalformedPayload(t *testing.T) {
	h := NewSocialHandler(nil, nil, noopLogger(), nil, []byte("secret-key"), "http://localhost:8080")

	// No dot separator
	_, err := h.verifyCookiePayload("nodotseparator")
	if err == nil {
		t.Error("expected error for cookie without dot separator")
	}

	// Invalid hex signature
	_, err = h.verifyCookiePayload("validbase64.invalidhex!!!")
	if err == nil {
		t.Error("expected error for invalid hex signature")
	}
}

func TestSocialCallbackNewUserHasNonEmptyPasswordHash(t *testing.T) {
	orgID := uuid.New()
	st := &mockSocialStore{
		oauthClient:  newSocialTestOAuthClient(orgID),
		defaultOrgID: orgID,
		emailUser:    nil, // no existing user — triggers user creation
	}
	hmacKey := []byte("test-hmac-key")
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(st, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")

	payload := socialCookiePayload{
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:3002/callback",
		Scope:         "openid",
		State:         "original-state",
		CodeChallenge: "xyz",
		ProviderState: "provider-state-pw",
	}
	cookieValue, err := h.signCookiePayload(&payload)
	if err != nil {
		t.Fatalf("failed to sign cookie: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}/callback", h.Callback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google/callback?code=authcode&state=provider-state-pw", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  socialCookieName,
		Value: cookieValue,
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusFound, w.Body.String())
	}

	// The user created via social login must have a non-empty password hash
	// so that password-based login is impossible for social-only accounts.
	if st.capturedUser == nil {
		t.Fatal("expected CreateUser to be called for new social user")
	}
	if len(st.capturedUser.PasswordHash) == 0 {
		t.Error("social-created user must have a non-empty PasswordHash to prevent password bypass")
	}
	if !strings.HasPrefix(string(st.capturedUser.PasswordHash), "$argon2id$") {
		t.Errorf("PasswordHash = %q, want argon2id-formatted hash", string(st.capturedUser.PasswordHash))
	}
}

// mockExchangeProvider returns configurable UserInfo from Exchange.
type mockExchangeProvider struct {
	name     string
	userInfo *social.UserInfo
	err      error
}

func (p *mockExchangeProvider) Name() string { return p.name }
func (p *mockExchangeProvider) AuthURL(state, redirectURL string) string {
	return "https://provider.example.com/auth?state=" + state
}
func (p *mockExchangeProvider) Exchange(_ context.Context, _, _ string) (*social.UserInfo, error) {
	return p.userInfo, p.err
}

func TestResolveUser_UnverifiedEmail_NoAutoLink(t *testing.T) {
	orgID := uuid.New()
	existingUser := &model.User{
		ID:       uuid.New(),
		OrgID:    orgID,
		Username: "victim",
		Email:    "victim@example.com",
		Enabled:  true,
	}

	store := &mockSocialStore{
		defaultOrgID: orgID,
		emailUser:    existingUser, // existing user with this email
	}

	reg := social.NewRegistry()
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")

	r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	// Attempt to resolve with an UNVERIFIED email that matches an existing user.
	userInfo := &social.UserInfo{
		ProviderUserID: "attacker-123",
		Email:          "victim@example.com",
		EmailVerified:  false,
		Name:           "Attacker",
	}

	_, _, err := h.resolveUser(r.Context(), r, "evil-provider", userInfo)
	if !errors.Is(err, errSocialEmailNotVerified) {
		t.Fatalf("expected errSocialEmailNotVerified, got: %v", err)
	}
}

func TestResolveUser_VerifiedEmail_AutoLinks(t *testing.T) {
	orgID := uuid.New()
	existingUser := &model.User{
		ID:       uuid.New(),
		OrgID:    orgID,
		Username: "alice",
		Email:    "alice@example.com",
		Enabled:  true,
	}

	store := &mockSocialStore{
		defaultOrgID: orgID,
		emailUser:    existingUser, // existing user with this email
	}

	reg := social.NewRegistry()
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")

	r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	userInfo := &social.UserInfo{
		ProviderUserID: "alice-google-123",
		Email:          "alice@example.com",
		EmailVerified:  true,
		Name:           "Alice",
	}

	user, returnedOrgID, err := h.resolveUser(r.Context(), r, "google", userInfo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != existingUser.ID {
		t.Fatalf("expected user ID %s, got %s", existingUser.ID, user.ID)
	}
	if returnedOrgID != orgID {
		t.Fatalf("expected org ID %s, got %s", orgID, returnedOrgID)
	}
}

func TestCallback_UnverifiedEmail_Returns403(t *testing.T) {
	orgID := uuid.New()
	existingUser := &model.User{
		ID:       uuid.New(),
		OrgID:    orgID,
		Username: "victim",
		Email:    "victim@example.com",
		Enabled:  true,
	}

	store := &mockSocialStore{
		oauthClient:  newSocialTestOAuthClient(orgID),
		defaultOrgID: orgID,
		emailUser:    existingUser,
	}

	provider := &mockExchangeProvider{
		name: "evil-provider",
		userInfo: &social.UserInfo{
			ProviderUserID: "attacker-456",
			Email:          "victim@example.com",
			EmailVerified:  false,
			Name:           "Attacker",
		},
	}

	reg := social.NewRegistry()
	reg.Register(provider)

	hmacKey := []byte("test-hmac-key")
	h := NewSocialHandler(store, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")

	// Build a valid signed cookie
	payload := socialCookiePayload{
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:3002/callback",
		Scope:         "openid",
		State:         "original-state",
		CodeChallenge: "challenge",
		ProviderState: "provider-state-abc",
	}
	cookieValue, err := h.signCookiePayload(&payload)
	if err != nil {
		t.Fatalf("failed to sign cookie: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}/callback", h.Callback)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/evil-provider/callback?code=authcode&state=provider-state-abc", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  socialCookieName,
		Value: cookieValue,
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d; body: %s", w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "not been verified") {
		t.Fatalf("expected body to mention email not verified, got: %s", w.Body.String())
	}
}

func TestSocialInitiateLoginInvalidRedirectURI(t *testing.T) {
	orgID := uuid.New()
	store := &mockSocialStore{
		oauthClient: newSocialTestOAuthClient(orgID),
	}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")

	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}", h.InitiateLogin)

	req := httptest.NewRequest(http.MethodGet, "/oauth/social/google?client_id=test-client&redirect_uri=http://evil.com/callback&state=abc", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Invalid redirect_uri") {
		t.Error("expected redirect_uri error")
	}
}
