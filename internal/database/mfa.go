package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// CreateTOTPDevice inserts a new TOTP device and returns the populated struct.
func (db *DB) CreateTOTPDevice(ctx context.Context, device *model.TOTPDevice) (*model.TOTPDevice, error) {
	query := `
		INSERT INTO mfa_totp_devices (user_id, secret, name)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, secret, name, verified, created_at, last_used_at`

	var d model.TOTPDevice
	err := db.Pool.QueryRow(ctx, query, device.UserID, device.Secret, device.Name).Scan(
		&d.ID, &d.UserID, &d.Secret, &d.Name, &d.Verified, &d.CreatedAt, &d.LastUsedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting TOTP device: %w", err)
	}
	return &d, nil
}

// GetTOTPDevicesByUserID returns all TOTP devices for a given user.
func (db *DB) GetTOTPDevicesByUserID(ctx context.Context, userID uuid.UUID) ([]*model.TOTPDevice, error) {
	query := `
		SELECT id, user_id, secret, name, verified, created_at, last_used_at
		FROM mfa_totp_devices
		WHERE user_id = $1
		ORDER BY created_at`

	rows, err := db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("querying TOTP devices: %w", err)
	}
	defer rows.Close()

	var devices []*model.TOTPDevice
	for rows.Next() {
		var d model.TOTPDevice
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.Secret, &d.Name, &d.Verified, &d.CreatedAt, &d.LastUsedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning TOTP device row: %w", err)
		}
		devices = append(devices, &d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating TOTP device rows: %w", err)
	}
	return devices, nil
}

// VerifyTOTPDevice marks a TOTP device as verified.
func (db *DB) VerifyTOTPDevice(ctx context.Context, deviceID uuid.UUID) error {
	tag, err := db.Pool.Exec(ctx,
		"UPDATE mfa_totp_devices SET verified = TRUE WHERE id = $1", deviceID)
	if err != nil {
		return fmt.Errorf("verifying TOTP device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("TOTP device not found")
	}
	return nil
}

// DeleteTOTPDevice removes a TOTP device by ID.
func (db *DB) DeleteTOTPDevice(ctx context.Context, deviceID uuid.UUID) error {
	tag, err := db.Pool.Exec(ctx,
		"DELETE FROM mfa_totp_devices WHERE id = $1", deviceID)
	if err != nil {
		return fmt.Errorf("deleting TOTP device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("TOTP device not found")
	}
	return nil
}

// UpdateTOTPDeviceLastUsed sets the last_used_at timestamp to now.
func (db *DB) UpdateTOTPDeviceLastUsed(ctx context.Context, deviceID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		"UPDATE mfa_totp_devices SET last_used_at = now() WHERE id = $1", deviceID)
	if err != nil {
		return fmt.Errorf("updating TOTP device last_used_at: %w", err)
	}
	return nil
}

// CreateRecoveryCodes inserts a batch of recovery codes for a user.
func (db *DB) CreateRecoveryCodes(ctx context.Context, userID uuid.UUID, codes []*model.RecoveryCode) error {
	for _, code := range codes {
		_, err := db.Pool.Exec(ctx,
			"INSERT INTO mfa_recovery_codes (user_id, code_hash) VALUES ($1, $2)",
			userID, code.CodeHash,
		)
		if err != nil {
			return fmt.Errorf("inserting recovery code: %w", err)
		}
	}
	return nil
}

// UseRecoveryCode marks a recovery code as used.
func (db *DB) UseRecoveryCode(ctx context.Context, codeID uuid.UUID) error {
	tag, err := db.Pool.Exec(ctx,
		"UPDATE mfa_recovery_codes SET used = TRUE, used_at = now() WHERE id = $1 AND used = FALSE",
		codeID)
	if err != nil {
		return fmt.Errorf("using recovery code: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("recovery code not found or already used")
	}
	return nil
}

// GetUnusedRecoveryCodes returns all unused recovery codes for a user.
func (db *DB) GetUnusedRecoveryCodes(ctx context.Context, userID uuid.UUID) ([]*model.RecoveryCode, error) {
	query := `
		SELECT id, user_id, code_hash, used, created_at, used_at
		FROM mfa_recovery_codes
		WHERE user_id = $1 AND used = FALSE
		ORDER BY created_at`

	rows, err := db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("querying unused recovery codes: %w", err)
	}
	defer rows.Close()

	var codes []*model.RecoveryCode
	for rows.Next() {
		var c model.RecoveryCode
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.CodeHash, &c.Used, &c.CreatedAt, &c.UsedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning recovery code row: %w", err)
		}
		codes = append(codes, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating recovery code rows: %w", err)
	}
	return codes, nil
}

// SetUserMFAEnabled updates the mfa_enabled flag on a user.
func (db *DB) SetUserMFAEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	_, err := db.Pool.Exec(ctx,
		"UPDATE users SET mfa_enabled = $2, updated_at = now() WHERE id = $1",
		userID, enabled)
	if err != nil {
		return fmt.Errorf("setting user MFA enabled: %w", err)
	}
	return nil
}
