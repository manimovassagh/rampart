package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UserConsent represents a stored user consent grant for an OAuth client.
type UserConsent struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ClientID  string
	Scopes    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// HasConsent checks if a user has already granted consent for a client with the given scopes.
func (db *DB) HasConsent(ctx context.Context, userID uuid.UUID, clientID, scopes string) (bool, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	var exists bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_consents WHERE user_id = $1 AND client_id = $2 AND scopes = $3)`,
		userID, clientID, scopes,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking consent: %w", err)
	}
	return exists, nil
}

// GrantConsent stores or updates a user's consent for an OAuth client.
func (db *DB) GrantConsent(ctx context.Context, userID uuid.UUID, clientID, scopes string) error {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	_, err := db.Pool.Exec(ctx,
		`INSERT INTO user_consents (user_id, client_id, scopes)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, client_id)
		 DO UPDATE SET scopes = $3, updated_at = now()`,
		userID, clientID, scopes,
	)
	if err != nil {
		return fmt.Errorf("granting consent: %w", err)
	}
	return nil
}

// RevokeConsent removes a user's consent for an OAuth client.
func (db *DB) RevokeConsent(ctx context.Context, userID uuid.UUID, clientID string) error {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	_, err := db.Pool.Exec(ctx,
		`DELETE FROM user_consents WHERE user_id = $1 AND client_id = $2`,
		userID, clientID,
	)
	if err != nil {
		return fmt.Errorf("revoking consent: %w", err)
	}
	return nil
}

// ListUserConsents returns all active consents for a user.
func (db *DB) ListUserConsents(ctx context.Context, userID uuid.UUID) ([]UserConsent, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	rows, err := db.Pool.Query(ctx,
		`SELECT id, user_id, client_id, scopes, created_at, updated_at
		 FROM user_consents WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing user consents: %w", err)
	}
	defer rows.Close()

	var consents []UserConsent
	for rows.Next() {
		var c UserConsent
		if err := rows.Scan(&c.ID, &c.UserID, &c.ClientID, &c.Scopes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning consent row: %w", err)
		}
		consents = append(consents, c)
	}
	return consents, rows.Err()
}

// GetConsent retrieves a specific consent record.
func (db *DB) GetConsent(ctx context.Context, userID uuid.UUID, clientID string) (*UserConsent, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	var c UserConsent
	err := db.Pool.QueryRow(ctx,
		`SELECT id, user_id, client_id, scopes, created_at, updated_at
		 FROM user_consents WHERE user_id = $1 AND client_id = $2`,
		userID, clientID,
	).Scan(&c.ID, &c.UserID, &c.ClientID, &c.Scopes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting consent: %w", err)
	}
	return &c, nil
}
