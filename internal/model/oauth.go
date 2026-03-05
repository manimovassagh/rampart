package model

import (
	"time"

	"github.com/google/uuid"
)

// OAuthClient represents a registered OAuth 2.0 client.
type OAuthClient struct {
	ID               string    `json:"id"`
	OrgID            uuid.UUID `json:"org_id"`
	Name             string    `json:"name"`
	ClientType       string    `json:"client_type"`
	RedirectURIs     []string  `json:"redirect_uris"`
	ClientSecretHash []byte    `json:"-"`
	Description      string    `json:"description"`
	Enabled          bool      `json:"enabled"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// AdminClientResponse is the admin console representation of an OAuth client.
type AdminClientResponse struct {
	ID           string    `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	Name         string    `json:"name"`
	ClientType   string    `json:"client_type"`
	RedirectURIs []string  `json:"redirect_uris"`
	Description  string    `json:"description"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToAdminResponse converts an OAuthClient to an AdminClientResponse.
func (c *OAuthClient) ToAdminResponse() *AdminClientResponse {
	return &AdminClientResponse{
		ID:           c.ID,
		OrgID:        c.OrgID,
		Name:         c.Name,
		ClientType:   c.ClientType,
		RedirectURIs: c.RedirectURIs,
		Description:  c.Description,
		Enabled:      c.Enabled,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

// CreateClientRequest is the expected form data for creating an OAuth client.
type CreateClientRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ClientType   string `json:"client_type"`
	RedirectURIs string `json:"redirect_uris"`
}

// UpdateClientRequest is the expected form data for updating an OAuth client.
type UpdateClientRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	RedirectURIs string `json:"redirect_uris"`
	Enabled      bool   `json:"enabled"`
}

// AuthorizationCode represents a stored authorization code.
type AuthorizationCode struct {
	ID            uuid.UUID `json:"id"`
	CodeHash      []byte    `json:"-"`
	ClientID      string    `json:"client_id"`
	UserID        uuid.UUID `json:"user_id"`
	OrgID         uuid.UUID `json:"org_id"`
	RedirectURI   string    `json:"redirect_uri"`
	CodeChallenge string    `json:"code_challenge"`
	Scope         string    `json:"scope"`
	Used          bool      `json:"used"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
}
