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
	ClientID         string
	RefreshTokenHash []byte
	Scope            string
	ExpiresAt        time.Time
	CreatedAt        time.Time
}

// WithUser enriches a Session with user information for global views.
type WithUser struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Username  string
	Email     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// ErrTokenAlreadyRotated is returned when a refresh token has already been
// consumed by a concurrent request (race condition / replay detected).
var ErrTokenAlreadyRotated = errors.New("refresh token already rotated")

// Store defines operations for managing sessions.
type Store interface {
	Create(ctx context.Context, userID uuid.UUID, clientID string, refreshToken string, scope string, expiresAt time.Time) (*Session, error)
	FindByRefreshToken(ctx context.Context, refreshToken string) (*Session, error)
	RotateRefreshToken(ctx context.Context, oldRefreshToken, newRefreshToken string) (*Session, error)
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
func (s *PGStore) Create(ctx context.Context, userID uuid.UUID, clientID, refreshToken, scope string, expiresAt time.Time) (*Session, error) {
	hash := HashToken(refreshToken)

	query := `
		INSERT INTO sessions (user_id, client_id, refresh_token_hash, scope, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, client_id, refresh_token_hash, scope, expires_at, created_at`

	var sess Session
	err := s.pool.QueryRow(ctx, query, userID, clientID, hash, scope, expiresAt).Scan(
		&sess.ID, &sess.UserID, &sess.ClientID, &sess.RefreshTokenHash, &sess.Scope, &sess.ExpiresAt, &sess.CreatedAt,
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
		SELECT id, user_id, client_id, refresh_token_hash, scope, expires_at, created_at
		FROM sessions
		WHERE refresh_token_hash = $1 AND expires_at > now()`

	var sess Session
	err := s.pool.QueryRow(ctx, query, hash).Scan(
		&sess.ID, &sess.UserID, &sess.ClientID, &sess.RefreshTokenHash, &sess.Scope, &sess.ExpiresAt, &sess.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding session by refresh token: %w", err)
	}
	return &sess, nil
}

// RotateRefreshToken atomically replaces the refresh token hash using the old
// token hash as the WHERE condition. This prevents race conditions where two
// concurrent requests try to rotate the same token. If 0 rows are affected
// the token was already consumed and ErrTokenAlreadyRotated is returned.
func (s *PGStore) RotateRefreshToken(ctx context.Context, oldRefreshToken, newRefreshToken string) (*Session, error) {
	oldHash := HashToken(oldRefreshToken)
	newHash := HashToken(newRefreshToken)

	query := `
		UPDATE sessions SET refresh_token_hash = $1
		WHERE refresh_token_hash = $2 AND expires_at > now()
		RETURNING id, user_id, client_id, refresh_token_hash, scope, expires_at, created_at`

	var sess Session
	err := s.pool.QueryRow(ctx, query, newHash, oldHash).Scan(
		&sess.ID, &sess.UserID, &sess.ClientID, &sess.RefreshTokenHash, &sess.Scope, &sess.ExpiresAt, &sess.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenAlreadyRotated
		}
		return nil, fmt.Errorf("rotating refresh token: %w", err)
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

// ListByUserID returns all active sessions for a user.
func (s *PGStore) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	query := `
		SELECT id, user_id, client_id, refresh_token_hash, scope, expires_at, created_at
		FROM sessions
		WHERE user_id = $1 AND expires_at > now()
		ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing sessions by user: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.UserID, &sess.ClientID, &sess.RefreshTokenHash, &sess.Scope, &sess.ExpiresAt, &sess.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning session row: %w", err)
		}
		sessions = append(sessions, &sess)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating session rows: %w", err)
	}
	return sessions, nil
}

// CountByUserID returns the number of active sessions for a user.
func (s *PGStore) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM sessions WHERE user_id = $1 AND expires_at > now()",
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting sessions by user: %w", err)
	}
	return count, nil
}

// CountActive returns the total number of active sessions for an organization.
func (s *PGStore) CountActive(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.expires_at > now() AND u.org_id = $1`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting active sessions: %w", err)
	}
	return count, nil
}

// ListAll returns all active sessions with user info for an organization, paginated and searchable.
func (s *PGStore) ListAll(ctx context.Context, orgID uuid.UUID, search string, limit, offset int) ([]*WithUser, int, error) {
	baseWhere := "s.expires_at > now() AND u.org_id = $1"
	args := []any{orgID}
	paramIdx := 2

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (u.username ILIKE $%d OR u.email ILIKE $%d)", paramIdx, paramIdx)
		args = append(args, "%"+search+"%")
		paramIdx++
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM sessions s JOIN users u ON u.id = s.user_id WHERE %s", baseWhere)
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting all sessions: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT s.id, s.user_id, u.username, u.email, s.expires_at, s.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE %s
		ORDER BY s.created_at DESC
		LIMIT $%d OFFSET $%d`, baseWhere, paramIdx, paramIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing all sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*WithUser
	for rows.Next() {
		var sess WithUser
		if err := rows.Scan(&sess.ID, &sess.UserID, &sess.Username, &sess.Email, &sess.ExpiresAt, &sess.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning session with user row: %w", err)
		}
		sessions = append(sessions, &sess)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating session with user rows: %w", err)
	}
	return sessions, total, nil
}

// DeleteAll removes all active sessions for an organization.
func (s *PGStore) DeleteAll(ctx context.Context, orgID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM sessions WHERE expires_at > now()
		 AND user_id IN (SELECT id FROM users WHERE org_id = $1)`, orgID)
	if err != nil {
		return fmt.Errorf("deleting all sessions: %w", err)
	}
	return nil
}
