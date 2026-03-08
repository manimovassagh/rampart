package model

import (
	"time"

	"github.com/google/uuid"
)

// Audit event type constants.
const (
	EventUserLogin               = "user.login"
	EventUserLoginFailed         = "user.login_failed"
	EventUserCreated             = "user.created"
	EventUserUpdated             = "user.updated"
	EventUserDeleted             = "user.deleted"
	EventUserPasswordReset       = "user.password_reset"
	EventClientCreated      = "client.created"
	EventOrgCreated         = "org.created"
	EventSessionRevoked          = "session.revoked"
	EventSessionsRevokedAll      = "session.revoked_all"
	EventSocialLogin             = "social_login"
	EventSocialLoginFailed       = "social_login_failed"
	EventSocialAccountLinked     = "social_account_linked"
)

// AuditEvent represents a row in the audit_events table.
type AuditEvent struct {
	ID         uuid.UUID      `json:"id"`
	OrgID      uuid.UUID      `json:"org_id"`
	EventType  string         `json:"event_type"`
	ActorID    *uuid.UUID     `json:"actor_id,omitempty"`
	ActorName  string         `json:"actor_name"`
	TargetType string         `json:"target_type"`
	TargetID   string         `json:"target_id"`
	TargetName string         `json:"target_name"`
	IPAddress  string         `json:"ip_address"`
	UserAgent  string         `json:"user_agent"`
	Details    map[string]any `json:"details"`
	CreatedAt  time.Time      `json:"created_at"`
}
