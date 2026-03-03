package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might be set
	os.Unsetenv("RAMPART_PORT")
	os.Unsetenv("RAMPART_DB_URL")
	os.Unsetenv("RAMPART_REDIS_URL")
	os.Unsetenv("RAMPART_LOG_LEVEL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://rampart:rampart@localhost:5432/rampart?sslmode=disable" {
		t.Errorf("DatabaseURL = %q, want default", cfg.DatabaseURL)
	}
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("RedisURL = %q, want default", cfg.RedisURL)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("RAMPART_PORT", "9090")
	t.Setenv("RAMPART_DB_URL", "postgres://custom:custom@db:5432/custom")
	t.Setenv("RAMPART_REDIS_URL", "redis://cache:6379/1")
	t.Setenv("RAMPART_LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://custom:custom@db:5432/custom" {
		t.Errorf("DatabaseURL = %q, want custom", cfg.DatabaseURL)
	}
	if cfg.RedisURL != "redis://cache:6379/1" {
		t.Errorf("RedisURL = %q, want custom", cfg.RedisURL)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("RAMPART_PORT", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestLoad_PortOutOfRange(t *testing.T) {
	t.Setenv("RAMPART_PORT", "99999")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for out-of-range port")
	}
}

func TestConfig_Addr(t *testing.T) {
	cfg := &Config{Port: 3000}
	if got := cfg.Addr(); got != ":3000" {
		t.Errorf("Addr() = %q, want :3000", got)
	}
}
