package model

import (
	"time"

	"github.com/google/uuid"
)

// Group represents a group row in the database.
type Group struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GroupResponse is the admin console representation of a group.
type GroupResponse struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MemberCount int       `json:"member_count"`
	RoleCount   int       `json:"role_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToGroupResponse converts a Group to a GroupResponse.
func (g *Group) ToGroupResponse(memberCount, roleCount int) *GroupResponse {
	return &GroupResponse{
		ID:          g.ID,
		OrgID:       g.OrgID,
		Name:        g.Name,
		Description: g.Description,
		MemberCount: memberCount,
		RoleCount:   roleCount,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}

// CreateGroupRequest is the expected form data for creating a group.
type CreateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateGroupRequest is the expected form data for updating a group.
type UpdateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GroupMember represents a user in a group.
type GroupMember struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	AddedAt  time.Time `json:"added_at"`
}

// GroupRoleAssignment represents a role assigned to a group.
type GroupRoleAssignment struct {
	RoleID      uuid.UUID `json:"role_id"`
	RoleName    string    `json:"role_name"`
	Description string    `json:"description"`
}
