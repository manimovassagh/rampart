package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAccessTokenTTL  = 900    // 15 minutes
	defaultRefreshTokenTTL = 604800 // 7 days
	defaultSigningKeyPath  = "rampart-signing-key.pem"
	defaultIssuer          = "http://localhost:8080"
)

// Config holds all server configuration loaded from environment variables.
type Config struct {
	Port            int
	DatabaseURL     string
	RedisURL        string
	LogLevel        string
	AllowedOrigins  []string
	SigningKeyPath  string
	Issuer          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// Social login providers
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	AppleClientID      string
	AppleTeamID        string
	AppleKeyID         string
	ApplePrivateKey    string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Port:           8080,
		RedisURL:       "redis://localhost:6379/0",
		LogLevel:       "info",
		SigningKeyPath: defaultSigningKeyPath,
		Issuer:         defaultIssuer,
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

	if v := os.Getenv("RAMPART_SIGNING_KEY_PATH"); v != "" {
		cfg.SigningKeyPath = v
	}

	if v := os.Getenv("RAMPART_ISSUER"); v != "" {
		cfg.Issuer = v
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

	// Social login providers
	cfg.GoogleClientID = os.Getenv("RAMPART_GOOGLE_CLIENT_ID")
	cfg.GoogleClientSecret = os.Getenv("RAMPART_GOOGLE_CLIENT_SECRET")
	cfg.GitHubClientID = os.Getenv("RAMPART_GITHUB_CLIENT_ID")
	cfg.GitHubClientSecret = os.Getenv("RAMPART_GITHUB_CLIENT_SECRET")
	cfg.AppleClientID = os.Getenv("RAMPART_APPLE_CLIENT_ID")
	cfg.AppleTeamID = os.Getenv("RAMPART_APPLE_TEAM_ID")
	cfg.AppleKeyID = os.Getenv("RAMPART_APPLE_KEY_ID")
	cfg.ApplePrivateKey = os.Getenv("RAMPART_APPLE_PRIVATE_KEY")

	return cfg, nil
}

// Addr returns the listen address string (e.g., ":8080").
func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}
