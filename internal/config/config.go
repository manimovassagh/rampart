package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAccessTokenTTL  = 900     // 15 minutes
	defaultRefreshTokenTTL = 604800  // 7 days
	minJWTSecretLength     = 32
)

// Config holds all server configuration loaded from environment variables.
type Config struct {
	Port            int
	DatabaseURL     string
	RedisURL        string
	LogLevel        string
	AllowedOrigins  []string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Port:     8080,
		RedisURL: "redis://localhost:6379/0",
		LogLevel: "info",
	}

	if v := os.Getenv("RAMPART_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid RAMPART_PORT %q: %w", v, err)
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("RAMPART_PORT %d out of range (1-65535)", port)
		}
		cfg.Port = port
	}

	cfg.DatabaseURL = os.Getenv("RAMPART_DB_URL")

	if v := os.Getenv("RAMPART_REDIS_URL"); v != "" {
		cfg.RedisURL = v
	}

	if v := os.Getenv("RAMPART_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	if v := os.Getenv("RAMPART_ALLOWED_ORIGINS"); v != "" {
		origins := strings.Split(v, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		cfg.AllowedOrigins = origins
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("RAMPART_DB_URL is required")
	}

	cfg.JWTSecret = os.Getenv("RAMPART_JWT_SECRET")
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("RAMPART_JWT_SECRET is required")
	}
	if len(cfg.JWTSecret) < minJWTSecretLength {
		return nil, fmt.Errorf("RAMPART_JWT_SECRET must be at least %d bytes", minJWTSecretLength)
	}

	cfg.AccessTokenTTL = time.Duration(defaultAccessTokenTTL) * time.Second
	if v := os.Getenv("RAMPART_ACCESS_TOKEN_TTL"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid RAMPART_ACCESS_TOKEN_TTL %q: %w", v, err)
		}
		if secs < 1 {
			return nil, fmt.Errorf("RAMPART_ACCESS_TOKEN_TTL must be positive")
		}
		cfg.AccessTokenTTL = time.Duration(secs) * time.Second
	}

	cfg.RefreshTokenTTL = time.Duration(defaultRefreshTokenTTL) * time.Second
	if v := os.Getenv("RAMPART_REFRESH_TOKEN_TTL"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid RAMPART_REFRESH_TOKEN_TTL %q: %w", v, err)
		}
		if secs < 1 {
			return nil, fmt.Errorf("RAMPART_REFRESH_TOKEN_TTL must be positive")
		}
		cfg.RefreshTokenTTL = time.Duration(secs) * time.Second
	}

	return cfg, nil
}

// Addr returns the listen address string (e.g., ":8080").
func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}
