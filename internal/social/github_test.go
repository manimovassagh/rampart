package social

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubExchangeSuccess(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := githubTokenResponse{
			AccessToken:  "gh-access-token",
			RefreshToken: "gh-refresh-token",
			ExpiresIn:    3600,
			TokenType:    "bearer",
			Scope:        "user:email read:user",
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode token response: %v", err)
		}
	}))
	defer tokenServer.Close()

	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := githubUserResponse{
			ID:        12345,
			Login:     "testuser",
			Name:      "Test User",
			AvatarURL: "https://avatars.githubusercontent.com/u/12345",
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode user response: %v", err)
		}
	}))
	defer userServer.Close()

	emailServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := []githubEmailResponse{
			{Email: "secondary@example.com", Primary: false, Verified: true},
			{Email: "primary@example.com", Primary: true, Verified: true},
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode email response: %v", err)
		}
	}))
	defer emailServer.Close()

	provider := &GitHubProvider{
		ClientID:     "gh-client-id",
		ClientSecret: "gh-client-secret",
		HTTPClient: newRedirectClient(map[string]string{
			githubTokenURL: tokenServer.URL,
			githubUserURL:  userServer.URL,
			githubEmailURL: emailServer.URL,
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
		{"ProviderUserID", info.ProviderUserID, "12345"},
		{"Email", info.Email, "primary@example.com"},
		{"Name", info.Name, "Test User"},
		{"AvatarURL", info.AvatarURL, "https://avatars.githubusercontent.com/u/12345"},
		{"AccessToken", info.AccessToken, "gh-access-token"},
		{"RefreshToken", info.RefreshToken, "gh-refresh-token"},
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

func TestGitHubExchangeTokenError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad_verification_code"}`))
	}))
	defer tokenServer.Close()

	provider := &GitHubProvider{
		ClientID:     "gh-client-id",
		ClientSecret: "gh-client-secret",
		HTTPClient: newRedirectClient(map[string]string{
			githubTokenURL: tokenServer.URL,
		}),
	}

	_, err := provider.Exchange(context.Background(), "bad-code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGitHubExchangeNoPrimaryEmail(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := githubTokenResponse{AccessToken: "gh-token", ExpiresIn: 3600}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode token response: %v", err)
		}
	}))
	defer tokenServer.Close()

	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := githubUserResponse{ID: 99, Login: "noprimary", Name: "No Primary"}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode user response: %v", err)
		}
	}))
	defer userServer.Close()

	emailServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// No primary email, should fall back to first email
		resp := []githubEmailResponse{
			{Email: "fallback@example.com", Primary: false, Verified: false},
			{Email: "second@example.com", Primary: false, Verified: true},
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode email response: %v", err)
		}
	}))
	defer emailServer.Close()

	provider := &GitHubProvider{
		ClientID:     "gh-client-id",
		ClientSecret: "gh-client-secret",
		HTTPClient: newRedirectClient(map[string]string{
			githubTokenURL: tokenServer.URL,
			githubUserURL:  userServer.URL,
			githubEmailURL: emailServer.URL,
		}),
	}

	info, err := provider.Exchange(context.Background(), "auth-code", "https://example.com/callback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Email != "fallback@example.com" {
		t.Errorf("Email = %q, want fallback@example.com (first email as fallback)", info.Email)
	}
	if info.EmailVerified {
		t.Error("expected EmailVerified to be false for unverified fallback email")
	}
}

func TestGitHubExchangeEmptyAccessToken(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// GitHub returns 200 with error details when code is invalid
		_, _ = w.Write([]byte(`{"access_token":"","error":"bad_verification_code"}`))
	}))
	defer tokenServer.Close()

	provider := &GitHubProvider{
		ClientID:     "gh-client-id",
		ClientSecret: "gh-client-secret",
		HTTPClient: newRedirectClient(map[string]string{
			githubTokenURL: tokenServer.URL,
		}),
	}

	_, err := provider.Exchange(context.Background(), "bad-code", "https://example.com/callback")
	if err == nil {
		t.Fatal("expected error for empty access token, got nil")
	}
}
