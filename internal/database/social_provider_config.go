package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
)

// UpsertSocialProviderConfig creates or updates a social provider config for an org.
func (db *DB) UpsertSocialProviderConfig(ctx context.Context, cfg *model.SocialProviderConfig) error {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	extra, err := json.Marshal(cfg.ExtraConfig)
	if err != nil {
		return fmt.Errorf("marshalling extra_config: %w", err)
	}

	query := `
		INSERT INTO social_provider_configs (org_id, provider, enabled, client_id, client_secret, extra_config)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (org_id, provider) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			client_id = EXCLUDED.client_id,
			client_secret = CASE WHEN EXCLUDED.client_secret = '' THEN social_provider_configs.client_secret ELSE EXCLUDED.client_secret END,
			extra_config = EXCLUDED.extra_config,
			updated_at = now()
		RETURNING id, created_at, updated_at`

	encSecret, encErr := db.encryptToken(cfg.ClientSecret)
	if encErr != nil {
		return fmt.Errorf("encrypting client secret: %w", encErr)
	}

	return db.Pool.QueryRow(ctx, query,
		cfg.OrgID, cfg.Provider, cfg.Enabled, cfg.ClientID, encSecret, extra,
	).Scan(&cfg.ID, &cfg.CreatedAt, &cfg.UpdatedAt)
}

// GetSocialProviderConfig returns a single provider config for an org.
func (db *DB) GetSocialProviderConfig(ctx context.Context, orgID uuid.UUID, provider string) (*model.SocialProviderConfig, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	query := `
		SELECT id, org_id, provider, enabled, client_id, client_secret, extra_config, created_at, updated_at
		FROM social_provider_configs
		WHERE org_id = $1 AND provider = $2`

	var cfg model.SocialProviderConfig
	var extraJSON []byte
	err := db.Pool.QueryRow(ctx, query, orgID, provider).Scan(
		&cfg.ID, &cfg.OrgID, &cfg.Provider, &cfg.Enabled,
		&cfg.ClientID, &cfg.ClientSecret, &extraJSON,
		&cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting social provider config: %w", err)
	}

	if err := json.Unmarshal(extraJSON, &cfg.ExtraConfig); err != nil {
		return nil, fmt.Errorf("unmarshalling extra_config: %w", err)
	}
	if cfg.ClientSecret, err = db.decryptToken(cfg.ClientSecret); err != nil {
		return nil, fmt.Errorf("decrypting client secret: %w", err)
	}
	return &cfg, nil
}

// ListSocialProviderConfigs returns all provider configs for an org.
func (db *DB) ListSocialProviderConfigs(ctx context.Context, orgID uuid.UUID) ([]*model.SocialProviderConfig, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	query := `
		SELECT id, org_id, provider, enabled, client_id, client_secret, extra_config, created_at, updated_at
		FROM social_provider_configs
		WHERE org_id = $1
		ORDER BY provider`

	rows, err := db.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("listing social provider configs: %w", err)
	}
	defer rows.Close()

	var configs []*model.SocialProviderConfig
	for rows.Next() {
		var cfg model.SocialProviderConfig
		var extraJSON []byte
		if err := rows.Scan(
			&cfg.ID, &cfg.OrgID, &cfg.Provider, &cfg.Enabled,
			&cfg.ClientID, &cfg.ClientSecret, &extraJSON,
			&cfg.CreatedAt, &cfg.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning social provider config: %w", err)
		}
		if err := json.Unmarshal(extraJSON, &cfg.ExtraConfig); err != nil {
			return nil, fmt.Errorf("unmarshalling extra_config: %w", err)
		}
		if cfg.ClientSecret, err = db.decryptToken(cfg.ClientSecret); err != nil {
			return nil, fmt.Errorf("decrypting client secret: %w", err)
		}
		configs = append(configs, &cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating social provider configs: %w", err)
	}
	return configs, nil
}

// DeleteSocialProviderConfig removes a provider config for an org.
func (db *DB) DeleteSocialProviderConfig(ctx context.Context, orgID uuid.UUID, provider string) error {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `DELETE FROM social_provider_configs WHERE org_id = $1 AND provider = $2`, orgID, provider)
	if err != nil {
		return fmt.Errorf("deleting social provider config: %w", err)
	}
	return nil
}
