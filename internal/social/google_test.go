package social

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoogleExchangeSuccess(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := googleTokenResponse{
			AccessToken:  "google-access-token",
			RefreshToken: "google-refresh-token",
			ExpiresIn:    3600,
			TokenType:    "Bearer",
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode token response: %v", err)
		}
	}))
	defer tokenServer.Close()

	userInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := googleUserInfoResponse{
			Sub:           "google-uid-123",
			Email:         "user@gmail.com",
			EmailVerified: true,
			Name:          "Test User",
			GivenName:     "Test",
			FamilyName:    "User",
			Picture:       "https://example.com/photo.jpg",
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode userinfo response: %v", err)
		}
	}))
	defer userInfoServer.Close()

	provider := &GoogleProvider{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		HTTPClient:   newRedirectClient(map[string]string{
			googleTokenURL:    tokenServer.URL,
			googleUserInfoURL: userInfoServer.URL,
		}),
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
		{"ProviderUserID", info.ProviderUserID, "google-uid-123"},
		{"Email", info.Email, "user@gmail.com"},
		{"Name", info.Name, "Test User"},
		{"GivenName", info.GivenName, "Test"},
		{"FamilyName", info.FamilyName, "User"},
		{"AvatarURL", info.AvatarURL, "https://example.com/photo.jpg"},
		{"AccessToken", info.AccessToken, "google-access-token"},
		{"RefreshToken", info.RefreshToken, "google-refresh-token"},
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

func TestGoogleExchangeTokenError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer tokenServer.Close()

	provider := &GoogleProvider{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		HTTPClient:   newRedirectClient(map[string]string{
			googleTokenURL: tokenServer.URL,
		}),
	}

	_, err := provider.Exchange(context.Background(), "bad-code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGoogleExchangeUserInfoError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := googleTokenResponse{AccessToken: "token", ExpiresIn: 3600}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode token response: %v", err)
		}
	}))
	defer tokenServer.Close()

	userInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`internal error`))
	}))
	defer userInfoServer.Close()

	provider := &GoogleProvider{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		HTTPClient:   newRedirectClient(map[string]string{
			googleTokenURL:    tokenServer.URL,
			googleUserInfoURL: userInfoServer.URL,
		}),
	}

	_, err := provider.Exchange(context.Background(), "auth-code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
