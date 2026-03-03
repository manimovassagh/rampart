package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user row in the database.
type User struct {
	ID                  uuid.UUID  `json:"id"`
	OrgID               uuid.UUID  `json:"org_id"`
	Username            string     `json:"username"`
	Email               string     `json:"email"`
	EmailVerified       bool       `json:"email_verified"`
	GivenName           string     `json:"given_name,omitempty"`
	FamilyName          string     `json:"family_name,omitempty"`
	Picture             string     `json:"picture,omitempty"`
	PhoneNumber         string     `json:"phone_number,omitempty"`
	PhoneNumberVerified bool       `json:"phone_number_verified"`
	PasswordHash        []byte     `json:"-"`
	Enabled             bool       `json:"enabled"`
	MFAEnabled          bool       `json:"mfa_enabled"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// RegistrationRequest is the expected JSON body for POST /register.
type RegistrationRequest struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

// UserResponse is the public representation of a user — never contains password_hash.
type UserResponse struct {
	ID            uuid.UUID `json:"id"`
	OrgID         uuid.UUID `json:"org_id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	GivenName     string    `json:"given_name,omitempty"`
	FamilyName    string    `json:"family_name,omitempty"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ToResponse converts a User to a UserResponse, stripping sensitive fields.
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:            u.ID,
		OrgID:         u.OrgID,
		Username:      u.Username,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		GivenName:     u.GivenName,
		FamilyName:    u.FamilyName,
		Enabled:       u.Enabled,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}
