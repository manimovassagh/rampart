package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all server configuration loaded from environment variables.
type Config struct {
	Port        int
	DatabaseURL string
	RedisURL    string
	LogLevel    string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Port:        8080,
		DatabaseURL: "postgres://rampart:rampart@localhost:5432/rampart?sslmode=disable",
		RedisURL:    "redis://localhost:6379/0",
		LogLevel:    "info",
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

	if v := os.Getenv("RAMPART_DB_URL"); v != "" {
		cfg.DatabaseURL = v
	}

	if v := os.Getenv("RAMPART_REDIS_URL"); v != "" {
		cfg.RedisURL = v
	}

	if v := os.Getenv("RAMPART_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	return cfg, nil
}

// Addr returns the listen address string (e.g., ":8080").
func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}
