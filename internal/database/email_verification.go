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

// CreateEmailVerificationToken stores a hashed verification token.
// The plaintext token is NOT stored — only its SHA-256 hash.
// Any existing unused tokens for the user are invalidated.
func (db *DB) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenPlaintext string, expiresAt time.Time) error {
	ctx, cancel := txCtx(ctx)
	defer cancel()

	hash := sha256.Sum256([]byte(tokenPlaintext))

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback best-effort on deferred cleanup

	// Invalidate any existing unused tokens for this user
	_, err = tx.Exec(ctx, `UPDATE email_verification_tokens SET used = true WHERE user_id = $1 AND used = false`, userID)
	if err != nil {
		return fmt.Errorf("invalidating old verification tokens: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO email_verification_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, hash[:], expiresAt,
	)
	if err != nil {
		return fmt.Errorf("inserting email verification token: %w", err)
	}
	return tx.Commit(ctx)
}

// ConsumeEmailVerificationToken validates and consumes an email verification token.
// Returns the user ID if valid, or an error if expired/used/not found.
func (db *DB) ConsumeEmailVerificationToken(ctx context.Context, tokenPlaintext string) (uuid.UUID, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	hash := sha256.Sum256([]byte(tokenPlaintext))

	var userID uuid.UUID
	err := db.Pool.QueryRow(ctx,
		`UPDATE email_verification_tokens SET used = true
		 WHERE token_hash = $1 AND used = false AND expires_at > now()
		 RETURNING user_id`,
		hash[:],
	).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("invalid, expired, or already-used verification token")
		}
		return uuid.Nil, fmt.Errorf("consuming email verification token: %w", err)
	}
	return userID, nil
}

// MarkEmailVerified sets email_verified = true for the given user.
func (db *DB) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	_, err := db.Pool.Exec(ctx,
		`UPDATE users SET email_verified = true, updated_at = now() WHERE id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("marking email verified: %w", err)
	}
	return nil
}

// DeleteExpiredEmailVerificationTokens removes expired or consumed tokens.
func (db *DB) DeleteExpiredEmailVerificationTokens(ctx context.Context) (int64, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	tag, err := db.Pool.Exec(ctx,
		`DELETE FROM email_verification_tokens WHERE expires_at < now() OR (used = true AND created_at < now() - interval '1 day')`,
	)
	if err != nil {
		return 0, fmt.Errorf("deleting expired email verification tokens: %w", err)
	}
	return tag.RowsAffected(), nil
}
