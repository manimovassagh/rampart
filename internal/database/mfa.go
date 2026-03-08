package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/manimovassagh/rampart/internal/model"
)

// CreateMFADevice inserts a new unverified MFA device.
func (db *DB) CreateMFADevice(ctx context.Context, userID uuid.UUID, deviceType, name, secret string) (*model.MFADevice, error) {
	encSecret, err := db.encryptToken(secret)
	if err != nil {
		return nil, fmt.Errorf("encrypting MFA secret: %w", err)
	}

	var d model.MFADevice
	err = db.Pool.QueryRow(ctx,
		`INSERT INTO mfa_devices (user_id, device_type, name, secret)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, device_type, name, secret, verified, created_at, updated_at`,
		userID, deviceType, name, encSecret,
	).Scan(&d.ID, &d.UserID, &d.DeviceType, &d.Name, &d.Secret, &d.Verified, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting MFA device: %w", err)
	}
	if d.Secret, err = db.decryptToken(d.Secret); err != nil {
		return nil, fmt.Errorf("decrypting MFA secret: %w", err)
	}
	return &d, nil
}

// VerifyMFADevice marks a device as verified and enables MFA on the user.
func (db *DB) VerifyMFADevice(ctx context.Context, deviceID, userID uuid.UUID) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback best-effort on deferred cleanup

	_, err = tx.Exec(ctx, `UPDATE mfa_devices SET verified = true, updated_at = now() WHERE id = $1 AND user_id = $2`, deviceID, userID)
	if err != nil {
		return fmt.Errorf("verifying MFA device: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE users SET mfa_enabled = true, updated_at = now() WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("enabling MFA on user: %w", err)
	}
	return tx.Commit(ctx)
}

// GetVerifiedMFADevice returns the verified TOTP device for a user, if any.
func (db *DB) GetVerifiedMFADevice(ctx context.Context, userID uuid.UUID) (*model.MFADevice, error) {
	var d model.MFADevice
	err := db.Pool.QueryRow(ctx,
		`SELECT id, user_id, device_type, name, secret, verified, created_at, updated_at
		 FROM mfa_devices WHERE user_id = $1 AND verified = true AND device_type = 'totp'
		 LIMIT 1`,
		userID,
	).Scan(&d.ID, &d.UserID, &d.DeviceType, &d.Name, &d.Secret, &d.Verified, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting verified MFA device: %w", err)
	}
	var decErr error
	if d.Secret, decErr = db.decryptToken(d.Secret); decErr != nil {
		return nil, fmt.Errorf("decrypting MFA secret: %w", decErr)
	}
	return &d, nil
}

// GetPendingMFADevice returns the unverified TOTP device for a user (enrollment in progress).
func (db *DB) GetPendingMFADevice(ctx context.Context, userID uuid.UUID) (*model.MFADevice, error) {
	var d model.MFADevice
	err := db.Pool.QueryRow(ctx,
		`SELECT id, user_id, device_type, name, secret, verified, created_at, updated_at
		 FROM mfa_devices WHERE user_id = $1 AND verified = false AND device_type = 'totp'
		 ORDER BY created_at DESC LIMIT 1`,
		userID,
	).Scan(&d.ID, &d.UserID, &d.DeviceType, &d.Name, &d.Secret, &d.Verified, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting pending MFA device: %w", err)
	}
	var decErr error
	if d.Secret, decErr = db.decryptToken(d.Secret); decErr != nil {
		return nil, fmt.Errorf("decrypting MFA secret: %w", decErr)
	}
	return &d, nil
}

// DeleteUnverifiedMFADevices removes all unverified devices for a user.
func (db *DB) DeleteUnverifiedMFADevices(ctx context.Context, userID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM mfa_devices WHERE user_id = $1 AND verified = false`, userID)
	return err
}

// DisableMFA removes all MFA devices and backup codes, and sets mfa_enabled=false.
func (db *DB) DisableMFA(ctx context.Context, userID uuid.UUID) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback best-effort on deferred cleanup

	_, _ = tx.Exec(ctx, `DELETE FROM mfa_devices WHERE user_id = $1`, userID)
	_, _ = tx.Exec(ctx, `DELETE FROM mfa_backup_codes WHERE user_id = $1`, userID)
	_, err = tx.Exec(ctx, `UPDATE users SET mfa_enabled = false, updated_at = now() WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("disabling MFA: %w", err)
	}
	return tx.Commit(ctx)
}

// StoreBackupCodes stores hashed backup codes for a user.
func (db *DB) StoreBackupCodes(ctx context.Context, userID uuid.UUID, codeHashes [][]byte) error {
	// Delete existing codes first
	_, _ = db.Pool.Exec(ctx, `DELETE FROM mfa_backup_codes WHERE user_id = $1`, userID)

	for _, hash := range codeHashes {
		_, err := db.Pool.Exec(ctx,
			`INSERT INTO mfa_backup_codes (user_id, code_hash) VALUES ($1, $2)`,
			userID, hash,
		)
		if err != nil {
			return fmt.Errorf("inserting backup code: %w", err)
		}
	}
	return nil
}

// ConsumeBackupCode marks a backup code as used if the hash matches.
// Returns true if a valid code was consumed.
func (db *DB) ConsumeBackupCode(ctx context.Context, userID uuid.UUID, codeHash []byte) (bool, error) {
	tag, err := db.Pool.Exec(ctx,
		`UPDATE mfa_backup_codes SET used = true
		 WHERE user_id = $1 AND code_hash = $2 AND used = false`,
		userID, codeHash,
	)
	if err != nil {
		return false, fmt.Errorf("consuming backup code: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
