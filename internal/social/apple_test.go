package social

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// buildFakeIDToken constructs a fake JWT with the given claims payload.
func buildFakeIDToken(claims interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	payload, _ := json.Marshal(claims)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + payloadB64 + ".fake-signature"
}

func TestAppleExchangeSuccess(t *testing.T) {
	idToken := buildFakeIDToken(map[string]interface{}{
		"sub":            "apple-uid-001",
		"email":          "user@icloud.com",
		"email_verified": true,
	})

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := appleTokenResponse{
			AccessToken:  "apple-access-token",
			RefreshToken: "apple-refresh-token",
			IDToken:      idToken,
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode token response: %v", err)
		}
	}))
	defer tokenServer.Close()

	provider := &AppleProvider{
		ClientID: "com.example.app",
		TeamID:   "TEAMID",
		KeyID:    "KEYID",
		HTTPClient: newRedirectClient(map[string]string{
			appleTokenURL: tokenServer.URL,
		}),
		ClientSecretFunc: func() (string, error) {
			return "fake-client-secret-jwt", nil
		},
	}

	info, err := provider.Exchange(context.Background(), "auth-code", "https://example.com/callback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		field    string
		got      string
		expected string
	}{
		{"ProviderUserID", info.ProviderUserID, "apple-uid-001"},
		{"Email", info.Email, "user@icloud.com"},
		{"AccessToken", info.AccessToken, "apple-access-token"},
		{"RefreshToken", info.RefreshToken, "apple-refresh-token"},
	}
	for _, tc := range tests {
		if tc.got != tc.expected {
			t.Errorf("%s = %q, want %q", tc.field, tc.got, tc.expected)
		}
	}

	if !info.EmailVerified {
		t.Error("expected EmailVerified to be true")
	}
	if info.TokenExpiresAt == nil {
		t.Error("expected TokenExpiresAt to be set")
	}
}

func TestAppleExchangeNoClientSecret(t *testing.T) {
	provider := &AppleProvider{
		ClientID: "com.example.app",
		TeamID:   "TEAMID",
		KeyID:    "KEYID",
		// ClientSecretFunc is nil
	}

	_, err := provider.Exchange(context.Background(), "auth-code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error when ClientSecretFunc is nil, got nil")
	}
}

func TestAppleExchangeClientSecretFuncError(t *testing.T) {
	provider := &AppleProvider{
		ClientID: "com.example.app",
		ClientSecretFunc: func() (string, error) {
			return "", fmt.Errorf("key generation failed")
		},
	}

	_, err := provider.Exchange(context.Background(), "auth-code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error when ClientSecretFunc returns error, got nil")
	}
}

func TestAppleExchangeTokenError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer tokenServer.Close()

	provider := &AppleProvider{
		ClientID: "com.example.app",
		HTTPClient: newRedirectClient(map[string]string{
			appleTokenURL: tokenServer.URL,
		}),
		ClientSecretFunc: func() (string, error) {
			return "fake-secret", nil
		},
	}

	_, err := provider.Exchange(context.Background(), "bad-code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAppleExchangeInvalidIDToken(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := appleTokenResponse{
			AccessToken: "apple-token",
			IDToken:     "not-a-valid-jwt",
			ExpiresIn:   3600,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode token response: %v", err)
		}
	}))
	defer tokenServer.Close()

	provider := &AppleProvider{
		ClientID: "com.example.app",
		HTTPClient: newRedirectClient(map[string]string{
			appleTokenURL: tokenServer.URL,
		}),
		ClientSecretFunc: func() (string, error) {
			return "fake-secret", nil
		},
	}

	_, err := provider.Exchange(context.Background(), "auth-code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error for invalid ID token, got nil")
	}
}

func TestAppleExchangeEmailVerifiedAsString(t *testing.T) {
	idToken := buildFakeIDToken(map[string]interface{}{
		"sub":            "apple-uid-002",
		"email":          "str@icloud.com",
		"email_verified": "true",
	})

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := appleTokenResponse{
			AccessToken: "token",
			IDToken:     idToken,
			ExpiresIn:   3600,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode token response: %v", err)
		}
	}))
	defer tokenServer.Close()

	provider := &AppleProvider{
		ClientID: "com.example.app",
		HTTPClient: newRedirectClient(map[string]string{
			appleTokenURL: tokenServer.URL,
		}),
		ClientSecretFunc: func() (string, error) {
			return "fake-secret", nil
		},
	}

	info, err := provider.Exchange(context.Background(), "auth-code", "https://example.com/callback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !info.EmailVerified {
		t.Error("expected EmailVerified to be true when sent as string")
	}
}
