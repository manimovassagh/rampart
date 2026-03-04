package model

import (
	"time"

	"github.com/google/uuid"
)

// OAuthClient represents a registered OAuth 2.0 client.
type OAuthClient struct {
	ID           string    `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	Name         string    `json:"name"`
	ClientType   string    `json:"client_type"`
	RedirectURIs []string  `json:"redirect_uris"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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
