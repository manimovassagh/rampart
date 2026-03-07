package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/manimovassagh/rampart/internal/model"
)

// encryptToken encrypts a token if an Encryptor is configured.
func (db *DB) encryptToken(value string) (string, error) {
	if db.Encryptor == nil || value == "" {
		return value, nil
	}
	return db.Encryptor.Encrypt(value)
}

// decryptToken decrypts a token if an Encryptor is configured.
// Plaintext values (without the "enc:" prefix) pass through unchanged.
func (db *DB) decryptToken(value string) (string, error) {
	if db.Encryptor == nil || value == "" {
		return value, nil
	}
	return db.Encryptor.Decrypt(value)
}

// CreateSocialAccount inserts a new social account link and returns the populated struct.
func (db *DB) CreateSocialAccount(ctx context.Context, account *model.SocialAccount) (*model.SocialAccount, error) {
	encAccess, err := db.encryptToken(account.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("encrypting access token: %w", err)
	}
	encRefresh, err := db.encryptToken(account.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("encrypting refresh token: %w", err)
	}

	query := `
		INSERT INTO social_accounts (user_id, provider, provider_user_id, email, name, avatar_url,
		                             access_token, refresh_token, token_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, provider, provider_user_id, email, name, avatar_url,
		          access_token, refresh_token, token_expires_at, created_at, updated_at`

	row := db.Pool.QueryRow(ctx, query,
		account.UserID,
		account.Provider,
		account.ProviderUserID,
		account.Email,
		account.Name,
		account.AvatarURL,
		encAccess,
		encRefresh,
		account.TokenExpiresAt,
	)

	var created model.SocialAccount
	if err := row.Scan(
		&created.ID,
		&created.UserID,
		&created.Provider,
		&created.ProviderUserID,
		&created.Email,
		&created.Name,
		&created.AvatarURL,
		&created.AccessToken,
		&created.RefreshToken,
		&created.TokenExpiresAt,
		&created.CreatedAt,
		&created.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("inserting social account: %w", err)
	}

	if err := db.decryptSocialAccount(&created); err != nil {
		return nil, err
	}
	return &created, nil
}

// decryptSocialAccount decrypts the tokens on a SocialAccount in place.
func (db *DB) decryptSocialAccount(sa *model.SocialAccount) error {
	var err error
	if sa.AccessToken, err = db.decryptToken(sa.AccessToken); err != nil {
		return fmt.Errorf("decrypting social access token: %w", err)
	}
	if sa.RefreshToken, err = db.decryptToken(sa.RefreshToken); err != nil {
		return fmt.Errorf("decrypting social refresh token: %w", err)
	}
	return nil
}

// GetSocialAccount finds a social account by provider and provider user ID.
func (db *DB) GetSocialAccount(ctx context.Context, provider, providerUserID string) (*model.SocialAccount, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, email, name, avatar_url,
		       access_token, refresh_token, token_expires_at, created_at, updated_at
		FROM social_accounts
		WHERE provider = $1 AND provider_user_id = $2`

	var sa model.SocialAccount
	err := db.Pool.QueryRow(ctx, query, provider, providerUserID).Scan(
		&sa.ID, &sa.UserID, &sa.Provider, &sa.ProviderUserID,
		&sa.Email, &sa.Name, &sa.AvatarURL,
		&sa.AccessToken, &sa.RefreshToken, &sa.TokenExpiresAt,
		&sa.CreatedAt, &sa.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying social account by provider: %w", err)
	}
	if err := db.decryptSocialAccount(&sa); err != nil {
		return nil, err
	}
	return &sa, nil
}

// GetSocialAccountsByUserID returns all linked social accounts for a user.
func (db *DB) GetSocialAccountsByUserID(ctx context.Context, userID uuid.UUID) ([]*model.SocialAccount, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, email, name, avatar_url,
		       access_token, refresh_token, token_expires_at, created_at, updated_at
		FROM social_accounts
		WHERE user_id = $1
		ORDER BY created_at`

	rows, err := db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing social accounts by user: %w", err)
	}
	defer rows.Close()

	var accounts []*model.SocialAccount
	for rows.Next() {
		var sa model.SocialAccount
		if err := rows.Scan(
			&sa.ID, &sa.UserID, &sa.Provider, &sa.ProviderUserID,
			&sa.Email, &sa.Name, &sa.AvatarURL,
			&sa.AccessToken, &sa.RefreshToken, &sa.TokenExpiresAt,
			&sa.CreatedAt, &sa.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning social account row: %w", err)
		}
		if err := db.decryptSocialAccount(&sa); err != nil {
			return nil, err
		}
		accounts = append(accounts, &sa)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating social account rows: %w", err)
	}

	return accounts, nil
}

// UpdateSocialAccountTokens updates the OAuth tokens for a social account.
func (db *DB) UpdateSocialAccountTokens(ctx context.Context, id uuid.UUID, accessToken, refreshToken string, expiresAt *time.Time) error {
	encAccess, err := db.encryptToken(accessToken)
	if err != nil {
		return fmt.Errorf("encrypting access token: %w", err)
	}
	encRefresh, err := db.encryptToken(refreshToken)
	if err != nil {
		return fmt.Errorf("encrypting refresh token: %w", err)
	}

	query := `
		UPDATE social_accounts
		SET access_token = $2, refresh_token = $3, token_expires_at = $4, updated_at = now()
		WHERE id = $1`

	tag, err := db.Pool.Exec(ctx, query, id, encAccess, encRefresh, expiresAt)
	if err != nil {
		return fmt.Errorf("updating social account tokens: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("social account not found")
	}
	return nil
}

// DeleteSocialAccount removes a social account link by ID.
func (db *DB) DeleteSocialAccount(ctx context.Context, id uuid.UUID) error {
	tag, err := db.Pool.Exec(ctx, "DELETE FROM social_accounts WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting social account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("social account not found")
	}
	return nil
}
