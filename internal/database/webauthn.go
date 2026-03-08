package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// CreateWebAuthnCredential inserts a new WebAuthn credential.
func (db *DB) CreateWebAuthnCredential(ctx context.Context, cred *model.WebAuthnCredential) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO webauthn_credentials (user_id, credential_id, public_key, attestation_type, transport, flags_raw, aaguid, sign_count, name)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		cred.UserID, cred.CredentialID, cred.PublicKey, cred.AttestationType,
		cred.Transport, cred.FlagsRaw, cred.AAGUID, cred.SignCount, cred.Name,
	)
	if err != nil {
		return fmt.Errorf("creating webauthn credential: %w", err)
	}
	return nil
}

// GetWebAuthnCredentialsByUserID returns all WebAuthn credentials for a user.
func (db *DB) GetWebAuthnCredentialsByUserID(ctx context.Context, userID uuid.UUID) ([]*model.WebAuthnCredential, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, user_id, credential_id, public_key, attestation_type, transport, flags_raw, aaguid, sign_count, name, created_at, updated_at
		 FROM webauthn_credentials WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("getting webauthn credentials: %w", err)
	}
	defer rows.Close()

	var creds []*model.WebAuthnCredential
	for rows.Next() {
		var c model.WebAuthnCredential
		if err := rows.Scan(&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey,
			&c.AttestationType, &c.Transport, &c.FlagsRaw, &c.AAGUID,
			&c.SignCount, &c.Name, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning webauthn credential: %w", err)
		}
		creds = append(creds, &c)
	}
	return creds, nil
}

// UpdateWebAuthnSignCount updates the sign counter for a credential after successful authentication.
func (db *DB) UpdateWebAuthnSignCount(ctx context.Context, credentialID []byte, signCount uint32) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE webauthn_credentials SET sign_count = $2, updated_at = now() WHERE credential_id = $1`,
		credentialID, signCount,
	)
	return err
}

// DeleteWebAuthnCredential deletes a credential by ID for a given user.
func (db *DB) DeleteWebAuthnCredential(ctx context.Context, id, userID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		`DELETE FROM webauthn_credentials WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	return err
}

// CountWebAuthnCredentials returns the number of WebAuthn credentials for a user.
func (db *DB) CountWebAuthnCredentials(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM webauthn_credentials WHERE user_id = $1`, userID,
	).Scan(&count)
	return count, err
}

// StoreWebAuthnSessionData stores temporary session data for a WebAuthn ceremony.
func (db *DB) StoreWebAuthnSessionData(ctx context.Context, userID uuid.UUID, data []byte, ceremony string, expiresAt time.Time) error {
	// Clean up any previous session data for this user+ceremony
	_, _ = db.Pool.Exec(ctx,
		`DELETE FROM webauthn_session_data WHERE user_id = $1 AND ceremony = $2`,
		userID, ceremony,
	)

	_, err := db.Pool.Exec(ctx,
		`INSERT INTO webauthn_session_data (user_id, session_data, ceremony, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		userID, data, ceremony, expiresAt,
	)
	return err
}

// GetWebAuthnSessionData retrieves and deletes the session data for a ceremony (one-time use).
func (db *DB) GetWebAuthnSessionData(ctx context.Context, userID uuid.UUID, ceremony string) ([]byte, error) {
	var data []byte
	err := db.Pool.QueryRow(ctx,
		`DELETE FROM webauthn_session_data
		 WHERE user_id = $1 AND ceremony = $2 AND expires_at > now()
		 RETURNING session_data`,
		userID, ceremony,
	).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("getting webauthn session data: %w", err)
	}
	return data, nil
}

// DeleteExpiredWebAuthnSessions cleans up expired ceremony sessions.
func (db *DB) DeleteExpiredWebAuthnSessions(ctx context.Context) (int64, error) {
	tag, err := db.Pool.Exec(ctx,
		`DELETE FROM webauthn_session_data WHERE expires_at < now()`,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
