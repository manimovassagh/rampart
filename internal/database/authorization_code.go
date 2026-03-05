package database

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/manimovassagh/rampart/internal/model"
)

// StoreAuthorizationCode persists an authorization code (as SHA-256 hash) in the database.
func (db *DB) StoreAuthorizationCode(ctx context.Context, code, clientID string, userID, orgID uuid.UUID, redirectURI, codeChallenge, scope string, expiresAt time.Time) error {
	hash := sha256.Sum256([]byte(code))
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO authorization_codes (code_hash, client_id, user_id, org_id, redirect_uri, code_challenge, scope, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		hash[:], clientID, userID, orgID, redirectURI, codeChallenge, scope, expiresAt)
	if err != nil {
		return fmt.Errorf("storing authorization code: %w", err)
	}
	return nil
}

// ConsumeAuthorizationCode atomically marks an authorization code as used and returns it.
// Returns nil if the code doesn't exist, is already used, or is expired.
func (db *DB) ConsumeAuthorizationCode(ctx context.Context, code string) (*model.AuthorizationCode, error) {
	hash := sha256.Sum256([]byte(code))
	row := db.Pool.QueryRow(ctx, `
		UPDATE authorization_codes
		SET used = true
		WHERE code_hash = $1 AND used = false AND expires_at > now()
		RETURNING id, code_hash, client_id, user_id, org_id, redirect_uri, code_challenge, scope, used, expires_at, created_at`,
		hash[:])

	var ac model.AuthorizationCode
	err := row.Scan(&ac.ID, &ac.CodeHash, &ac.ClientID, &ac.UserID, &ac.OrgID,
		&ac.RedirectURI, &ac.CodeChallenge, &ac.Scope, &ac.Used, &ac.ExpiresAt, &ac.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("consuming authorization code: %w", err)
	}
	return &ac, nil
}
