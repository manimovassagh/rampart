package config

import (
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("RAMPART_DB_URL", "postgres://localhost:5432/rampart")
}

func TestLoadDefaults(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.SigningKeyPath != "rampart-signing-key.pem" {
		t.Errorf("SigningKeyPath = %q, want rampart-signing-key.pem", cfg.SigningKeyPath)
	}
	if cfg.Issuer != "http://localhost:8080" {
		t.Errorf("Issuer = %q, want http://localhost:8080", cfg.Issuer)
	}
	if cfg.AccessTokenTTL != 900*time.Second {
		t.Errorf("AccessTokenTTL = %v, want 900s", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 604800*time.Second {
		t.Errorf("RefreshTokenTTL = %v, want 604800s", cfg.RefreshTokenTTL)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("RAMPART_PORT", "9090")
	t.Setenv("RAMPART_DB_URL", "postgres://custom:secret@db:5432/custom")
	t.Setenv("RAMPART_LOG_LEVEL", "debug")
	t.Setenv("RAMPART_ACCESS_TOKEN_TTL", "300")
	t.Setenv("RAMPART_REFRESH_TOKEN_TTL", "86400")
	t.Setenv("RAMPART_SIGNING_KEY_PATH", "/etc/rampart/key.pem")
	t.Setenv("RAMPART_ISSUER", "https://auth.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://custom:secret@db:5432/custom" {
		t.Errorf("DatabaseURL = %q, want custom", cfg.DatabaseURL)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.AccessTokenTTL != 300*time.Second {
		t.Errorf("AccessTokenTTL = %v, want 300s", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 86400*time.Second {
		t.Errorf("RefreshTokenTTL = %v, want 86400s", cfg.RefreshTokenTTL)
	}
	if cfg.SigningKeyPath != "/etc/rampart/key.pem" {
		t.Errorf("SigningKeyPath = %q, want /etc/rampart/key.pem", cfg.SigningKeyPath)
	}
	if cfg.Issuer != "https://auth.example.com" {
		t.Errorf("Issuer = %q, want https://auth.example.com", cfg.Issuer)
	}
}

func TestLoadMissingDatabaseURL(t *testing.T) {
	t.Setenv("RAMPART_DB_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when RAMPART_DB_URL is not set")
	}
}

func TestLoadInvalidPort(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("RAMPART_PORT", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestLoadPortOutOfRange(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("RAMPART_PORT", "99999")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for out-of-range port")
	}
}

func TestLoadInvalidAccessTokenTTL(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("RAMPART_ACCESS_TOKEN_TTL", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid access token TTL")
	}
}

func TestLoadInvalidRefreshTokenTTL(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("RAMPART_REFRESH_TOKEN_TTL", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for zero refresh token TTL")
	}
}

func TestLoadAllowedOrigins(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("RAMPART_ALLOWED_ORIGINS", "http://localhost:3000, https://app.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("AllowedOrigins length = %d, want 2", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("AllowedOrigins[0] = %q, want http://localhost:3000", cfg.AllowedOrigins[0])
	}
	if cfg.AllowedOrigins[1] != "https://app.example.com" {
		t.Errorf("AllowedOrigins[1] = %q, want https://app.example.com", cfg.AllowedOrigins[1])
	}
}

func TestLoadAllowedOriginsDefault(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AllowedOrigins != nil {
		t.Errorf("AllowedOrigins = %v, want nil (no cross-origin by default)", cfg.AllowedOrigins)
	}
}

func TestLoadSocialProviderConfig(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("RAMPART_GOOGLE_CLIENT_ID", "google-id-123")
	t.Setenv("RAMPART_GOOGLE_CLIENT_SECRET", "google-secret-456")
	t.Setenv("RAMPART_GITHUB_CLIENT_ID", "github-id-789")
	t.Setenv("RAMPART_GITHUB_CLIENT_SECRET", "github-secret-012")
	t.Setenv("RAMPART_APPLE_CLIENT_ID", "com.example.app")
	t.Setenv("RAMPART_APPLE_TEAM_ID", "TEAM123")
	t.Setenv("RAMPART_APPLE_KEY_ID", "KEY456")
	t.Setenv("RAMPART_APPLE_PRIVATE_KEY", "-----BEGIN PRIVATE KEY-----\nfake\n-----END PRIVATE KEY-----")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GoogleClientID != "google-id-123" {
		t.Errorf("GoogleClientID = %q, want google-id-123", cfg.GoogleClientID)
	}
	if cfg.GoogleClientSecret != "google-secret-456" {
		t.Errorf("GoogleClientSecret = %q, want google-secret-456", cfg.GoogleClientSecret)
	}
	if cfg.GitHubClientID != "github-id-789" {
		t.Errorf("GitHubClientID = %q, want github-id-789", cfg.GitHubClientID)
	}
	if cfg.GitHubClientSecret != "github-secret-012" {
		t.Errorf("GitHubClientSecret = %q, want github-secret-012", cfg.GitHubClientSecret)
	}
	if cfg.AppleClientID != "com.example.app" {
		t.Errorf("AppleClientID = %q, want com.example.app", cfg.AppleClientID)
	}
	if cfg.AppleTeamID != "TEAM123" {
		t.Errorf("AppleTeamID = %q, want TEAM123", cfg.AppleTeamID)
	}
	if cfg.AppleKeyID != "KEY456" {
		t.Errorf("AppleKeyID = %q, want KEY456", cfg.AppleKeyID)
	}
	if cfg.ApplePrivateKey != "-----BEGIN PRIVATE KEY-----\nfake\n-----END PRIVATE KEY-----" {
		t.Errorf("ApplePrivateKey = %q, want PEM key", cfg.ApplePrivateKey)
	}
}

func TestLoadSocialProviderConfigEmpty(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GoogleClientID != "" {
		t.Errorf("GoogleClientID = %q, want empty", cfg.GoogleClientID)
	}
	if cfg.GoogleClientSecret != "" {
		t.Errorf("GoogleClientSecret = %q, want empty", cfg.GoogleClientSecret)
	}
	if cfg.GitHubClientID != "" {
		t.Errorf("GitHubClientID = %q, want empty", cfg.GitHubClientID)
	}
	if cfg.GitHubClientSecret != "" {
		t.Errorf("GitHubClientSecret = %q, want empty", cfg.GitHubClientSecret)
	}
	if cfg.AppleClientID != "" {
		t.Errorf("AppleClientID = %q, want empty", cfg.AppleClientID)
	}
	if cfg.AppleTeamID != "" {
		t.Errorf("AppleTeamID = %q, want empty", cfg.AppleTeamID)
	}
	if cfg.AppleKeyID != "" {
		t.Errorf("AppleKeyID = %q, want empty", cfg.AppleKeyID)
	}
	if cfg.ApplePrivateKey != "" {
		t.Errorf("ApplePrivateKey = %q, want empty", cfg.ApplePrivateKey)
	}
}

func TestLoadSecureCookiesTrue(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("RAMPART_SECURE_COOKIES", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.SecureCookies {
		t.Error("SecureCookies = false, want true")
	}
}

func TestLoadSecureCookiesFalse(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("RAMPART_SECURE_COOKIES", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SecureCookies {
		t.Error("SecureCookies = true, want false")
	}
}

func TestLoadSecureCookiesDefaultFalse(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SecureCookies {
		t.Error("SecureCookies should default to false")
	}
}

func TestLoadSecureCookiesInvalid(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("RAMPART_SECURE_COOKIES", "maybe")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid RAMPART_SECURE_COOKIES value")
	}
}

func TestLoadSecureCookiesAlternateValues(t *testing.T) {
	setRequiredEnv(t)

	for _, val := range []string{"1", "yes", "YES"} {
		t.Setenv("RAMPART_SECURE_COOKIES", val)
		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", val, err)
		}
		if !cfg.SecureCookies {
			t.Errorf("SecureCookies = false for %q, want true", val)
		}
	}

	for _, val := range []string{"0", "no", "NO"} {
		t.Setenv("RAMPART_SECURE_COOKIES", val)
		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", val, err)
		}
		if cfg.SecureCookies {
			t.Errorf("SecureCookies = true for %q, want false", val)
		}
	}
}

func TestConfigAddr(t *testing.T) {
	cfg := &Config{Port: 3000}
	if got := cfg.Addr(); got != ":3000" {
		t.Errorf("Addr() = %q, want :3000", got)
	}
}
