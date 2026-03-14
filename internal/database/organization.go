package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/store"
)

const defaultOrgSlug = "default"

// GetDefaultOrganizationID returns the UUID of the "default" organization seeded by migration 000001.
func (db *DB) GetDefaultOrganizationID(ctx context.Context) (uuid.UUID, error) {
	return db.GetOrganizationIDBySlug(ctx, defaultOrgSlug)
}

// GetOrganizationIDBySlug returns the UUID of an organization by its slug.
func (db *DB) GetOrganizationIDBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

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

// GetOrganizationByID returns a full Organization by its UUID.
func (db *DB) GetOrganizationByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	query := `
		SELECT id, name, slug, display_name, enabled, created_at, updated_at
		FROM organizations
		WHERE id = $1`

	var o model.Organization
	err := db.Pool.QueryRow(ctx, query, id).Scan(
		&o.ID, &o.Name, &o.Slug, &o.DisplayName, &o.Enabled, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying organization by id: %w", err)
	}
	return &o, nil
}

// ListOrganizations returns a paginated, searchable list of organizations.
func (db *DB) ListOrganizations(ctx context.Context, search string, limit, offset int) ([]*model.Organization, int, error) {
	where := []string{"1=1"}
	args := []any{}
	paramIdx := 1

	if search != "" {
		where = append(where, fmt.Sprintf(
			"(name ILIKE $%d OR slug ILIKE $%d OR display_name ILIKE $%d)",
			paramIdx, paramIdx, paramIdx,
		))
		args = append(args, "%"+search+"%")
		paramIdx++
	}

	whereClause := strings.Join(where, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM organizations WHERE %s", whereClause)
	var total int
	if err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting organizations: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, name, slug, display_name, enabled, created_at, updated_at
		FROM organizations
		WHERE %s
		ORDER BY created_at ASC
		LIMIT $%d OFFSET $%d`, whereClause, paramIdx, paramIdx+1)
	args = append(args, limit, offset)

	rows, err := db.Pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing organizations: %w", err)
	}
	defer rows.Close()

	var orgs []*model.Organization
	for rows.Next() {
		var o model.Organization
		if err := rows.Scan(
			&o.ID, &o.Name, &o.Slug, &o.DisplayName, &o.Enabled, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning organization row: %w", err)
		}
		orgs = append(orgs, &o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating organization rows: %w", err)
	}

	return orgs, total, nil
}

// CreateOrganization inserts an organization and its default settings atomically.
func (db *DB) CreateOrganization(ctx context.Context, req *model.CreateOrgRequest) (*model.Organization, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	orgQuery := `
		INSERT INTO organizations (name, slug, display_name)
		VALUES ($1, $2, $3)
		RETURNING id, name, slug, display_name, enabled, created_at, updated_at`

	var o model.Organization
	err = tx.QueryRow(ctx, orgQuery, req.Name, req.Slug, req.DisplayName).Scan(
		&o.ID, &o.Name, &o.Slug, &o.DisplayName, &o.Enabled, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil, fmt.Errorf("inserting organization: %w", store.ErrDuplicateKey)
		}
		return nil, fmt.Errorf("inserting organization: %w", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO organization_settings (org_id) VALUES ($1)", o.ID)
	if err != nil {
		return nil, fmt.Errorf("inserting default organization settings: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return &o, nil
}

// UpdateOrganization updates mutable fields on an organization.
func (db *DB) UpdateOrganization(ctx context.Context, id uuid.UUID, req *model.UpdateOrgRequest) (*model.Organization, error) {
	query := `
		UPDATE organizations
		SET name = COALESCE(NULLIF($2, ''), name),
		    display_name = $3,
		    enabled = $4,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, name, slug, display_name, enabled, created_at, updated_at`

	var o model.Organization
	err := db.Pool.QueryRow(ctx, query, id, req.Name, req.DisplayName, req.Enabled).Scan(
		&o.ID, &o.Name, &o.Slug, &o.DisplayName, &o.Enabled, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("updating organization: %w", err)
	}
	return &o, nil
}

// DeleteOrganization removes an organization by ID. The default org cannot be deleted.
func (db *DB) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	// Check that it's not the default org.
	var slug string
	err := db.Pool.QueryRow(ctx, "SELECT slug FROM organizations WHERE id = $1", id).Scan(&slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.ErrNotFound
		}
		return fmt.Errorf("querying organization slug: %w", err)
	}
	if slug == defaultOrgSlug {
		return store.ErrDefaultOrg
	}

	tag, err := db.Pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting organization: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

// CountOrganizations returns the total number of organizations.
func (db *DB) CountOrganizations(ctx context.Context) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM organizations").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting organizations: %w", err)
	}
	return count, nil
}
