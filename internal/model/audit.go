package model

import (
	"time"

	"github.com/google/uuid"
)

// Audit event type constants.
const (
	EventUserLogin            = "user.login"
	EventUserLoginFailed      = "user.login_failed"
	EventUserCreated          = "user.created"
	EventUserUpdated          = "user.updated"
	EventUserDeleted          = "user.deleted"
	EventUserPasswordReset    = "user.password_reset"
	EventRoleAssigned         = "role.assigned"
	EventRoleUnassigned       = "role.unassigned"
	EventRoleCreated          = "role.created"
	EventRoleUpdated          = "role.updated"
	EventRoleDeleted          = "role.deleted"
	EventClientCreated        = "client.created"
	EventClientUpdated        = "client.updated"
	EventClientDeleted        = "client.deleted"
	EventClientSecretRegenerated = "client.secret_regenerated"
	EventOrgCreated           = "org.created"
	EventOrgUpdated           = "org.updated"
	EventOrgDeleted           = "org.deleted"
	EventOrgSettingsUpdated   = "org.settings_updated"
	EventSessionRevoked       = "session.revoked"
	EventSessionsRevokedAll   = "session.revoked_all"
)

// AuditEvent represents a row in the audit_events table.
type AuditEvent struct {
	ID         uuid.UUID              `json:"id"`
	OrgID      uuid.UUID              `json:"org_id"`
	EventType  string                 `json:"event_type"`
	ActorID    *uuid.UUID             `json:"actor_id,omitempty"`
	ActorName  string                 `json:"actor_name"`
	TargetType string                 `json:"target_type"`
	TargetID   string                 `json:"target_id"`
	TargetName string                 `json:"target_name"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	Details    map[string]interface{} `json:"details"`
	CreatedAt  time.Time              `json:"created_at"`
}
