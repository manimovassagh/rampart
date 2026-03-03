package session

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Session represents a row in the sessions table.
type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash []byte
	ExpiresAt        time.Time
	CreatedAt        time.Time
}

// Store defines operations for managing sessions.
type Store interface {
	Create(ctx context.Context, userID uuid.UUID, refreshToken string, expiresAt time.Time) (*Session, error)
	FindByRefreshToken(ctx context.Context, refreshToken string) (*Session, error)
	Delete(ctx context.Context, sessionID uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

// PGStore implements Store using PostgreSQL.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore creates a new PostgreSQL session store.
func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

// HashToken returns the SHA-256 hash of a refresh token.
func HashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// Create inserts a new session with a hashed refresh token.
func (s *PGStore) Create(ctx context.Context, userID uuid.UUID, refreshToken string, expiresAt time.Time) (*Session, error) {
	hash := HashToken(refreshToken)

	query := `
		INSERT INTO sessions (user_id, refresh_token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, refresh_token_hash, expires_at, created_at`

	var sess Session
	err := s.pool.QueryRow(ctx, query, userID, hash, expiresAt).Scan(
		&sess.ID, &sess.UserID, &sess.RefreshTokenHash, &sess.ExpiresAt, &sess.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}
	return &sess, nil
}

// FindByRefreshToken looks up a session by hashing the provided token.
func (s *PGStore) FindByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	hash := HashToken(refreshToken)

	query := `
		SELECT id, user_id, refresh_token_hash, expires_at, created_at
		FROM sessions
		WHERE refresh_token_hash = $1 AND expires_at > now()`

	var sess Session
	err := s.pool.QueryRow(ctx, query, hash).Scan(
		&sess.ID, &sess.UserID, &sess.RefreshTokenHash, &sess.ExpiresAt, &sess.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding session by refresh token: %w", err)
	}
	return &sess, nil
}

// Delete removes a session by ID.
func (s *PGStore) Delete(ctx context.Context, sessionID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM sessions WHERE id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

// DeleteByUserID removes all sessions for a user.
func (s *PGStore) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("deleting sessions by user: %w", err)
	}
	return nil
}
