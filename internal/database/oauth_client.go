package database

import (
	"context"
	"fmt"

	"github.com/manimovassagh/rampart/internal/model"
)

// GetOAuthClient retrieves an OAuth client by its ID.
func (db *DB) GetOAuthClient(ctx context.Context, clientID string) (*model.OAuthClient, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, client_type, redirect_uris, created_at, updated_at
		FROM oauth_clients
		WHERE id = $1`, clientID)

	var c model.OAuthClient
	err := row.Scan(&c.ID, &c.OrgID, &c.Name, &c.ClientType, &c.RedirectURIs, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
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
