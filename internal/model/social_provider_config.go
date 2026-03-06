package model

import (
	"time"

	"github.com/google/uuid"
)

// SocialProviderConfig stores OAuth credentials for a social login provider.
type SocialProviderConfig struct {
	ID           uuid.UUID         `json:"id"`
	OrgID        uuid.UUID         `json:"org_id"`
	Provider     string            `json:"provider"`
	Enabled      bool              `json:"enabled"`
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"-"`
	ExtraConfig  map[string]string `json:"extra_config,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// SocialProviderConfigResponse is the public representation (secret masked).
type SocialProviderConfigResponse struct {
	ID           uuid.UUID         `json:"id"`
	Provider     string            `json:"provider"`
	Enabled      bool              `json:"enabled"`
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"client_secret"`
	ExtraConfig  map[string]string `json:"extra_config,omitempty"`
}

// ToResponse converts to a response with the secret masked.
func (c *SocialProviderConfig) ToResponse() *SocialProviderConfigResponse {
	masked := ""
	if c.ClientSecret != "" {
		masked = "********"
	}
	return &SocialProviderConfigResponse{
		ID:           c.ID,
		Provider:     c.Provider,
		Enabled:      c.Enabled,
		ClientID:     c.ClientID,
		ClientSecret: masked,
		ExtraConfig:  c.ExtraConfig,
	}
}
