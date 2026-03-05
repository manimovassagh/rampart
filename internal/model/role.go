package model

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a role row in the database.
type Role struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Builtin     bool      `json:"builtin"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleResponse is the admin console representation of a role.
type RoleResponse struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Builtin     bool      `json:"builtin"`
	UserCount   int       `json:"user_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToRoleResponse converts a Role to a RoleResponse.
func (r *Role) ToRoleResponse(userCount int) *RoleResponse {
	return &RoleResponse{
		ID:          r.ID,
		OrgID:       r.OrgID,
		Name:        r.Name,
		Description: r.Description,
		Builtin:     r.Builtin,
		UserCount:   userCount,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// CreateRoleRequest is the expected form data for creating a role.
type CreateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateRoleRequest is the expected form data for updating a role.
type UpdateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UserRoleAssignment represents a user with an assigned role.
type UserRoleAssignment struct {
	UserID     uuid.UUID `json:"user_id"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	AssignedAt time.Time `json:"assigned_at"`
}
