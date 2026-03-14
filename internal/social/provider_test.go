package social

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	schemeHTTPS  = "https"
	testKID      = "test-key-1"
	testClientID = "com.example.test"
)

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

func TestVerifyIDTokenValid(t *testing.T) {
	// Generate an RSA key pair for testing.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}

	// Create a mock JWKS server.
	kid := testKID
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
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

	// Create a valid ID token.
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
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
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

	// Create an expired token.
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
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
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

	// Token with wrong audience.
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
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
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

	// Token with wrong issuer.
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
	// Generate two different key pairs — sign with one, serve the other in JWKS.
	signingKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating signing key: %v", err)
	}
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating wrong key: %v", err)
	}

	kid := testKID
	// Serve wrongKey in JWKS.
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
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

	// Sign with signingKey (not the one in JWKS).
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
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
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

	// Apple sometimes sends email_verified as a string "true".
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
