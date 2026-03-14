package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/social"
)

// failingProvider implements social.Provider but returns an error on Exchange.
type failingProvider struct {
	name    string
	authURL string
}

func (p *failingProvider) Name() string { return p.name }

func (p *failingProvider) AuthURL(state, redirectURL string) string {
	return p.authURL + "?state=" + state + "&redirect_uri=" + redirectURL
}

func (p *failingProvider) Exchange(_ context.Context, _, _ string) (*social.UserInfo, error) {
	return nil, fmt.Errorf("exchange failed: token endpoint unreachable")
}

// noEmailProvider implements social.Provider but returns empty email.
type noEmailProvider struct {
	name    string
	authURL string
}

func (p *noEmailProvider) Name() string { return p.name }

func (p *noEmailProvider) AuthURL(state, redirectURL string) string {
	return p.authURL + "?state=" + state + "&redirect_uri=" + redirectURL
}

func (p *noEmailProvider) Exchange(_ context.Context, _, _ string) (*social.UserInfo, error) {
	return &social.UserInfo{
		ProviderUserID: "provider-no-email",
		Email:          "",
	}, nil
}

func newIntegrationRouter(h *SocialHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/oauth/social/{provider}", h.InitiateLogin)
	r.Get("/oauth/social/{provider}/callback", h.Callback)
	return r
}

func TestSocialInitiateRedirectsToProviderWithCorrectParams(t *testing.T) {
	orgID := uuid.New()
	store := &mockSocialStore{
		oauthClient: newSocialTestOAuthClient(orgID),
	}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/o/oauth2/v2/auth"})
	hmacKey := []byte("integration-test-hmac-key")
	h := NewSocialHandler(store, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")
	router := newIntegrationRouter(h)

	req := httptest.NewRequest(http.MethodGet,
		"/oauth/social/google?client_id=test-client&redirect_uri=http://localhost:3002/callback&state=mystate&scope=openid+email&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if !strings.HasPrefix(location, "https://accounts.google.com/o/oauth2/v2/auth?") {
		t.Errorf("Location = %q, want prefix https://accounts.google.com/o/oauth2/v2/auth?", location)
	}

	// The redirect must include a state parameter (provider state, not the client state)
	if !strings.Contains(location, "state=") {
		t.Error("expected state parameter in redirect URL")
	}

	// The redirect must include the callback redirect_uri pointing back to Rampart
	if !strings.Contains(location, "redirect_uri=http://localhost:8080/oauth/social/google/callback") {
		t.Errorf("expected callback redirect_uri in Location, got %q", location)
	}

	// Verify the signed cookie is set
	cookies := w.Result().Cookies()
	var socialCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == socialCookieName {
			socialCookie = c
			break
		}
	}
	if socialCookie == nil {
		t.Fatal("expected social login state cookie to be set")
	}
	if !socialCookie.HttpOnly {
		t.Error("cookie must be HttpOnly")
	}
	if !socialCookie.Secure {
		t.Error("cookie must be Secure")
	}
	if socialCookie.SameSite != http.SameSiteNoneMode {
		t.Errorf("cookie SameSite = %d, want None (%d)", socialCookie.SameSite, http.SameSiteNoneMode)
	}
	if socialCookie.Path != socialCookiePath {
		t.Errorf("cookie Path = %q, want %q", socialCookie.Path, socialCookiePath)
	}

	// Verify the cookie payload can be decoded and contains correct data
	payload, err := h.verifyCookiePayload(socialCookie.Value)
	if err != nil {
		t.Fatalf("failed to verify cookie payload: %v", err)
	}
	if payload.ClientID != "test-client" {
		t.Errorf("cookie ClientID = %q, want test-client", payload.ClientID)
	}
	if payload.RedirectURI != "http://localhost:3002/callback" {
		t.Errorf("cookie RedirectURI = %q, want http://localhost:3002/callback", payload.RedirectURI)
	}
	if payload.State != "mystate" {
		t.Errorf("cookie State = %q, want mystate", payload.State)
	}
	if payload.CodeChallenge != "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" {
		t.Errorf("cookie CodeChallenge = %q, want E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", payload.CodeChallenge)
	}
	if payload.ProviderState == "" {
		t.Error("cookie ProviderState must not be empty")
	}
}

func TestSocialInitiateUnknownProviderTableDriven(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantBody string
	}{
		{"unknown provider", "unknown", "Unknown social provider"},
		{"nonexistent provider", "nonexistent", "Unknown social provider"},
		{"empty name via URL", "doesnotexist", "Unknown social provider"},
	}

	store := &mockSocialStore{}
	// Register only google so others are unknown
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")
	router := newIntegrationRouter(h)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/oauth/social/"+tc.provider+"?client_id=test&redirect_uri=http://localhost/cb&state=abc",
				http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
			if !strings.Contains(w.Body.String(), tc.wantBody) {
				t.Errorf("body = %q, want substring %q", w.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestSocialInitiateMissingParamsTableDriven(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantBody string
	}{
		{"no params at all", "", "client_id"},
		{"missing redirect uri", "client_id=test", "redirect_uri"},
		{"missing client id", "redirect_uri=http://localhost/cb", "client_id"},
		{"missing state", "client_id=test&redirect_uri=http://localhost/cb", "state"},
	}

	store := &mockSocialStore{}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")
	router := newIntegrationRouter(h)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := "/oauth/social/google"
			if tc.query != "" {
				path += "?" + tc.query
			}
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
			if !strings.Contains(w.Body.String(), tc.wantBody) {
				t.Errorf("body = %q, want substring %q", w.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestSocialCallbackInvalidStateTableDriven(t *testing.T) {
	hmacKey := []byte("test-hmac-key")
	store := &mockSocialStore{}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")
	router := newIntegrationRouter(h)

	// Create a valid signed cookie with a known provider state
	payload := socialCookiePayload{
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:3002/callback",
		Scope:         "openid",
		State:         "original-state",
		CodeChallenge: "xyz",
		ProviderState: "valid-provider-state",
	}
	validCookie, err := h.signCookiePayload(&payload)
	if err != nil {
		t.Fatalf("failed to sign cookie: %v", err)
	}

	tests := []struct {
		name       string
		query      string
		cookie     *http.Cookie
		wantStatus int
		wantBody   string
	}{
		{
			name:       "no cookie present",
			query:      "code=xxx&state=invalid",
			cookie:     nil,
			wantStatus: http.StatusBadRequest,
			wantBody:   "cookie",
		},
		{
			name:  "tampered cookie signature",
			query: "code=xxx&state=somestate",
			cookie: &http.Cookie{
				Name:  socialCookieName,
				Value: "dGFtcGVyZWQ.0000000000000000000000000000000000000000000000000000000000000000",
			},
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid",
		},
		{
			name:  "valid cookie but wrong state",
			query: "code=xxx&state=wrong-state",
			cookie: &http.Cookie{
				Name:  socialCookieName,
				Value: validCookie,
			},
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid social login state",
		},
		{
			name:  "malformed cookie value no dot separator",
			query: "code=xxx&state=somestate",
			cookie: &http.Cookie{
				Name:  socialCookieName,
				Value: "nodotinthisvalue",
			},
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/oauth/social/google/callback?"+tc.query, http.NoBody)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
			if !strings.Contains(w.Body.String(), tc.wantBody) {
				t.Errorf("body = %q, want substring %q", w.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestSocialCallbackNoProvider(t *testing.T) {
	store := &mockSocialStore{}
	// Register only google
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")
	router := newIntegrationRouter(h)

	tests := []struct {
		name     string
		provider string
	}{
		{"nonexistent provider", "nonexistent"},
		{"github not registered", "github"},
		{"apple not registered", "apple"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/oauth/social/"+tc.provider+"/callback?code=authcode&state=somestate",
				http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
			if !strings.Contains(w.Body.String(), "Unknown social provider") {
				t.Errorf("body = %q, want substring 'Unknown social provider'", w.Body.String())
			}
		})
	}
}

func TestSocialCallbackMissingCodeOrState(t *testing.T) {
	store := &mockSocialStore{}
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, []byte("test-hmac-key"), "http://localhost:8080")
	router := newIntegrationRouter(h)

	tests := []struct {
		name  string
		query string
	}{
		{"missing code", "state=somestate"},
		{"missing state", "code=authcode"},
		{"both missing", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := "/oauth/social/google/callback"
			if tc.query != "" {
				path += "?" + tc.query
			}
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
			if !strings.Contains(w.Body.String(), "Missing code or state") {
				t.Errorf("body = %q, want substring 'Missing code or state'", w.Body.String())
			}
		})
	}
}

func TestSocialCallbackProviderExchangeFails(t *testing.T) {
	orgID := uuid.New()
	store := &mockSocialStore{
		oauthClient:  newSocialTestOAuthClient(orgID),
		defaultOrgID: orgID,
	}
	hmacKey := []byte("test-hmac-key")
	reg := newTestSocialRegistry(&failingProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")
	router := newIntegrationRouter(h)

	payload := socialCookiePayload{
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:3002/callback",
		Scope:         "openid",
		State:         "original-state",
		CodeChallenge: "xyz",
		ProviderState: "provider-state-456",
	}
	cookieValue, err := h.signCookiePayload(&payload)
	if err != nil {
		t.Fatalf("failed to sign cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/oauth/social/google/callback?code=badcode&state=provider-state-456",
		http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  socialCookieName,
		Value: cookieValue,
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadGateway)
	}
	if !strings.Contains(w.Body.String(), "Failed to authenticate") {
		t.Errorf("body = %q, want substring 'Failed to authenticate'", w.Body.String())
	}
}

func TestSocialCallbackProviderReturnsNoEmail(t *testing.T) {
	orgID := uuid.New()
	store := &mockSocialStore{
		oauthClient:  newSocialTestOAuthClient(orgID),
		defaultOrgID: orgID,
	}
	hmacKey := []byte("test-hmac-key")
	reg := newTestSocialRegistry(&noEmailProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")
	router := newIntegrationRouter(h)

	payload := socialCookiePayload{
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:3002/callback",
		Scope:         "openid",
		State:         "original-state",
		CodeChallenge: "xyz",
		ProviderState: "provider-state-789",
	}
	cookieValue, err := h.signCookiePayload(&payload)
	if err != nil {
		t.Fatalf("failed to sign cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/oauth/social/google/callback?code=validcode&state=provider-state-789",
		http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  socialCookieName,
		Value: cookieValue,
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "email address") {
		t.Errorf("body = %q, want substring 'email address'", w.Body.String())
	}
}

func TestSocialCallbackSuccessLinksExistingUser(t *testing.T) {
	orgID := uuid.New()
	existingUser := &model.User{
		ID:       uuid.New(),
		OrgID:    orgID,
		Username: "existinguser",
		Email:    "test@example.com",
		Enabled:  true,
	}
	store := &mockSocialStore{
		oauthClient:   newSocialTestOAuthClient(orgID),
		defaultOrgID:  orgID,
		emailUser:     existingUser, // user already exists with this email
		socialAccount: nil,          // no social account linked yet
	}
	hmacKey := []byte("test-hmac-key")
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")
	router := newIntegrationRouter(h)

	payload := socialCookiePayload{
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:3002/callback",
		Scope:         "openid",
		State:         "client-state-abc",
		CodeChallenge: "challengeXYZ",
		ProviderState: "provider-state-link",
	}
	cookieValue, err := h.signCookiePayload(&payload)
	if err != nil {
		t.Fatalf("failed to sign cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/oauth/social/google/callback?code=validcode&state=provider-state-link",
		http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  socialCookieName,
		Value: cookieValue,
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusFound, w.Body.String())
	}

	location := w.Header().Get("Location")
	if !strings.HasPrefix(location, "http://localhost:3002/callback?code=") {
		t.Errorf("Location = %q, want redirect to client callback with code", location)
	}
	if !strings.Contains(location, "state=client-state-abc") {
		t.Errorf("expected original client state in redirect, got %q", location)
	}
	if !store.storedCode {
		t.Error("expected authorization code to be stored")
	}
}

func TestSocialCallbackCookieClearedAfterCallback(t *testing.T) {
	orgID := uuid.New()
	store := &mockSocialStore{
		oauthClient:  newSocialTestOAuthClient(orgID),
		defaultOrgID: orgID,
	}
	hmacKey := []byte("test-hmac-key")
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")
	router := newIntegrationRouter(h)

	payload := socialCookiePayload{
		ClientID:      "test-client",
		RedirectURI:   "http://localhost:3002/callback",
		Scope:         "openid",
		State:         "state-clear",
		CodeChallenge: "challenge",
		ProviderState: "pstate-clear",
	}
	cookieValue, err := h.signCookiePayload(&payload)
	if err != nil {
		t.Fatalf("failed to sign cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/oauth/social/google/callback?code=validcode&state=pstate-clear",
		http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  socialCookieName,
		Value: cookieValue,
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}

	// Verify the social cookie is cleared (MaxAge = -1)
	for _, c := range w.Result().Cookies() {
		if c.Name == socialCookieName {
			if c.MaxAge != -1 {
				t.Errorf("cookie MaxAge = %d, want -1 (deleted)", c.MaxAge)
			}
			if c.Value != "" {
				t.Errorf("cookie Value = %q, want empty", c.Value)
			}
			return
		}
	}
	t.Error("expected social cookie to be set (with MaxAge=-1 for deletion)")
}

func TestSocialInitiateDefaultScopeOpenID(t *testing.T) {
	orgID := uuid.New()
	store := &mockSocialStore{
		oauthClient: newSocialTestOAuthClient(orgID),
	}
	hmacKey := []byte("test-hmac-key")
	reg := newTestSocialRegistry(&mockProvider{name: "google", authURL: "https://accounts.google.com/auth"})
	h := NewSocialHandler(store, reg, noopLogger(), nil, hmacKey, "http://localhost:8080")
	router := newIntegrationRouter(h)

	// No scope parameter provided
	req := httptest.NewRequest(http.MethodGet,
		"/oauth/social/google?client_id=test-client&redirect_uri=http://localhost:3002/callback&state=abc",
		http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}

	// Verify the cookie contains default scope "openid"
	for _, c := range w.Result().Cookies() {
		if c.Name == socialCookieName {
			payload, err := h.verifyCookiePayload(c.Value)
			if err != nil {
				t.Fatalf("failed to verify cookie: %v", err)
			}
			if payload.Scope != "openid" {
				t.Errorf("scope = %q, want 'openid' as default", payload.Scope)
			}
			return
		}
	}
	t.Error("expected social cookie to be set")
}
