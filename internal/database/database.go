package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/manimovassagh/rampart/internal/crypto"
)

const (
	defaultMaxConns = 25
	defaultMinConns = 2
	connectTimeout  = 10 * time.Second

	// defaultQueryTimeout is the maximum duration a single database query should
	// take before being cancelled. This prevents runaway queries from holding
	// connections indefinitely.
	defaultQueryTimeout = 5 * time.Second

	// pgUniqueViolation is the PostgreSQL error code for unique constraint violations.
	pgUniqueViolation = "23505"
)

// queryCtx derives a child context with the default query timeout.
// Callers must call the returned cancel function when done (typically via defer).
func queryCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultQueryTimeout)
}

// DB wraps a pgx connection pool.
type DB struct {
	Pool      *pgxpool.Pool
	Encryptor *crypto.Encryptor // nil means no encryption (plaintext fallback)
}

// Connect creates a new connection pool and verifies connectivity.
func Connect(ctx context.Context, databaseURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	config.MaxConns = defaultMaxConns
	config.MinConns = defaultMinConns

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Ping checks database connectivity.
func (db *DB) Ping(ctx context.Context) error {
	if db.Pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}
	return db.Pool.Ping(ctx)
}

// Close closes the connection pool.
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}
