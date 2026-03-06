package model

import (
	"time"

	"github.com/google/uuid"
)

// SocialAccountResponse is the public representation of a linked social account.
type SocialAccountResponse struct {
	ID       uuid.UUID `json:"id"`
	Provider string    `json:"provider"`
	Email    string    `json:"email"`
	Name     string    `json:"name,omitempty"`
}

// SocialAccount represents a linked social/federated identity.
type SocialAccount struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	Provider       string     `json:"provider"`
	ProviderUserID string     `json:"provider_user_id"`
	Email          string     `json:"email"`
	Name           string     `json:"name,omitempty"`
	AvatarURL      string     `json:"avatar_url,omitempty"`
	AccessToken    string     `json:"-"`
	RefreshToken   string     `json:"-"`
	TokenExpiresAt *time.Time `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ToResponse converts a SocialAccount to a SocialAccountResponse, stripping sensitive fields.
func (s *SocialAccount) ToResponse() *SocialAccountResponse {
	return &SocialAccountResponse{
		ID:       s.ID,
		Provider: s.Provider,
		Email:    s.Email,
		Name:     s.Name,
	}
}
