package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// ListUsers returns a paginated, searchable, filterable list of users.
func (db *DB) ListUsers(ctx context.Context, orgID uuid.UUID, search, status string, limit, offset int) ([]*model.User, int, error) {
	where := []string{"org_id = $1"}
	args := []any{orgID}
	paramIdx := 2

	if search != "" {
		where = append(where, fmt.Sprintf(
			"(username ILIKE $%d OR email ILIKE $%d OR given_name ILIKE $%d OR family_name ILIKE $%d)",
			paramIdx, paramIdx, paramIdx, paramIdx,
		))
		args = append(args, "%"+search+"%")
		paramIdx++
	}

	switch status {
	case "enabled":
		where = append(where, fmt.Sprintf("enabled = $%d", paramIdx))
		args = append(args, true)
		paramIdx++
	case "disabled":
		where = append(where, fmt.Sprintf("enabled = $%d", paramIdx))
		args = append(args, false)
		paramIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Count total matching rows.
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE %s", whereClause)
	var total int
	if err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting users: %w", err)
	}

	// Fetch page.
	dataQuery := fmt.Sprintf(`
		SELECT id, org_id, username, email, email_verified, given_name, family_name,
		       enabled, mfa_enabled, last_login_at, created_at, updated_at
		FROM users
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, paramIdx, paramIdx+1)
	args = append(args, limit, offset)

	rows, err := db.Pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.OrgID, &u.Username, &u.Email, &u.EmailVerified,
			&u.GivenName, &u.FamilyName, &u.Enabled, &u.MFAEnabled,
			&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning user row: %w", err)
		}
		users = append(users, &u)
	}

	return users, total, nil
}

// UpdateUser updates mutable fields on a user.
func (db *DB) UpdateUser(ctx context.Context, id uuid.UUID, req *model.UpdateUserRequest) (*model.User, error) {
	query := `
		UPDATE users
		SET username = COALESCE(NULLIF($2, ''), username),
		    email = COALESCE(NULLIF($3, ''), email),
		    given_name = $4,
		    family_name = $5,
		    enabled = $6,
		    email_verified = $7,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, org_id, username, email, email_verified, given_name, family_name,
		          enabled, mfa_enabled, last_login_at, created_at, updated_at`

	var u model.User
	err := db.Pool.QueryRow(ctx, query,
		id, req.Username, req.Email,
		req.GivenName, req.FamilyName,
		req.Enabled, req.EmailVerified,
	).Scan(
		&u.ID, &u.OrgID, &u.Username, &u.Email, &u.EmailVerified,
		&u.GivenName, &u.FamilyName, &u.Enabled, &u.MFAEnabled,
		&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("updating user: %w", err)
	}
	return &u, nil
}

// DeleteUser removes a user by ID.
func (db *DB) DeleteUser(ctx context.Context, id uuid.UUID) error {
	tag, err := db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdatePassword sets a new password hash for a user.
func (db *DB) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash []byte) error {
	_, err := db.Pool.Exec(ctx, "UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1", id, passwordHash)
	if err != nil {
		return fmt.Errorf("updating password: %w", err)
	}
	return nil
}

// CountUsers returns the total number of users in an organization.
func (db *DB) CountUsers(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE org_id = $1", orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}
	return count, nil
}

// CountRecentUsers returns the number of users created in the last N days.
func (db *DB) CountRecentUsers(ctx context.Context, orgID uuid.UUID, days int) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM users WHERE org_id = $1 AND created_at > now() - make_interval(days => $2)",
		orgID, days,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting recent users: %w", err)
	}
	return count, nil
}
