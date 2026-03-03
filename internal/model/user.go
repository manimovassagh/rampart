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
	OrgSlug    string `json:"org_slug,omitempty"`
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

// AdminUserResponse is an enriched user representation for the admin console.
type AdminUserResponse struct {
	ID            uuid.UUID  `json:"id"`
	OrgID         uuid.UUID  `json:"org_id"`
	Username      string     `json:"username"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	GivenName     string     `json:"given_name,omitempty"`
	FamilyName    string     `json:"family_name,omitempty"`
	Enabled       bool       `json:"enabled"`
	MFAEnabled    bool       `json:"mfa_enabled"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	SessionCount  int        `json:"session_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ToAdminResponse converts a User to an AdminUserResponse.
func (u *User) ToAdminResponse(sessionCount int) *AdminUserResponse {
	return &AdminUserResponse{
		ID:            u.ID,
		OrgID:         u.OrgID,
		Username:      u.Username,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		GivenName:     u.GivenName,
		FamilyName:    u.FamilyName,
		Enabled:       u.Enabled,
		MFAEnabled:    u.MFAEnabled,
		LastLoginAt:   u.LastLoginAt,
		SessionCount:  sessionCount,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

// ListUsersResponse is a paginated list of users for the admin API.
type ListUsersResponse struct {
	Users []*AdminUserResponse `json:"users"`
	Total int                  `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}

// CreateUserRequest is the expected JSON body for admin user creation.
type CreateUserRequest struct {
	Username      string `json:"username"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Enabled       bool   `json:"enabled"`
	EmailVerified bool   `json:"email_verified"`
}

// UpdateUserRequest is the expected JSON body for admin user updates.
type UpdateUserRequest struct {
	Username      string `json:"username"`
	Email         string `json:"email"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Enabled       bool   `json:"enabled"`
	EmailVerified bool   `json:"email_verified"`
}

// ResetPasswordRequest is the expected JSON body for admin password resets.
type ResetPasswordRequest struct {
	Password string `json:"password"`
}

// DashboardStats contains summary statistics for the admin dashboard.
type DashboardStats struct {
	TotalUsers         int `json:"total_users"`
	ActiveSessions     int `json:"active_sessions"`
	RecentUsers        int `json:"recent_users"`
	TotalOrganizations int `json:"total_organizations"`
}

// SessionResponse is a session representation for the admin API.
type SessionResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
