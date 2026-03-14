package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/manimovassagh/rampart/internal/model"
)

// GetOAuthClient retrieves an OAuth client by its ID.
func (db *DB) GetOAuthClient(ctx context.Context, clientID string) (*model.OAuthClient, error) {
	ctx, cancel := queryCtx(ctx)
	defer cancel()

	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, client_type, redirect_uris,
		       COALESCE(client_secret_hash, ''::bytea), COALESCE(description, ''),
		       COALESCE(enabled, true), first_party, created_at, updated_at
		FROM oauth_clients
		WHERE id = $1`, clientID)

	var c model.OAuthClient
	err := row.Scan(&c.ID, &c.OrgID, &c.Name, &c.ClientType, &c.RedirectURIs,
		&c.ClientSecretHash, &c.Description, &c.Enabled, &c.FirstParty, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying oauth client %q: %w", clientID, err)
	}
	return &c, nil
}

// ValidateRedirectURI checks if the given URI is in the client's registered redirect URIs.
// Uses exact string match per RFC 6749.
func ValidateRedirectURI(client *model.OAuthClient, uri string) bool {
	for _, registered := range client.RedirectURIs {
		if registered == uri {
			return true
		}
	}
	return false
}

// ListOAuthClients returns a paginated, searchable list of OAuth clients for an org.
func (db *DB) ListOAuthClients(ctx context.Context, orgID uuid.UUID, search string, limit, offset int) ([]*model.OAuthClient, int, error) {
	where := []string{"org_id = $1"}
	args := []any{orgID}
	paramIdx := 2

	if search != "" {
		where = append(where, fmt.Sprintf(
			"(name ILIKE $%d OR id ILIKE $%d OR COALESCE(description, '') ILIKE $%d)",
			paramIdx, paramIdx, paramIdx,
		))
		args = append(args, "%"+search+"%")
		paramIdx++
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM oauth_clients WHERE %s", whereClause)
	if err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting oauth clients: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, org_id, name, client_type, redirect_uris,
		       COALESCE(client_secret_hash, ''::bytea), COALESCE(description, ''),
		       COALESCE(enabled, true), first_party, created_at, updated_at
		FROM oauth_clients
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, paramIdx, paramIdx+1)
	args = append(args, limit, offset)

	rows, err := db.Pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing oauth clients: %w", err)
	}
	defer rows.Close()

	var clients []*model.OAuthClient
	for rows.Next() {
		var c model.OAuthClient
		if err := rows.Scan(&c.ID, &c.OrgID, &c.Name, &c.ClientType, &c.RedirectURIs,
			&c.ClientSecretHash, &c.Description, &c.Enabled, &c.FirstParty, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning oauth client row: %w", err)
		}
		clients = append(clients, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating oauth client rows: %w", err)
	}

	return clients, total, nil
}

// CreateOAuthClient inserts a new OAuth client with a generated client ID.
func (db *DB) CreateOAuthClient(ctx context.Context, client *model.OAuthClient) (*model.OAuthClient, error) {
	if client.ID == "" {
		client.ID = generateClientID()
	}

	query := `
		INSERT INTO oauth_clients (id, org_id, name, client_type, redirect_uris, client_secret_hash, description, enabled, first_party)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, org_id, name, client_type, redirect_uris,
		          COALESCE(client_secret_hash, ''::bytea), COALESCE(description, ''),
		          enabled, first_party, created_at, updated_at`

	var c model.OAuthClient
	err := db.Pool.QueryRow(ctx, query,
		client.ID, client.OrgID, client.Name, client.ClientType,
		client.RedirectURIs, client.ClientSecretHash, client.Description, client.Enabled, client.FirstParty,
	).Scan(&c.ID, &c.OrgID, &c.Name, &c.ClientType, &c.RedirectURIs,
		&c.ClientSecretHash, &c.Description, &c.Enabled, &c.FirstParty, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting oauth client: %w", err)
	}
	return &c, nil
}

// UpdateOAuthClient updates mutable fields on an OAuth client, scoped to the given organization.
func (db *DB) UpdateOAuthClient(ctx context.Context, clientID string, orgID uuid.UUID, req *model.UpdateClientRequest) (*model.OAuthClient, error) {
	uris := parseRedirectURIs(req.RedirectURIs)

	query := `
		UPDATE oauth_clients
		SET name = COALESCE(NULLIF($2, ''), name),
		    description = $3,
		    redirect_uris = $4,
		    enabled = $5,
		    updated_at = now()
		WHERE id = $1 AND org_id = $6
		RETURNING id, org_id, name, client_type, redirect_uris,
		          COALESCE(client_secret_hash, ''::bytea), COALESCE(description, ''),
		          enabled, first_party, created_at, updated_at`

	var c model.OAuthClient
	err := db.Pool.QueryRow(ctx, query, clientID, req.Name, req.Description, uris, req.Enabled, orgID).Scan(
		&c.ID, &c.OrgID, &c.Name, &c.ClientType, &c.RedirectURIs,
		&c.ClientSecretHash, &c.Description, &c.Enabled, &c.FirstParty, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("updating oauth client: %w", err)
	}
	return &c, nil
}

// DeleteOAuthClient removes an OAuth client by ID, scoped to the given organization.
func (db *DB) DeleteOAuthClient(ctx context.Context, clientID string, orgID uuid.UUID) error {
	tag, err := db.Pool.Exec(ctx, "DELETE FROM oauth_clients WHERE id = $1 AND org_id = $2", clientID, orgID)
	if err != nil {
		return fmt.Errorf("deleting oauth client: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("oauth client not found")
	}
	return nil
}

// UpdateClientSecret sets a new secret hash for a confidential client, scoped to the given organization.
func (db *DB) UpdateClientSecret(ctx context.Context, clientID string, orgID uuid.UUID, secretHash []byte) error {
	_, err := db.Pool.Exec(ctx,
		"UPDATE oauth_clients SET client_secret_hash = $2, updated_at = now() WHERE id = $1 AND org_id = $3",
		clientID, secretHash, orgID)
	if err != nil {
		return fmt.Errorf("updating client secret: %w", err)
	}
	return nil
}

// CountOAuthClients returns the total number of OAuth clients in an org.
func (db *DB) CountOAuthClients(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM oauth_clients WHERE org_id = $1", orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting oauth clients: %w", err)
	}
	return count, nil
}

func generateClientID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()
	}
	return hex.EncodeToString(b)
}

func parseRedirectURIs(raw string) []string {
	var uris []string
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			uris = append(uris, trimmed)
		}
	}
	return uris
}
