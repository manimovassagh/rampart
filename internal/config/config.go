package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// LogFormat defines the output format for structured logging.
type LogFormat string

const (
	// LogFormatPretty outputs colorized human-friendly logs (default).
	LogFormatPretty LogFormat = "pretty"
	// LogFormatText outputs plain key=value logs.
	LogFormatText LogFormat = "text"
	// LogFormatJSON outputs machine-readable JSON logs.
	LogFormatJSON LogFormat = "json"
)

// LogLevel defines the minimum severity for log output.
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

const (
	defaultAccessTokenTTL  = 900    // 15 minutes
	defaultRefreshTokenTTL = 604800 // 7 days
	defaultSigningKeyPath  = "rampart-signing-key.pem"
	defaultIssuer          = "http://localhost:8080"

	defaultLoginRateLimit    = 10 // requests per minute
	defaultRegisterRateLimit = 5  // requests per minute
	defaultTokenRateLimit    = 10 // requests per minute
)

// RateLimitConfig holds rate limiting settings for auth endpoints.
type RateLimitConfig struct {
	LoginPerMinute    int
	RegisterPerMinute int
	TokenPerMinute    int
}

// Config holds all server configuration loaded from environment variables.
type Config struct {
	Port            int
	DatabaseURL     string
	RedisURL        string
	LogLevel        LogLevel
	LogFormat       LogFormat
	AllowedOrigins  []string
	SigningKeyPath  string
	Issuer          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// Security
	HSTSEnabled bool

	// SecureCookies enables the Secure flag on all cookies.
	// MUST be true in production (requires HTTPS). Defaults to false for development.
	SecureCookies bool

	// Rate limiting (requests per minute per IP)
	RateLimit RateLimitConfig

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
		LogLevel:       LogLevelInfo,
		LogFormat:      LogFormatPretty,
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
		level := LogLevel(strings.ToLower(v))
		switch level {
		case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
			cfg.LogLevel = level
		default:
			return nil, fmt.Errorf("invalid RAMPART_LOG_LEVEL %q (valid: debug, info, warn, error)", v)
		}
	}

	if v := os.Getenv("RAMPART_LOG_FORMAT"); v != "" {
		format := LogFormat(strings.ToLower(v))
		switch format {
		case LogFormatPretty, LogFormatText, LogFormatJSON:
			cfg.LogFormat = format
		default:
			return nil, fmt.Errorf("invalid RAMPART_LOG_FORMAT %q (valid: pretty, text, json)", v)
		}
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

	// Security
	if v := os.Getenv("RAMPART_HSTS_ENABLED"); strings.EqualFold(v, "true") || v == "1" {
		cfg.HSTSEnabled = true
	}

	// Rate limiting
	cfg.RateLimit = RateLimitConfig{
		LoginPerMinute:    defaultLoginRateLimit,
		RegisterPerMinute: defaultRegisterRateLimit,
		TokenPerMinute:    defaultTokenRateLimit,
	}

	if v := os.Getenv("RAMPART_RATE_LIMIT_LOGIN"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid RAMPART_RATE_LIMIT_LOGIN %q: %w", v, err)
		}
		if n < 1 {
			return nil, fmt.Errorf("RAMPART_RATE_LIMIT_LOGIN must be positive")
		}
		cfg.RateLimit.LoginPerMinute = n
	}

	if v := os.Getenv("RAMPART_RATE_LIMIT_REGISTER"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid RAMPART_RATE_LIMIT_REGISTER %q: %w", v, err)
		}
		if n < 1 {
			return nil, fmt.Errorf("RAMPART_RATE_LIMIT_REGISTER must be positive")
		}
		cfg.RateLimit.RegisterPerMinute = n
	}

	if v := os.Getenv("RAMPART_RATE_LIMIT_TOKEN"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid RAMPART_RATE_LIMIT_TOKEN %q: %w", v, err)
		}
		if n < 1 {
			return nil, fmt.Errorf("RAMPART_RATE_LIMIT_TOKEN must be positive")
		}
		cfg.RateLimit.TokenPerMinute = n
	}

	if v := os.Getenv("RAMPART_SECURE_COOKIES"); v != "" {
		switch strings.ToLower(v) {
		case "true", "1", "yes":
			cfg.SecureCookies = true
		case "false", "0", "no":
			cfg.SecureCookies = false
		default:
			return nil, fmt.Errorf("invalid RAMPART_SECURE_COOKIES %q (valid: true, false)", v)
		}
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
