package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const defaultOrgSlug = "default"

// GetDefaultOrganizationID returns the UUID of the "default" organization seeded by migration 000001.
func (db *DB) GetDefaultOrganizationID(ctx context.Context) (uuid.UUID, error) {
	return db.GetOrganizationIDBySlug(ctx, defaultOrgSlug)
}

// GetOrganizationIDBySlug returns the UUID of an organization by its slug.
func (db *DB) GetOrganizationIDBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.Pool.QueryRow(ctx,
		"SELECT id FROM organizations WHERE slug = $1", slug,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("organization %q not found", slug)
		}
		return uuid.Nil, fmt.Errorf("querying organization by slug: %w", err)
	}
	return id, nil
}
