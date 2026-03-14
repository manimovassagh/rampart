package database

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PasswordResetToken represents a stored password reset token.
type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

// CreatePasswordResetToken stores a hashed token and returns the record.
// The plaintext token is NOT stored — only its SHA-256 hash.
func (db *DB) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenPlaintext string, expiresAt time.Time) error {
	ctx, cancel := txCtx(ctx)
	defer cancel()

	hash := sha256.Sum256([]byte(tokenPlaintext))

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback best-effort on deferred cleanup

	// Invalidate any existing unused tokens for this user
	_, err = tx.Exec(ctx, `UPDATE password_reset_tokens SET used = true WHERE user_id = $1 AND used = false`, userID)
	if err != nil {
		return fmt.Errorf("invalidating old reset tokens: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, hash[:], expiresAt,
	)
	if err != nil {
		return fmt.Errorf("inserting password reset token: %w", err)
	}
	return tx.Commit(ctx)
}

// ConsumePasswordResetToken validates and consumes a password reset token.
// Returns the user ID if valid, or an error if expired/used/not found.
func (db *DB) ConsumePasswordResetToken(ctx context.Context, tokenPlaintext string) (uuid.UUID, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	hash := sha256.Sum256([]byte(tokenPlaintext))

	var token PasswordResetToken
	err := db.Pool.QueryRow(ctx,
		`UPDATE password_reset_tokens SET used = true
		 WHERE token_hash = $1 AND used = false AND expires_at > now()
		 RETURNING id, user_id, expires_at, used, created_at`,
		hash[:],
	).Scan(&token.ID, &token.UserID, &token.ExpiresAt, &token.Used, &token.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("invalid, expired, or already-used reset token")
		}
		return uuid.Nil, fmt.Errorf("consuming password reset token: %w", err)
	}
	return token.UserID, nil
}

// DeleteExpiredPasswordResetTokens removes tokens older than the given age.
func (db *DB) DeleteExpiredPasswordResetTokens(ctx context.Context) (int64, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	tag, err := db.Pool.Exec(ctx,
		`DELETE FROM password_reset_tokens WHERE expires_at < now() OR (used = true AND created_at < now() - interval '1 day')`,
	)
	if err != nil {
		return 0, fmt.Errorf("deleting expired password reset tokens: %w", err)
	}
	return tag.RowsAffected(), nil
}
