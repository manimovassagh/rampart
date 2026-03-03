package config

import (
	"testing"
	"time"
)

const testJWTSecret = "this-is-a-test-secret-that-is-at-least-32-bytes-long"

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("RAMPART_DB_URL", "postgres://localhost:5432/rampart")
	t.Setenv("RAMPART_JWT_SECRET", testJWTSecret)
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
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("RedisURL = %q, want default", cfg.RedisURL)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.JWTSecret != testJWTSecret {
		t.Errorf("JWTSecret = %q, want test secret", cfg.JWTSecret)
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
	t.Setenv("RAMPART_REDIS_URL", "redis://cache:6379/1")
	t.Setenv("RAMPART_LOG_LEVEL", "debug")
	t.Setenv("RAMPART_JWT_SECRET", testJWTSecret)
	t.Setenv("RAMPART_ACCESS_TOKEN_TTL", "300")
	t.Setenv("RAMPART_REFRESH_TOKEN_TTL", "86400")

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
	if cfg.RedisURL != "redis://cache:6379/1" {
		t.Errorf("RedisURL = %q, want custom", cfg.RedisURL)
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
}

func TestLoadMissingDatabaseURL(t *testing.T) {
	t.Setenv("RAMPART_DB_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when RAMPART_DB_URL is not set")
	}
}

func TestLoadMissingJWTSecret(t *testing.T) {
	t.Setenv("RAMPART_DB_URL", "postgres://localhost:5432/rampart")
	t.Setenv("RAMPART_JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when RAMPART_JWT_SECRET is not set")
	}
}

func TestLoadJWTSecretTooShort(t *testing.T) {
	t.Setenv("RAMPART_DB_URL", "postgres://localhost:5432/rampart")
	t.Setenv("RAMPART_JWT_SECRET", "short")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when RAMPART_JWT_SECRET is too short")
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

func TestConfigAddr(t *testing.T) {
	cfg := &Config{Port: 3000}
	if got := cfg.Addr(); got != ":3000" {
		t.Errorf("Addr() = %q, want :3000", got)
	}
}
