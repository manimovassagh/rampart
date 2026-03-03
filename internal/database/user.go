package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/manimovassagh/rampart/internal/model"
)

// CreateUser inserts a new user and returns the populated User struct.
func (db *DB) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	query := `
		INSERT INTO users (org_id, username, email, given_name, family_name, password_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, org_id, username, email, email_verified, given_name, family_name,
		          enabled, mfa_enabled, created_at, updated_at`

	row := db.Pool.QueryRow(ctx, query,
		user.OrgID,
		user.Username,
		user.Email,
		user.GivenName,
		user.FamilyName,
		user.PasswordHash,
	)

	var created model.User
	err := row.Scan(
		&created.ID,
		&created.OrgID,
		&created.Username,
		&created.Email,
		&created.EmailVerified,
		&created.GivenName,
		&created.FamilyName,
		&created.Enabled,
		&created.MFAEnabled,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return &created, nil
}

// GetUserByEmail finds a user by email within an organization.
func (db *DB) GetUserByEmail(ctx context.Context, email string, orgID uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, org_id, username, email, email_verified, given_name, family_name,
		       enabled, mfa_enabled, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1 AND org_id = $2`

	var u model.User
	err := db.Pool.QueryRow(ctx, query, email, orgID).Scan(
		&u.ID, &u.OrgID, &u.Username, &u.Email, &u.EmailVerified,
		&u.GivenName, &u.FamilyName, &u.Enabled, &u.MFAEnabled,
		&u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying user by email: %w", err)
	}
	return &u, nil
}

// GetUserByID finds a user by their UUID.
func (db *DB) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, org_id, username, email, email_verified, given_name, family_name,
		       enabled, mfa_enabled, password_hash, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1`

	var u model.User
	err := db.Pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.OrgID, &u.Username, &u.Email, &u.EmailVerified,
		&u.GivenName, &u.FamilyName, &u.Enabled, &u.MFAEnabled,
		&u.PasswordHash, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying user by id: %w", err)
	}
	return &u, nil
}

// UpdateLastLoginAt sets the last_login_at timestamp for a user.
func (db *DB) UpdateLastLoginAt(ctx context.Context, userID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, "UPDATE users SET last_login_at = now() WHERE id = $1", userID)
	if err != nil {
		return fmt.Errorf("updating last_login_at: %w", err)
	}
	return nil
}

// GetUserByUsername finds a user by username within an organization.
func (db *DB) GetUserByUsername(ctx context.Context, username string, orgID uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, org_id, username, email, email_verified, given_name, family_name,
		       enabled, mfa_enabled, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1 AND org_id = $2`

	var u model.User
	err := db.Pool.QueryRow(ctx, query, username, orgID).Scan(
		&u.ID, &u.OrgID, &u.Username, &u.Email, &u.EmailVerified,
		&u.GivenName, &u.FamilyName, &u.Enabled, &u.MFAEnabled,
		&u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying user by username: %w", err)
	}
	return &u, nil
}
