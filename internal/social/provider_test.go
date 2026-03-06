package social

import (
	"net/url"
	"testing"
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

	if parsed.Scheme != "https" || parsed.Host != "accounts.google.com" {
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

	if parsed.Scheme != "https" || parsed.Host != "github.com" {
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

	if parsed.Scheme != "https" || parsed.Host != "appleid.apple.com" {
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

func TestDecodeIDToken(t *testing.T) {
	// Build a fake JWT with a known payload.
	// Header and signature don't matter for decoding.
	// Payload: {"sub":"001122","email":"test@example.com","email_verified":true}
	header := "eyJhbGciOiJSUzI1NiJ9"                                                                     // {"alg":"RS256"}
	payload := "eyJzdWIiOiIwMDExMjIiLCJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZX0" // base64url of the JSON
	signature := "fake-signature"

	token := header + "." + payload + "." + signature
	claims, err := decodeIDToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.Sub != "001122" {
		t.Errorf("expected sub '001122', got %q", claims.Sub)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", claims.Email)
	}
	if !claims.EmailVerified {
		t.Error("expected email_verified to be true")
	}
}

func TestDecodeIDTokenInvalid(t *testing.T) {
	_, err := decodeIDToken("not-a-jwt")
	if err == nil {
		t.Error("expected error for invalid JWT")
	}
}

func TestDecodeIDTokenEmailVerifiedAsString(t *testing.T) {
	// Payload: {"sub":"abc","email":"user@example.com","email_verified":"true"}
	header := "eyJhbGciOiJSUzI1NiJ9"
	payload := "eyJzdWIiOiJhYmMiLCJlbWFpbCI6InVzZXJAZXhhbXBsZS5jb20iLCJlbWFpbF92ZXJpZmllZCI6InRydWUifQ"
	signature := "sig"

	token := header + "." + payload + "." + signature
	claims, err := decodeIDToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !claims.EmailVerified {
		t.Error("expected email_verified to be true when sent as string 'true'")
	}
}
