package social

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	schemeHTTPS      = "https"
	testKID          = "test-key-1"
	testClientID     = "com.example.test"
	testClientSecret = "mock-secret"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newJWKSServer(t *testing.T, privateKey *rsa.PrivateKey, kid string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
}

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	return key
}

func signToken(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}
	return signed
}

func validAppleClaims(clientID string) jwt.MapClaims { //nolint:unparam // parameter for test flexibility
	now := time.Now()
	return jwt.MapClaims{
		"iss":            appleIssuer,
		"aud":            clientID,
		"sub":            "user-123",
		"email":          "user@example.com",
		"email_verified": true,
		"exp":            now.Add(time.Hour).Unix(),
		"iat":            now.Unix(),
	}
}

// roundTripFunc allows using a function as http.RoundTripper.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// ---------------------------------------------------------------------------
// Registry tests
// ---------------------------------------------------------------------------

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	google := &GoogleProvider{ClientID: "google-id", ClientSecret: "google-secret"}
	github := &GitHubProvider{ClientID: "github-id", ClientSecret: "github-secret"}

	reg.Register(google)
	reg.Register(github)

	got, ok := reg.Get("google")
	if !ok {
		t.Fatal("expected google provider to be registered")
	}
	if got.Name() != "google" {
		t.Errorf("expected provider name 'google', got %q", got.Name())
	}

	got, ok = reg.Get("github")
	if !ok {
		t.Fatal("expected github provider to be registered")
	}
	if got.Name() != "github" {
		t.Errorf("expected provider name 'github', got %q", got.Name())
	}

	_, ok = reg.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent provider to not be found")
	}
}

func TestRegistryNames(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&GoogleProvider{ClientID: "g"})
	reg.Register(&GitHubProvider{ClientID: "gh"})
	reg.Register(&AppleProvider{ClientID: "a"})

	names := reg.Names()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}

	expected := []string{"apple", "github", "google"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected names[%d] = %q, got %q", i, expected[i], name)
		}
	}
}

func TestRegistryEmpty(t *testing.T) {
	reg := NewRegistry()
	names := reg.Names()
	if len(names) != 0 {
		t.Errorf("expected empty names, got %v", names)
	}

	_, ok := reg.Get("anything")
	if ok {
		t.Error("expected Get on empty registry to return false")
	}
}

func TestRegistryUnregister(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&GoogleProvider{ClientID: "g"})
	reg.Unregister("google")

	_, ok := reg.Get("google")
	if ok {
		t.Error("expected google to be unregistered")
	}
	if len(reg.Names()) != 0 {
		t.Errorf("expected empty names after unregister, got %v", reg.Names())
	}
}

// ---------------------------------------------------------------------------
// AuthURL tests
// ---------------------------------------------------------------------------

func TestGoogleAuthURL(t *testing.T) {
	provider := &GoogleProvider{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
	}

	authURL := provider.AuthURL("test-state", "https://example.com/callback")
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse auth URL: %v", err)
	}

	if parsed.Scheme != schemeHTTPS || parsed.Host != "accounts.google.com" {
		t.Errorf("unexpected auth URL base: %s://%s", parsed.Scheme, parsed.Host)
	}
	if parsed.Path != "/o/oauth2/v2/auth" {
		t.Errorf("unexpected path: %s", parsed.Path)
	}

	params := parsed.Query()
	tests := []struct {
		key      string
		expected string
	}{
		{"client_id", "test-client-id"},
		{"redirect_uri", "https://example.com/callback"},
		{"response_type", "code"},
		{"scope", "openid email profile"},
		{"state", "test-state"},
		{"access_type", "offline"},
		{"prompt", "consent"},
	}
	for _, tc := range tests {
		if got := params.Get(tc.key); got != tc.expected {
			t.Errorf("param %q: expected %q, got %q", tc.key, tc.expected, got)
		}
	}
}

func TestGitHubAuthURL(t *testing.T) {
	provider := &GitHubProvider{
		ClientID:     "gh-client-id",
		ClientSecret: "gh-secret",
	}

	authURL := provider.AuthURL("csrf-token", "https://example.com/gh/callback")
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse auth URL: %v", err)
	}

	if parsed.Scheme != schemeHTTPS || parsed.Host != "github.com" {
		t.Errorf("unexpected auth URL base: %s://%s", parsed.Scheme, parsed.Host)
	}
	if parsed.Path != "/login/oauth/authorize" {
		t.Errorf("unexpected path: %s", parsed.Path)
	}

	params := parsed.Query()
	tests := []struct {
		key      string
		expected string
	}{
		{"client_id", "gh-client-id"},
		{"redirect_uri", "https://example.com/gh/callback"},
		{"scope", "user:email read:user"},
		{"state", "csrf-token"},
	}
	for _, tc := range tests {
		if got := params.Get(tc.key); got != tc.expected {
			t.Errorf("param %q: expected %q, got %q", tc.key, tc.expected, got)
		}
	}
}

func TestAppleAuthURL(t *testing.T) {
	provider := &AppleProvider{
		ClientID: "com.example.app",
		TeamID:   "TEAMID",
		KeyID:    "KEYID",
	}

	authURL := provider.AuthURL("apple-state", "https://example.com/apple/callback")
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse auth URL: %v", err)
	}

	if parsed.Scheme != schemeHTTPS || parsed.Host != "appleid.apple.com" {
		t.Errorf("unexpected auth URL base: %s://%s", parsed.Scheme, parsed.Host)
	}
	if parsed.Path != "/auth/authorize" {
		t.Errorf("unexpected path: %s", parsed.Path)
	}

	params := parsed.Query()
	tests := []struct {
		key      string
		expected string
	}{
		{"client_id", "com.example.app"},
		{"redirect_uri", "https://example.com/apple/callback"},
		{"response_type", "code"},
		{"scope", "name email"},
		{"state", "apple-state"},
		{"response_mode", "form_post"},
	}
	for _, tc := range tests {
		if got := params.Get(tc.key); got != tc.expected {
			t.Errorf("param %q: expected %q, got %q", tc.key, tc.expected, got)
		}
	}
}

func TestProviderNames(t *testing.T) {
	tests := []struct {
		provider Provider
		expected string
	}{
		{&GoogleProvider{}, "google"},
		{&GitHubProvider{}, "github"},
		{&AppleProvider{}, "apple"},
	}
	for _, tc := range tests {
		if got := tc.provider.Name(); got != tc.expected {
			t.Errorf("expected name %q, got %q", tc.expected, got)
		}
	}
}

// ---------------------------------------------------------------------------
// Apple: verifyIDToken tests
// ---------------------------------------------------------------------------

func TestVerifyIDTokenValid(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}

	kid := testKID
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	clientID := testClientID
	provider := &AppleProvider{
		ClientID: clientID,
		JWKSURL:  jwksServer.URL,
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":            appleIssuer,
		"aud":            clientID,
		"sub":            "001122",
		"email":          "test@example.com",
		"email_verified": true,
		"exp":            now.Add(time.Hour).Unix(),
		"iat":            now.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	idToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	result, err := provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sub != "001122" {
		t.Errorf("expected sub '001122', got %q", result.Sub)
	}
	if result.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", result.Email)
	}
	if !result.EmailVerified {
		t.Error("expected email_verified to be true")
	}
}

func TestVerifyIDTokenExpired(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}

	kid := testKID
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	clientID := testClientID
	provider := &AppleProvider{
		ClientID: clientID,
		JWKSURL:  jwksServer.URL,
	}

	claims := jwt.MapClaims{
		"iss":   appleIssuer,
		"aud":   clientID,
		"sub":   "001122",
		"email": "test@example.com",
		"exp":   time.Now().Add(-time.Hour).Unix(),
		"iat":   time.Now().Add(-2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	idToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	_, err = provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyIDTokenWrongAudience(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}

	kid := testKID
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	provider := &AppleProvider{
		ClientID: "com.example.correct",
		JWKSURL:  jwksServer.URL,
	}

	claims := jwt.MapClaims{
		"iss":   appleIssuer,
		"aud":   "com.example.wrong",
		"sub":   "001122",
		"email": "test@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	idToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	_, err = provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err == nil {
		t.Fatal("expected error for wrong audience")
	}
}

func TestVerifyIDTokenWrongIssuer(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}

	kid := testKID
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	clientID := testClientID
	provider := &AppleProvider{
		ClientID: clientID,
		JWKSURL:  jwksServer.URL,
	}

	claims := jwt.MapClaims{
		"iss":   "https://evil.example.com",
		"aud":   clientID,
		"sub":   "001122",
		"email": "test@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	idToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	_, err = provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestVerifyIDTokenFakeSignature(t *testing.T) {
	signingKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating signing key: %v", err)
	}
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating wrong key: %v", err)
	}

	kid := testKID
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(wrongKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(wrongKey.E)).Bytes()),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	clientID := testClientID
	provider := &AppleProvider{
		ClientID: clientID,
		JWKSURL:  jwksServer.URL,
	}

	claims := jwt.MapClaims{
		"iss":   appleIssuer,
		"aud":   clientID,
		"sub":   "001122",
		"email": "test@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	idToken, err := token.SignedString(signingKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	_, err = provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err == nil {
		t.Fatal("expected error for token signed with wrong key")
	}
}

func TestVerifyIDTokenEmailVerifiedAsString(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}

	kid := testKID
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	clientID := testClientID
	provider := &AppleProvider{
		ClientID: clientID,
		JWKSURL:  jwksServer.URL,
	}

	claims := jwt.MapClaims{
		"iss":            appleIssuer,
		"aud":            clientID,
		"sub":            "abc",
		"email":          "user@example.com",
		"email_verified": "true",
		"exp":            time.Now().Add(time.Hour).Unix(),
		"iat":            time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	idToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	result, err := provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.EmailVerified {
		t.Error("expected email_verified to be true when sent as string 'true'")
	}
}

// ---------------------------------------------------------------------------
// Apple: verifyIDToken with missing kid header
// ---------------------------------------------------------------------------

func TestVerifyIDTokenMissingKID(t *testing.T) {
	key := generateTestKey(t)
	jwksServer := newJWKSServer(t, key, testKID)
	defer jwksServer.Close()

	provider := &AppleProvider{
		ClientID: testClientID,
		JWKSURL:  jwksServer.URL,
	}

	claims := validAppleClaims(testClientID)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	// Deliberately do NOT set token.Header["kid"]

	idToken, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	_, err = provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err == nil {
		t.Fatal("expected error for missing kid header")
	}
	if !strings.Contains(err.Error(), "kid") {
		t.Errorf("expected error to mention 'kid', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Apple: verifyIDToken with unsupported signing algorithm (HS256)
// ---------------------------------------------------------------------------

func TestVerifyIDTokenUnsupportedAlgorithm(t *testing.T) {
	key := generateTestKey(t)
	jwksServer := newJWKSServer(t, key, testKID)
	defer jwksServer.Close()

	provider := &AppleProvider{
		ClientID: testClientID,
		JWKSURL:  jwksServer.URL,
	}

	claims := validAppleClaims(testClientID)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = testKID

	hmacSecret := []byte("some-hmac-secret")
	idToken, err := token.SignedString(hmacSecret)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	_, err = provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err == nil {
		t.Fatal("expected error for unsupported signing algorithm")
	}
}

// ---------------------------------------------------------------------------
// Apple: verifyIDToken with no matching kid in JWKS
// ---------------------------------------------------------------------------

func TestVerifyIDTokenNoMatchingKID(t *testing.T) {
	key := generateTestKey(t)
	jwksServer := newJWKSServer(t, key, "different-kid")
	defer jwksServer.Close()

	provider := &AppleProvider{
		ClientID: testClientID,
		JWKSURL:  jwksServer.URL,
	}

	claims := validAppleClaims(testClientID)
	idToken := signToken(t, key, "nonexistent-kid", claims)

	_, err := provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err == nil {
		t.Fatal("expected error for non-matching kid")
	}
	if !strings.Contains(err.Error(), "no matching key") {
		t.Errorf("expected 'no matching key' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Apple: JWKS fetch failure (HTTP 500)
// ---------------------------------------------------------------------------

func TestVerifyIDTokenJWKSFetchFailure(t *testing.T) {
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer jwksServer.Close()

	provider := &AppleProvider{
		ClientID: testClientID,
		JWKSURL:  jwksServer.URL,
	}

	key := generateTestKey(t)
	claims := validAppleClaims(testClientID)
	idToken := signToken(t, key, testKID, claims)

	_, err := provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err == nil {
		t.Fatal("expected error when JWKS endpoint returns 500")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Apple: JWKS fetch returns invalid JSON
// ---------------------------------------------------------------------------

func TestVerifyIDTokenJWKSInvalidJSON(t *testing.T) {
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer jwksServer.Close()

	provider := &AppleProvider{
		ClientID: testClientID,
		JWKSURL:  jwksServer.URL,
	}

	key := generateTestKey(t)
	claims := validAppleClaims(testClientID)
	idToken := signToken(t, key, testKID, claims)

	_, err := provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err == nil {
		t.Fatal("expected error when JWKS endpoint returns invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// Apple: JWKS returns empty keys array
// ---------------------------------------------------------------------------

func TestVerifyIDTokenJWKSEmptyKeys(t *testing.T) {
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	defer jwksServer.Close()

	provider := &AppleProvider{
		ClientID: testClientID,
		JWKSURL:  jwksServer.URL,
	}

	key := generateTestKey(t)
	claims := validAppleClaims(testClientID)
	idToken := signToken(t, key, testKID, claims)

	_, err := provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err == nil {
		t.Fatal("expected error when JWKS contains no keys")
	}
	if !strings.Contains(err.Error(), "no keys") {
		t.Errorf("expected 'no keys' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Apple: JWKS cache expiry and refresh
// ---------------------------------------------------------------------------

func TestJWKSCacheExpiryAndRefresh(t *testing.T) {
	key := generateTestKey(t)
	var fetchCount int32

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&fetchCount, 1)
		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": testKID,
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	provider := &AppleProvider{
		ClientID: testClientID,
		JWKSURL:  jwksServer.URL,
	}

	ctx := context.Background()
	client := http.DefaultClient

	// First fetch: should hit the server.
	idToken := signToken(t, key, testKID, validAppleClaims(testClientID))
	if _, err := provider.verifyIDToken(ctx, client, idToken); err != nil {
		t.Fatalf("first verification failed: %v", err)
	}
	if c := atomic.LoadInt32(&fetchCount); c != 1 {
		t.Fatalf("expected 1 JWKS fetch, got %d", c)
	}

	// Second fetch (cache hit): should NOT hit the server again.
	idToken = signToken(t, key, testKID, validAppleClaims(testClientID))
	if _, err := provider.verifyIDToken(ctx, client, idToken); err != nil {
		t.Fatalf("second verification failed: %v", err)
	}
	if c := atomic.LoadInt32(&fetchCount); c != 1 {
		t.Fatalf("expected still 1 JWKS fetch (cache hit), got %d", c)
	}

	// Simulate cache expiry by backdating the FetchedAt.
	provider.jwksCacheMu.Lock()
	provider.jwksCache.FetchedAt = time.Now().Add(-2 * jwksCacheTTL)
	provider.jwksCacheMu.Unlock()

	// Third fetch (cache expired): should hit the server again.
	idToken = signToken(t, key, testKID, validAppleClaims(testClientID))
	if _, err := provider.verifyIDToken(ctx, client, idToken); err != nil {
		t.Fatalf("third verification (after cache expiry) failed: %v", err)
	}
	if c := atomic.LoadInt32(&fetchCount); c != 2 {
		t.Fatalf("expected 2 JWKS fetches after cache expiry, got %d", c)
	}
}

// ---------------------------------------------------------------------------
// Apple: JWKS network error (connection refused)
// ---------------------------------------------------------------------------

func TestVerifyIDTokenJWKSNetworkError(t *testing.T) {
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	jwksURL := jwksServer.URL
	jwksServer.Close()

	provider := &AppleProvider{
		ClientID: testClientID,
		JWKSURL:  jwksURL,
	}

	key := generateTestKey(t)
	claims := validAppleClaims(testClientID)
	idToken := signToken(t, key, testKID, claims)

	_, err := provider.verifyIDToken(context.Background(), http.DefaultClient, idToken)
	if err == nil {
		t.Fatal("expected error when JWKS server is unreachable")
	}
}

// ---------------------------------------------------------------------------
// appleIDTokenClaims UnmarshalJSON: various email_verified types
// ---------------------------------------------------------------------------

func TestAppleIDTokenClaimsUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name             string
		jsonStr          string
		expectedVerified bool
		expectedSub      string
		expectedEmail    string
	}{
		{
			name:             "email_verified as bool true",
			jsonStr:          `{"sub":"s1","email":"a@b.com","email_verified":true}`,
			expectedVerified: true,
			expectedSub:      "s1",
			expectedEmail:    "a@b.com",
		},
		{
			name:             "email_verified as bool false",
			jsonStr:          `{"sub":"s2","email":"c@d.com","email_verified":false}`,
			expectedVerified: false,
			expectedSub:      "s2",
			expectedEmail:    "c@d.com",
		},
		{
			name:             "email_verified as string true",
			jsonStr:          `{"sub":"s3","email":"e@f.com","email_verified":"true"}`,
			expectedVerified: true,
			expectedSub:      "s3",
			expectedEmail:    "e@f.com",
		},
		{
			name:             "email_verified as string false",
			jsonStr:          `{"sub":"s4","email":"g@h.com","email_verified":"false"}`,
			expectedVerified: false,
			expectedSub:      "s4",
			expectedEmail:    "g@h.com",
		},
		{
			name:             "email_verified as null",
			jsonStr:          `{"sub":"s5","email":"i@j.com","email_verified":null}`,
			expectedVerified: false,
			expectedSub:      "s5",
			expectedEmail:    "i@j.com",
		},
		{
			name:             "email_verified as number",
			jsonStr:          `{"sub":"s6","email":"k@l.com","email_verified":1}`,
			expectedVerified: false,
			expectedSub:      "s6",
			expectedEmail:    "k@l.com",
		},
		{
			name:             "email_verified missing",
			jsonStr:          `{"sub":"s7","email":"m@n.com"}`,
			expectedVerified: false,
			expectedSub:      "s7",
			expectedEmail:    "m@n.com",
		},
		{
			name:             "email_verified as random string",
			jsonStr:          `{"sub":"s8","email":"o@p.com","email_verified":"yes"}`,
			expectedVerified: false,
			expectedSub:      "s8",
			expectedEmail:    "o@p.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var claims appleIDTokenClaims
			if err := json.Unmarshal([]byte(tc.jsonStr), &claims); err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if claims.Sub != tc.expectedSub {
				t.Errorf("expected sub %q, got %q", tc.expectedSub, claims.Sub)
			}
			if claims.Email != tc.expectedEmail {
				t.Errorf("expected email %q, got %q", tc.expectedEmail, claims.Email)
			}
			if claims.EmailVerified != tc.expectedVerified {
				t.Errorf("expected email_verified=%v, got %v", tc.expectedVerified, claims.EmailVerified)
			}
		})
	}
}

func TestAppleIDTokenClaimsUnmarshalInvalidJSON(t *testing.T) {
	var claims appleIDTokenClaims
	err := json.Unmarshal([]byte(`{not valid`), &claims)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// base64URLDecode: various padding scenarios
// ---------------------------------------------------------------------------

func TestBase64URLDecode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "no padding needed (len%4==0)",
			input: base64.RawURLEncoding.EncodeToString([]byte("test")),
			want:  "test",
		},
		{
			name:  "1 pad needed (len%4==3)",
			input: base64.RawURLEncoding.EncodeToString([]byte("ab")),
			want:  "ab",
		},
		{
			name:  "2 pads needed (len%4==2)",
			input: base64.RawURLEncoding.EncodeToString([]byte("a")),
			want:  "a",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "longer input no padding",
			input: base64.RawURLEncoding.EncodeToString([]byte("hello world")),
			want:  "hello world",
		},
		{
			name:    "invalid base64",
			input:   "!!!invalid!!!",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := base64URLDecode(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("expected %q, got %q", tc.want, string(got))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// jwkToRSAPublicKey edge cases
// ---------------------------------------------------------------------------

func TestJWKToRSAPublicKeyNonRSA(t *testing.T) {
	jwk := &appleJWK{
		KTY: "EC",
		KID: "ec-key",
		N:   "AAAA",
		E:   "AQAB",
	}
	_, err := jwkToRSAPublicKey(jwk)
	if err == nil {
		t.Fatal("expected error for non-RSA key type")
	}
	if !strings.Contains(err.Error(), "unsupported key type") {
		t.Errorf("expected 'unsupported key type' error, got: %v", err)
	}
}

func TestJWKToRSAPublicKeyInvalidModulus(t *testing.T) {
	jwk := &appleJWK{
		KTY: "RSA",
		KID: "bad-n",
		N:   "!!!invalid-base64!!!",
		E:   "AQAB",
	}
	_, err := jwkToRSAPublicKey(jwk)
	if err == nil {
		t.Fatal("expected error for invalid modulus encoding")
	}
	if !strings.Contains(err.Error(), "decoding modulus") {
		t.Errorf("expected 'decoding modulus' error, got: %v", err)
	}
}

func TestJWKToRSAPublicKeyInvalidExponent(t *testing.T) {
	key := generateTestKey(t)
	jwk := &appleJWK{
		KTY: "RSA",
		KID: "bad-e",
		N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		E:   "!!!invalid!!!",
	}
	_, err := jwkToRSAPublicKey(jwk)
	if err == nil {
		t.Fatal("expected error for invalid exponent encoding")
	}
	if !strings.Contains(err.Error(), "decoding exponent") {
		t.Errorf("expected 'decoding exponent' error, got: %v", err)
	}
}

func TestJWKToRSAPublicKeyValid(t *testing.T) {
	key := generateTestKey(t)
	jwk := &appleJWK{
		KTY: "RSA",
		KID: "valid-key",
		N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
	pub, err := jwkToRSAPublicKey(jwk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pub.N.Cmp(key.N) != 0 {
		t.Error("modulus mismatch")
	}
	if pub.E != key.E {
		t.Errorf("exponent mismatch: got %d, want %d", pub.E, key.E)
	}
}

// ---------------------------------------------------------------------------
// Apple: Exchange tests
// ---------------------------------------------------------------------------

func TestAppleExchangeTokenEndpointError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer tokenServer.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(tokenServer.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &AppleProvider{
		ClientID:         testClientID,
		HTTPClient:       &http.Client{Transport: transport},
		ClientSecretFunc: func() (string, error) { return testClientSecret, nil },
	}

	_, err := provider.Exchange(context.Background(), "bad-code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error when token endpoint returns 400")
	}
	if !strings.Contains(err.Error(), "token exchange") {
		t.Errorf("expected 'token exchange' in error, got: %v", err)
	}
}

func TestAppleExchangeClientSecretError(t *testing.T) {
	provider := &AppleProvider{
		ClientID: testClientID,
		ClientSecretFunc: func() (string, error) {
			return "", fmt.Errorf("key file not found")
		},
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error when client secret generation fails")
	}
	if !strings.Contains(err.Error(), "client secret") {
		t.Errorf("expected 'client secret' in error, got: %v", err)
	}
}

func TestAppleExchangeNoClientSecretFunc(t *testing.T) {
	provider := &AppleProvider{
		ClientID: testClientID,
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error when ClientSecretFunc is not configured")
	}
	if !strings.Contains(err.Error(), "ClientSecretFunc") {
		t.Errorf("expected 'ClientSecretFunc' in error, got: %v", err)
	}
}

func TestAppleExchangeSuccess(t *testing.T) {
	key := generateTestKey(t)

	jwksServer := newJWKSServer(t, key, testKID)
	defer jwksServer.Close()

	claims := validAppleClaims(testClientID)
	idToken := signToken(t, key, testKID, claims)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"access_token":  "apple-access-token",
			"refresh_token": "apple-refresh-token",
			"id_token":      idToken,
			"token_type":    "bearer",
			"expires_in":    3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	parsedJWKS, _ := url.Parse(jwksServer.URL)
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		// Let JWKS requests through to the real JWKS server.
		if req.URL.Host == parsedJWKS.Host {
			return http.DefaultTransport.RoundTrip(req)
		}
		req.URL, _ = url.Parse(tokenServer.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &AppleProvider{
		ClientID:         testClientID,
		HTTPClient:       &http.Client{Transport: transport},
		JWKSURL:          jwksServer.URL,
		ClientSecretFunc: func() (string, error) { return testClientSecret, nil },
	}

	info, err := provider.Exchange(context.Background(), "auth-code", "https://example.com/callback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ProviderUserID != "user-123" {
		t.Errorf("expected sub 'user-123', got %q", info.ProviderUserID)
	}
	if info.Email != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got %q", info.Email)
	}
	if !info.EmailVerified {
		t.Error("expected email_verified to be true")
	}
	if info.AccessToken != "apple-access-token" {
		t.Errorf("expected access token 'apple-access-token', got %q", info.AccessToken)
	}
	if info.RefreshToken != "apple-refresh-token" {
		t.Errorf("expected refresh token 'apple-refresh-token', got %q", info.RefreshToken)
	}
	if info.TokenExpiresAt == nil {
		t.Error("expected TokenExpiresAt to be set")
	}
}

func TestAppleExchangeInvalidTokenJSON(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer tokenServer.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(tokenServer.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &AppleProvider{
		ClientID:         testClientID,
		HTTPClient:       &http.Client{Transport: transport},
		ClientSecretFunc: func() (string, error) { return testClientSecret, nil },
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error for invalid token JSON response")
	}
}

// ---------------------------------------------------------------------------
// Google: Exchange tests
// ---------------------------------------------------------------------------

func TestGoogleExchangeTokenError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer tokenServer.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(tokenServer.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GoogleProvider{
		ClientID:     "google-id",
		ClientSecret: "google-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	_, err := provider.Exchange(context.Background(), "bad-code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error when Google token endpoint returns 400")
	}
	if !strings.Contains(err.Error(), "token exchange") {
		t.Errorf("expected 'token exchange' in error, got: %v", err)
	}
}

func TestGoogleExchangeUserInfoFailure(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			resp := map[string]any{
				"access_token":  "google-token",
				"refresh_token": "google-refresh",
				"expires_in":    3600,
				"token_type":    "Bearer",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GoogleProvider{
		ClientID:     "google-id",
		ClientSecret: "google-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error when Google userinfo endpoint fails")
	}
	if !strings.Contains(err.Error(), "user info") {
		t.Errorf("expected 'user info' in error, got: %v", err)
	}
}

func TestGoogleExchangeInvalidTokenJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{broken`))
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GoogleProvider{
		ClientID:     "google-id",
		ClientSecret: "google-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error for invalid JSON from Google token endpoint")
	}
}

func TestGoogleExchangeInvalidUserInfoJSON(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			resp := map[string]any{
				"access_token": "token",
				"expires_in":   3600,
				"token_type":   "Bearer",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GoogleProvider{
		ClientID:     "google-id",
		ClientSecret: "google-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error for invalid JSON from Google userinfo endpoint")
	}
}

func TestGoogleExchangeSuccess(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "g-access",
				"refresh_token": "g-refresh",
				"expires_in":    3600,
				"token_type":    "Bearer",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":            "google-user-1",
			"email":          "guser@gmail.com",
			"email_verified": true,
			"name":           "Test User",
			"given_name":     "Test",
			"family_name":    "User",
			"picture":        "https://example.com/avatar.jpg",
		})
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GoogleProvider{
		ClientID:     "google-id",
		ClientSecret: "google-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	info, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ProviderUserID != "google-user-1" {
		t.Errorf("expected sub 'google-user-1', got %q", info.ProviderUserID)
	}
	if info.Email != "guser@gmail.com" {
		t.Errorf("expected email 'guser@gmail.com', got %q", info.Email)
	}
	if info.Name != "Test User" {
		t.Errorf("expected name 'Test User', got %q", info.Name)
	}
	if info.AvatarURL != "https://example.com/avatar.jpg" {
		t.Errorf("expected avatar URL, got %q", info.AvatarURL)
	}
	if info.TokenExpiresAt == nil {
		t.Error("expected TokenExpiresAt to be set")
	}
}

// ---------------------------------------------------------------------------
// GitHub: Exchange tests
// ---------------------------------------------------------------------------

func TestGitHubExchangeTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad_verification_code"}`))
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GitHubProvider{
		ClientID:     "gh-id",
		ClientSecret: "gh-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	_, err := provider.Exchange(context.Background(), "bad-code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error when GitHub token endpoint returns 401")
	}
	if !strings.Contains(err.Error(), "token exchange") {
		t.Errorf("expected 'token exchange' in error, got: %v", err)
	}
}

func TestGitHubExchangeEmptyAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "",
			"token_type":   "bearer",
		})
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GitHubProvider{
		ClientID:     "gh-id",
		ClientSecret: "gh-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error for empty access token")
	}
	if !strings.Contains(err.Error(), "empty access token") {
		t.Errorf("expected 'empty access token' in error, got: %v", err)
	}
}

func TestGitHubExchangeUserAPIFailure(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "gh-token",
				"token_type":   "bearer",
			})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GitHubProvider{
		ClientID:     "gh-id",
		ClientSecret: "gh-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error when GitHub user API fails")
	}
	if !strings.Contains(err.Error(), "fetch user") {
		t.Errorf("expected 'fetch user' in error, got: %v", err)
	}
}

func TestGitHubExchangeEmailAPIFailure(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "gh-token",
				"token_type":   "bearer",
			})
			return
		}
		if callCount == 2 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         12345,
				"login":      "testuser",
				"name":       "Test User",
				"avatar_url": "https://example.com/avatar.jpg",
			})
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GitHubProvider{
		ClientID:     "gh-id",
		ClientSecret: "gh-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error when GitHub email API fails")
	}
	if !strings.Contains(err.Error(), "fetch email") {
		t.Errorf("expected 'fetch email' in error, got: %v", err)
	}
}

func TestGitHubExchangeInvalidTokenJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{broken`))
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GitHubProvider{
		ClientID:     "gh-id",
		ClientSecret: "gh-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error for invalid JSON from GitHub token endpoint")
	}
}

func TestGitHubExchangeInvalidUserJSON(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "gh-token",
				"token_type":   "bearer",
			})
			return
		}
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GitHubProvider{
		ClientID:     "gh-id",
		ClientSecret: "gh-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error for invalid JSON from GitHub user endpoint")
	}
}

func TestGitHubExchangeInvalidEmailJSON(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "gh-token",
				"token_type":   "bearer",
			})
			return
		}
		if callCount == 2 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    12345,
				"login": "testuser",
			})
			return
		}
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GitHubProvider{
		ClientID:     "gh-id",
		ClientSecret: "gh-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error for invalid JSON from GitHub email endpoint")
	}
}

func TestGitHubExchangeSuccess(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		switch callCount {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "gh-access",
				"refresh_token": "gh-refresh",
				"expires_in":    3600,
				"token_type":    "bearer",
				"scope":         "user:email read:user",
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         42,
				"login":      "octocat",
				"name":       "Octo Cat",
				"avatar_url": "https://github.com/avatar.jpg",
			})
		case 3:
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"email": "octocat@github.com", "primary": true, "verified": true},
				{"email": "secondary@example.com", "primary": false, "verified": false},
			})
		}
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GitHubProvider{
		ClientID:     "gh-id",
		ClientSecret: "gh-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	info, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ProviderUserID != "42" {
		t.Errorf("expected provider user ID '42', got %q", info.ProviderUserID)
	}
	if info.Email != "octocat@github.com" {
		t.Errorf("expected email 'octocat@github.com', got %q", info.Email)
	}
	if !info.EmailVerified {
		t.Error("expected email_verified to be true")
	}
	if info.Name != "Octo Cat" {
		t.Errorf("expected name 'Octo Cat', got %q", info.Name)
	}
	if info.AvatarURL != "https://github.com/avatar.jpg" {
		t.Errorf("expected avatar URL, got %q", info.AvatarURL)
	}
	if info.TokenExpiresAt == nil {
		t.Error("expected TokenExpiresAt to be set")
	}
}

func TestGitHubExchangeNoPrimaryEmail(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		switch callCount {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "gh-access",
				"token_type":   "bearer",
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    1,
				"login": "user",
			})
		case 3:
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"email": "fallback@example.com", "primary": false, "verified": true},
			})
		}
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GitHubProvider{
		ClientID:     "gh-id",
		ClientSecret: "gh-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	info, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Email != "fallback@example.com" {
		t.Errorf("expected fallback email, got %q", info.Email)
	}
}

func TestGitHubExchangeNoEmails(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		switch callCount {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "gh-access",
				"token_type":   "bearer",
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    1,
				"login": "user",
			})
		case 3:
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	}))
	defer server.Close()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(server.URL + req.URL.Path)
		return http.DefaultTransport.RoundTrip(req)
	})

	provider := &GitHubProvider{
		ClientID:     "gh-id",
		ClientSecret: "gh-secret",
		HTTPClient:   &http.Client{Transport: transport},
	}

	_, err := provider.Exchange(context.Background(), "code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error when no emails are returned")
	}
	if !strings.Contains(err.Error(), "no email") {
		t.Errorf("expected 'no email' in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// httpClient and jwksEndpoint helper tests
// ---------------------------------------------------------------------------

func TestAppleHTTPClient(t *testing.T) {
	provider := &AppleProvider{}
	if provider.httpClient() != http.DefaultClient {
		t.Error("expected default client when HTTPClient is nil")
	}

	custom := &http.Client{Timeout: 5 * time.Second}
	provider.HTTPClient = custom
	if provider.httpClient() != custom {
		t.Error("expected custom client when HTTPClient is set")
	}
}

func TestGoogleHTTPClient(t *testing.T) {
	provider := &GoogleProvider{}
	if provider.httpClient() != http.DefaultClient {
		t.Error("expected default client when HTTPClient is nil")
	}

	custom := &http.Client{Timeout: 5 * time.Second}
	provider.HTTPClient = custom
	if provider.httpClient() != custom {
		t.Error("expected custom client when HTTPClient is set")
	}
}

func TestGitHubHTTPClient(t *testing.T) {
	provider := &GitHubProvider{}
	if provider.httpClient() != http.DefaultClient {
		t.Error("expected default client when HTTPClient is nil")
	}

	custom := &http.Client{Timeout: 5 * time.Second}
	provider.HTTPClient = custom
	if provider.httpClient() != custom {
		t.Error("expected custom client when HTTPClient is set")
	}
}

func TestAppleJWKSEndpoint(t *testing.T) {
	provider := &AppleProvider{}
	if provider.jwksEndpoint() != appleJWKSURL {
		t.Errorf("expected default JWKS URL, got %q", provider.jwksEndpoint())
	}

	provider.JWKSURL = "https://custom.example.com/keys"
	if provider.jwksEndpoint() != "https://custom.example.com/keys" {
		t.Errorf("expected custom JWKS URL, got %q", provider.jwksEndpoint())
	}
}
