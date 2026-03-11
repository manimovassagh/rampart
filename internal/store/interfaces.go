// Package store defines per-aggregate repository interfaces for the Rampart database layer.
// The concrete implementation lives in internal/database — the *database.DB struct
// implicitly satisfies all interfaces defined here.
//
// Handlers embed these interfaces in their own store types, eliminating
// duplicated method signatures across 15+ handler files.
package store

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// ── User ────────────────────────────────────────────────────────────────

// UserReader provides read-only user lookups.
type UserReader interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string, orgID uuid.UUID) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string, orgID uuid.UUID) (*model.User, error)
	FindUserByEmail(ctx context.Context, email string) (*model.User, error)
}

// UserWriter provides user mutation operations.
type UserWriter interface {
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	UpdateUser(ctx context.Context, id, orgID uuid.UUID, req *model.UpdateUserRequest) (*model.User, error)
	DeleteUser(ctx context.Context, id, orgID uuid.UUID) error
	UpdatePassword(ctx context.Context, id, orgID uuid.UUID, passwordHash []byte) error
	UpdateLastLoginAt(ctx context.Context, userID uuid.UUID) error
	IncrementFailedLogins(ctx context.Context, userID uuid.UUID, maxAttempts int, lockoutDuration time.Duration) error
	ResetFailedLogins(ctx context.Context, userID uuid.UUID) error
}

// UserLister provides paginated user listing with search.
type UserLister interface {
	ListUsers(ctx context.Context, orgID uuid.UUID, search, status string, limit, offset int) ([]*model.User, int, error)
	CountUsers(ctx context.Context, orgID uuid.UUID) (int, error)
	CountRecentUsers(ctx context.Context, orgID uuid.UUID, days int) (int, error)
}

// ── Organization ────────────────────────────────────────────────────────

// OrgReader provides organization lookups.
type OrgReader interface {
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*model.Organization, error)
	GetDefaultOrganizationID(ctx context.Context) (uuid.UUID, error)
	GetOrganizationIDBySlug(ctx context.Context, slug string) (uuid.UUID, error)
}

// OrgWriter provides organization mutations.
type OrgWriter interface {
	CreateOrganization(ctx context.Context, req *model.CreateOrgRequest) (*model.Organization, error)
	UpdateOrganization(ctx context.Context, id uuid.UUID, req *model.UpdateOrgRequest) (*model.Organization, error)
	DeleteOrganization(ctx context.Context, id uuid.UUID) error
}

// OrgLister provides paginated organization listing.
type OrgLister interface {
	ListOrganizations(ctx context.Context, search string, limit, offset int) ([]*model.Organization, int, error)
	CountOrganizations(ctx context.Context) (int, error)
}

// OrgSettingsReadWriter provides organization settings CRUD.
type OrgSettingsReadWriter interface {
	GetOrgSettings(ctx context.Context, orgID uuid.UUID) (*model.OrgSettings, error)
	UpdateOrgSettings(ctx context.Context, orgID uuid.UUID, req *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error)
}

// ── Role ────────────────────────────────────────────────────────────────

// RoleReader provides role lookups and user-role queries.
type RoleReader interface {
	GetRoleByID(ctx context.Context, id uuid.UUID) (*model.Role, error)
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*model.Role, error)
	GetUserRoleNames(ctx context.Context, userID uuid.UUID) ([]string, error)
	GetRoleUsers(ctx context.Context, roleID uuid.UUID) ([]*model.UserRoleAssignment, error)
	CountRoleUsers(ctx context.Context, roleID uuid.UUID) (int, error)
	CountRoles(ctx context.Context, orgID uuid.UUID) (int, error)
	UserCountsByRole(ctx context.Context, orgID uuid.UUID) ([]model.RoleCount, error)
}

// RoleWriter provides role mutation operations.
type RoleWriter interface {
	CreateRole(ctx context.Context, role *model.Role) (*model.Role, error)
	UpdateRole(ctx context.Context, id, orgID uuid.UUID, req *model.UpdateRoleRequest) (*model.Role, error)
	DeleteRole(ctx context.Context, id, orgID uuid.UUID) error
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
	UnassignRole(ctx context.Context, userID, roleID uuid.UUID) error
}

// RoleLister provides paginated role listing.
type RoleLister interface {
	ListRoles(ctx context.Context, orgID uuid.UUID, search string, limit, offset int) ([]*model.Role, int, error)
}

// ── Group ───────────────────────────────────────────────────────────────

// GroupReader provides group lookups and membership queries.
type GroupReader interface {
	GetGroupByID(ctx context.Context, id uuid.UUID) (*model.Group, error)
	GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]*model.GroupMember, error)
	GetGroupRoles(ctx context.Context, groupID uuid.UUID) ([]*model.GroupRoleAssignment, error)
	GetUserGroups(ctx context.Context, userID uuid.UUID) ([]*model.Group, error)
	GetEffectiveUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
	CountGroupMembers(ctx context.Context, groupID uuid.UUID) (int, error)
	CountGroupRoles(ctx context.Context, groupID uuid.UUID) (int, error)
	CountGroups(ctx context.Context, orgID uuid.UUID) (int, error)
}

// GroupWriter provides group mutation operations.
type GroupWriter interface {
	CreateGroup(ctx context.Context, group *model.Group) (*model.Group, error)
	UpdateGroup(ctx context.Context, id uuid.UUID, req *model.UpdateGroupRequest) (*model.Group, error)
	DeleteGroup(ctx context.Context, id uuid.UUID) error
	AddUserToGroup(ctx context.Context, userID, groupID uuid.UUID) error
	RemoveUserFromGroup(ctx context.Context, userID, groupID uuid.UUID) error
	AssignRoleToGroup(ctx context.Context, groupID, roleID uuid.UUID) error
	UnassignRoleFromGroup(ctx context.Context, groupID, roleID uuid.UUID) error
}

// GroupLister provides paginated group listing.
type GroupLister interface {
	ListGroups(ctx context.Context, orgID uuid.UUID, search string, limit, offset int) ([]*model.Group, int, error)
}

// ── OAuth Client ────────────────────────────────────────────────────────

// OAuthClientReader provides OAuth client lookups.
type OAuthClientReader interface {
	GetOAuthClient(ctx context.Context, clientID string) (*model.OAuthClient, error)
}

// OAuthClientWriter provides OAuth client mutations.
type OAuthClientWriter interface {
	CreateOAuthClient(ctx context.Context, client *model.OAuthClient) (*model.OAuthClient, error)
	UpdateOAuthClient(ctx context.Context, clientID string, orgID uuid.UUID, req *model.UpdateClientRequest) (*model.OAuthClient, error)
	DeleteOAuthClient(ctx context.Context, clientID string, orgID uuid.UUID) error
	UpdateClientSecret(ctx context.Context, clientID string, orgID uuid.UUID, secretHash []byte) error
}

// OAuthClientLister provides paginated OAuth client listing.
type OAuthClientLister interface {
	ListOAuthClients(ctx context.Context, orgID uuid.UUID, search string, limit, offset int) ([]*model.OAuthClient, int, error)
	CountOAuthClients(ctx context.Context, orgID uuid.UUID) (int, error)
}

// ── Authorization Code ──────────────────────────────────────────────────

// AuthCodeStore provides OAuth authorization code operations.
type AuthCodeStore interface {
	StoreAuthorizationCode(ctx context.Context, code string, clientID string, userID, orgID uuid.UUID, redirectURI, codeChallenge, scope, nonce string, expiresAt time.Time) error
	ConsumeAuthorizationCode(ctx context.Context, code string) (*model.AuthorizationCode, error)
	DeleteExpiredAuthorizationCodes(ctx context.Context) (int64, error)
}

// ── Consent ─────────────────────────────────────────────────────────────

// ConsentStore provides user OAuth consent operations.
type ConsentStore interface {
	HasConsent(ctx context.Context, userID uuid.UUID, clientID, scopes string) (bool, error)
	GrantConsent(ctx context.Context, userID uuid.UUID, clientID, scopes string) error
}

// ── Password Reset ──────────────────────────────────────────────────────

// PasswordResetTokenStore provides password reset token operations.
type PasswordResetTokenStore interface {
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error
	ConsumePasswordResetToken(ctx context.Context, token string) (uuid.UUID, error)
	DeleteExpiredPasswordResetTokens(ctx context.Context) (int64, error)
}

// ── Email Verification ──────────────────────────────────────────────────

// EmailVerificationTokenStore provides email verification token operations.
type EmailVerificationTokenStore interface {
	CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error
	ConsumeEmailVerificationToken(ctx context.Context, token string) (uuid.UUID, error)
	MarkEmailVerified(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredEmailVerificationTokens(ctx context.Context) (int64, error)
}

// ── MFA ─────────────────────────────────────────────────────────────────

// MFADeviceStore provides MFA device and backup code operations.
type MFADeviceStore interface {
	CreateMFADevice(ctx context.Context, userID uuid.UUID, deviceType, name, secret string) (*model.MFADevice, error)
	VerifyMFADevice(ctx context.Context, deviceID, userID uuid.UUID) error
	GetVerifiedMFADevice(ctx context.Context, userID uuid.UUID) (*model.MFADevice, error)
	GetPendingMFADevice(ctx context.Context, userID uuid.UUID) (*model.MFADevice, error)
	DeleteUnverifiedMFADevices(ctx context.Context, userID uuid.UUID) error
	DisableMFA(ctx context.Context, userID uuid.UUID) error
	StoreBackupCodes(ctx context.Context, userID uuid.UUID, codeHashes [][]byte) error
	ConsumeBackupCode(ctx context.Context, userID uuid.UUID, codeHash []byte) (bool, error)
}

// ── Audit ───────────────────────────────────────────────────────────────

// AuditStore provides audit event operations.
type AuditStore interface {
	CreateAuditEvent(ctx context.Context, event *model.AuditEvent) error
	ListAuditEvents(ctx context.Context, orgID uuid.UUID, eventType, search string, limit, offset int) ([]*model.AuditEvent, int, error)
	CountRecentEvents(ctx context.Context, orgID uuid.UUID, hours int) (int, error)
	LoginCountsByDay(ctx context.Context, orgID uuid.UUID, days int) ([]model.DayCount, error)
}

// ── Social ──────────────────────────────────────────────────────────────

// SocialAccountStore provides social login account operations.
type SocialAccountStore interface {
	CreateSocialAccount(ctx context.Context, account *model.SocialAccount) (*model.SocialAccount, error)
	GetSocialAccount(ctx context.Context, provider, providerUserID string) (*model.SocialAccount, error)
	GetSocialAccountsByUserID(ctx context.Context, userID uuid.UUID) ([]*model.SocialAccount, error)
	UpdateSocialAccountTokens(ctx context.Context, id uuid.UUID, accessToken, refreshToken string, expiresAt *time.Time) error
	DeleteSocialAccount(ctx context.Context, id uuid.UUID) error
}

// SocialProviderConfigStore provides social provider configuration operations.
type SocialProviderConfigStore interface {
	UpsertSocialProviderConfig(ctx context.Context, cfg *model.SocialProviderConfig) error
	ListSocialProviderConfigs(ctx context.Context, orgID uuid.UUID) ([]*model.SocialProviderConfig, error)
	DeleteSocialProviderConfig(ctx context.Context, orgID uuid.UUID, provider string) error
}

// ── WebAuthn ────────────────────────────────────────────────────────────

// WebAuthnCredentialStore provides WebAuthn credential operations.
type WebAuthnCredentialStore interface {
	CreateWebAuthnCredential(ctx context.Context, cred *model.WebAuthnCredential) error
	GetWebAuthnCredentialsByUserID(ctx context.Context, userID uuid.UUID) ([]*model.WebAuthnCredential, error)
	UpdateWebAuthnSignCount(ctx context.Context, credentialID []byte, signCount uint32) error
	DeleteWebAuthnCredential(ctx context.Context, id, userID uuid.UUID) error
	CountWebAuthnCredentials(ctx context.Context, userID uuid.UUID) (int, error)
}

// WebAuthnSessionStore provides WebAuthn ceremony session operations.
type WebAuthnSessionStore interface {
	StoreWebAuthnSessionData(ctx context.Context, userID uuid.UUID, data []byte, ceremony string, expiresAt time.Time) error
	GetWebAuthnSessionData(ctx context.Context, userID uuid.UUID, ceremony string) ([]byte, error)
	DeleteExpiredWebAuthnSessions(ctx context.Context) (int64, error)
}

// ── Webhook ─────────────────────────────────────────────────────────────

// WebhookReader provides webhook lookups.
type WebhookReader interface {
	GetWebhookByID(ctx context.Context, id uuid.UUID) (*model.Webhook, error)
	GetEnabledWebhooksForEvent(ctx context.Context, orgID uuid.UUID, eventType string) ([]*model.Webhook, error)
}

// WebhookWriter provides webhook mutation operations.
type WebhookWriter interface {
	CreateWebhook(ctx context.Context, w *model.Webhook) (*model.Webhook, error)
	UpdateWebhook(ctx context.Context, id uuid.UUID, req *model.UpdateWebhookRequest) (*model.Webhook, error)
	DeleteWebhook(ctx context.Context, id uuid.UUID) error
}

// WebhookLister provides paginated webhook listing.
type WebhookLister interface {
	ListWebhooks(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*model.Webhook, int, error)
}

// WebhookDeliveryStore provides webhook delivery operations.
type WebhookDeliveryStore interface {
	CreateWebhookDelivery(ctx context.Context, d *model.WebhookDelivery) error
	UpdateWebhookDelivery(ctx context.Context, id uuid.UUID, status string, attempts int, responseCode *int, lastError string, nextRetry *time.Time, completedAt *time.Time) error
	GetPendingDeliveries(ctx context.Context, limit int) ([]*model.WebhookDelivery, error)
	ListWebhookDeliveries(ctx context.Context, webhookID uuid.UUID, limit, offset int) ([]*model.WebhookDelivery, int, error)
	DeleteOldDeliveries(ctx context.Context, olderThan time.Duration) (int64, error)
}

// GetAuditEventByIDStore provides audit event lookup by ID (used by webhook dispatcher).
type GetAuditEventByIDStore interface {
	GetAuditEventByID(ctx context.Context, id uuid.UUID) (*model.AuditEvent, error)
}

// ── SAML ────────────────────────────────────────────────────────────────

// SAMLProviderStore provides SAML provider configuration operations.
type SAMLProviderStore interface {
	CreateSAMLProvider(ctx context.Context, p *model.SAMLProvider) (*model.SAMLProvider, error)
	GetSAMLProviderByID(ctx context.Context, id uuid.UUID) (*model.SAMLProvider, error)
	ListSAMLProviders(ctx context.Context, orgID uuid.UUID) ([]*model.SAMLProvider, error)
	GetEnabledSAMLProviders(ctx context.Context, orgID uuid.UUID) ([]*model.SAMLProvider, error)
	UpdateSAMLProvider(ctx context.Context, id uuid.UUID, req *model.UpdateSAMLProviderRequest) (*model.SAMLProvider, error)
	DeleteSAMLProvider(ctx context.Context, id uuid.UUID) error
}

// ── Export / Import ─────────────────────────────────────────────────────

// ExportImportStore provides organization export/import operations.
type ExportImportStore interface {
	ExportOrganization(ctx context.Context, orgID uuid.UUID) (*model.OrgExport, error)
	ImportOrganization(ctx context.Context, export *model.OrgExport) error
}
