package server

import (
	"crypto/rsa"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/metrics"
	"github.com/manimovassagh/rampart/internal/middleware"
)

// NewRouter creates and configures the chi router with middleware chain.
// Middleware order: RequestID → RealIP → Recovery → SecurityHeaders → CORS → Logging
func NewRouter(logger *slog.Logger, allowedOrigins []string, hstsEnabled bool) *chi.Mux {
	r := chi.NewRouter()

	// Middleware chain — order matters
	r.Use(middleware.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.SecurityHeaders(middleware.SecurityHeadersConfig{
		HSTSEnabled: hstsEnabled,
	}))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", middleware.HeaderRequestID, "X-Org-Context"},
		ExposedHeaders:   []string{middleware.HeaderRequestID},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.Logging(logger))
	r.Use(metrics.Middleware)

	// Custom 404 handler — return JSON instead of plain text
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		apierror.NotFound(w)
	})

	return r
}

// RegisterHealthRoutes mounts the health check endpoints.
func RegisterHealthRoutes(r *chi.Mux, healthHandler, readyHandler http.HandlerFunc) {
	r.Get("/healthz", healthHandler)
	r.Get("/readyz", readyHandler)
}

// RegisterMetricsRoutes mounts the Prometheus metrics endpoint.
func RegisterMetricsRoutes(r *chi.Mux) {
	r.Handle("/metrics", metrics.Handler())
}

// RegisterAuthRoutes mounts authentication-related endpoints.
// If rl is non-nil, it is applied as rate limiting middleware.
func RegisterAuthRoutes(r *chi.Mux, registerHandler http.HandlerFunc, rl *middleware.RateLimiter) {
	if rl != nil {
		r.With(middleware.RequireJSON, rl.Middleware()).Post("/register", registerHandler)
	} else {
		r.With(middleware.RequireJSON).Post("/register", registerHandler)
	}
}

// RegisterLoginRoutes mounts login, token refresh, and logout endpoints.
// If rl is non-nil, it is applied as rate limiting middleware to /login.
func RegisterLoginRoutes(r *chi.Mux, loginHandler, refreshHandler, logoutHandler http.HandlerFunc, rl *middleware.RateLimiter) {
	jsonMW := middleware.RequireJSON
	if rl != nil {
		r.With(jsonMW, rl.Middleware()).Post("/login", loginHandler)
	} else {
		r.With(jsonMW).Post("/login", loginHandler)
	}
	r.With(jsonMW).Post("/token/refresh", refreshHandler)
	r.With(jsonMW).Post("/logout", logoutHandler)
}

// RegisterEmailVerificationRoutes mounts email verification endpoints.
// If rl is non-nil, it is applied as rate limiting middleware to the send endpoint.
func RegisterEmailVerificationRoutes(r *chi.Mux, sendHandler, verifyHandler http.HandlerFunc, rl *middleware.RateLimiter) {
	jsonMW := middleware.RequireJSON
	if rl != nil {
		r.With(jsonMW, rl.Middleware()).Post("/verify-email/send", sendHandler)
	} else {
		r.With(jsonMW).Post("/verify-email/send", sendHandler)
	}
	r.Get("/verify-email", verifyHandler)
}

// RegisterPasswordResetRoutes mounts forgot-password and reset-password endpoints.
// If rl is non-nil, it is applied as rate limiting middleware.
func RegisterPasswordResetRoutes(r *chi.Mux, forgotHandler, resetHandler http.HandlerFunc, rl *middleware.RateLimiter) {
	jsonMW := middleware.RequireJSON
	if rl != nil {
		r.With(jsonMW, rl.Middleware()).Post("/forgot-password", forgotHandler)
		r.With(jsonMW, rl.Middleware()).Post("/reset-password", resetHandler)
	} else {
		r.With(jsonMW).Post("/forgot-password", forgotHandler)
		r.With(jsonMW).Post("/reset-password", resetHandler)
	}
}

// MFAEndpoints groups the handler methods needed by RegisterMFARoutes.
type MFAEndpoints interface {
	EnrollTOTP(w http.ResponseWriter, r *http.Request)
	VerifyTOTPSetup(w http.ResponseWriter, r *http.Request)
	DisableTOTP(w http.ResponseWriter, r *http.Request)
}

// WebAuthnEndpoints groups the handler methods for WebAuthn/Passkey support.
type WebAuthnEndpoints interface {
	BeginRegistration(w http.ResponseWriter, r *http.Request)
	FinishRegistration(w http.ResponseWriter, r *http.Request)
	BeginLogin(w http.ResponseWriter, r *http.Request)
	FinishLogin(w http.ResponseWriter, r *http.Request)
	ListCredentials(w http.ResponseWriter, r *http.Request)
	DeleteCredential(w http.ResponseWriter, r *http.Request)
}

// RegisterMFARoutes mounts MFA enrollment/management (authenticated) and MFA verify (unauthenticated) endpoints.
func RegisterMFARoutes(r *chi.Mux, pubKey *rsa.PublicKey, mfaEnroll MFAEndpoints, mfaVerify http.HandlerFunc, webauthn WebAuthnEndpoints, rl *middleware.RateLimiter) {
	jsonMW := middleware.RequireJSON

	// MFA verify during login — unauthenticated (uses MFA token)
	if rl != nil {
		r.With(jsonMW, rl.Middleware()).Post("/mfa/totp/verify", mfaVerify)
	} else {
		r.With(jsonMW).Post("/mfa/totp/verify", mfaVerify)
	}

	// WebAuthn login — unauthenticated (uses MFA token)
	if rl != nil {
		r.With(jsonMW, rl.Middleware()).Post("/mfa/webauthn/login/begin", webauthn.BeginLogin)
		r.With(rl.Middleware()).Post("/mfa/webauthn/login/complete", webauthn.FinishLogin)
	} else {
		r.With(jsonMW).Post("/mfa/webauthn/login/begin", webauthn.BeginLogin)
		r.Post("/mfa/webauthn/login/complete", webauthn.FinishLogin)
	}

	// MFA enrollment/management — requires authentication
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(pubKey))
		r.With(jsonMW).Post("/mfa/totp/enroll", mfaEnroll.EnrollTOTP)
		r.With(jsonMW).Post("/mfa/totp/verify-setup", mfaEnroll.VerifyTOTPSetup)
		r.With(jsonMW).Post("/mfa/totp/disable", mfaEnroll.DisableTOTP)

		// WebAuthn credential management — requires authentication
		r.Post("/mfa/webauthn/register/begin", webauthn.BeginRegistration)
		r.Post("/mfa/webauthn/register/complete", webauthn.FinishRegistration)
		r.Get("/mfa/webauthn/credentials", webauthn.ListCredentials)
		r.Delete("/mfa/webauthn/credentials/{id}", webauthn.DeleteCredential)
	})
}

// RegisterProtectedRoutes mounts endpoints that require authentication.
func RegisterProtectedRoutes(r *chi.Mux, pubKey *rsa.PublicKey, meHandler http.HandlerFunc) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(pubKey))
		r.Get("/me", meHandler)
	})
}

// AdminEndpoints groups the handler methods needed by RegisterAdminRoutes.
type AdminEndpoints interface {
	Stats(w http.ResponseWriter, r *http.Request)
	ListUsers(w http.ResponseWriter, r *http.Request)
	CreateUser(w http.ResponseWriter, r *http.Request)
	GetUser(w http.ResponseWriter, r *http.Request)
	UpdateUser(w http.ResponseWriter, r *http.Request)
	DeleteUser(w http.ResponseWriter, r *http.Request)
	ResetPassword(w http.ResponseWriter, r *http.Request)
	ListSessions(w http.ResponseWriter, r *http.Request)
	RevokeSessions(w http.ResponseWriter, r *http.Request)
}

// RegisterAdminRoutes mounts admin console endpoints under /api/v1/admin.
func RegisterAdminRoutes(r *chi.Mux, pubKey *rsa.PublicKey, admin AdminEndpoints) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(pubKey))
		r.Use(middleware.RequireRole("admin"))
		r.Use(middleware.RequireJSON)

		r.Get("/api/v1/admin/stats", admin.Stats)
		r.Get("/api/v1/admin/users", admin.ListUsers)
		r.Post("/api/v1/admin/users", admin.CreateUser)
		r.Get("/api/v1/admin/users/{id}", admin.GetUser)
		r.Put("/api/v1/admin/users/{id}", admin.UpdateUser)
		r.Delete("/api/v1/admin/users/{id}", admin.DeleteUser)
		r.Post("/api/v1/admin/users/{id}/reset-password", admin.ResetPassword)
		r.Get("/api/v1/admin/users/{id}/sessions", admin.ListSessions)
		r.Delete("/api/v1/admin/users/{id}/sessions", admin.RevokeSessions)
	})
}

// OrgEndpoints groups the handler methods needed by RegisterOrgRoutes.
type OrgEndpoints interface {
	ListOrgs(w http.ResponseWriter, r *http.Request)
	CreateOrg(w http.ResponseWriter, r *http.Request)
	GetOrg(w http.ResponseWriter, r *http.Request)
	UpdateOrg(w http.ResponseWriter, r *http.Request)
	DeleteOrg(w http.ResponseWriter, r *http.Request)
	GetOrgSettings(w http.ResponseWriter, r *http.Request)
	UpdateOrgSettings(w http.ResponseWriter, r *http.Request)
}

// RegisterOrgRoutes mounts organization management endpoints under /api/v1/admin/organizations.
func RegisterOrgRoutes(r *chi.Mux, pubKey *rsa.PublicKey, org OrgEndpoints) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(pubKey))
		r.Use(middleware.RequireRole("admin"))
		r.Use(middleware.RequireJSON)

		r.Get("/api/v1/admin/organizations", org.ListOrgs)
		r.Post("/api/v1/admin/organizations", org.CreateOrg)
		r.Get("/api/v1/admin/organizations/{id}", org.GetOrg)
		r.Put("/api/v1/admin/organizations/{id}", org.UpdateOrg)
		r.Delete("/api/v1/admin/organizations/{id}", org.DeleteOrg)
		r.Get("/api/v1/admin/organizations/{id}/settings", org.GetOrgSettings)
		r.Put("/api/v1/admin/organizations/{id}/settings", org.UpdateOrgSettings)
	})
}

// RegisterExportImportRoutes mounts organization config export/import endpoints.
func RegisterExportImportRoutes(r *chi.Mux, pubKey *rsa.PublicKey, exportHandler, importHandler http.HandlerFunc) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(pubKey))
		r.Use(middleware.RequireRole("admin"))
		r.Use(middleware.RequireJSON)

		r.Get("/api/v1/admin/organizations/{id}/export", exportHandler)
		r.Post("/api/v1/admin/organizations/{id}/import", importHandler)
	})
}

// RegisterOAuthRoutes mounts the OAuth 2.0 authorization and token endpoints.
// If rl is non-nil, it is applied as rate limiting middleware to /oauth/token.
func RegisterOAuthRoutes(r *chi.Mux, authorize, consent, token, revoke http.HandlerFunc, rl *middleware.RateLimiter) {
	r.Get("/oauth/authorize", authorize)
	r.Post("/oauth/authorize", authorize)
	r.Post("/oauth/consent", consent)
	if rl != nil {
		r.With(rl.Middleware()).Post("/oauth/token", token)
	} else {
		r.Post("/oauth/token", token)
	}
	r.Post("/oauth/revoke", revoke)
}

// RegisterSocialRoutes mounts social login initiation and callback endpoints.
func RegisterSocialRoutes(r *chi.Mux, initiate, callback http.HandlerFunc) {
	r.Get("/oauth/social/{provider}", initiate)
	r.Get("/oauth/social/{provider}/callback", callback)
}

// SAMLEndpoints groups the handler methods for SAML SP endpoints.
type SAMLEndpoints interface {
	Metadata(w http.ResponseWriter, r *http.Request)
	InitiateLogin(w http.ResponseWriter, r *http.Request)
	ACS(w http.ResponseWriter, r *http.Request)
	ListProviders(w http.ResponseWriter, r *http.Request)
}

// RegisterSAMLRoutes mounts SAML SP endpoints (public, no auth required).
func RegisterSAMLRoutes(r *chi.Mux, samlHandler SAMLEndpoints) {
	r.Get("/saml/providers", samlHandler.ListProviders)
	r.Get("/saml/{providerID}/metadata", samlHandler.Metadata)
	r.Get("/saml/{providerID}/login", samlHandler.InitiateLogin)
	r.Post("/saml/{providerID}/acs", samlHandler.ACS)
}

// SCIMEndpoints groups the handler methods for SCIM 2.0 provisioning.
type SCIMEndpoints interface {
	ServiceProviderConfig(w http.ResponseWriter, r *http.Request)
	ResourceTypes(w http.ResponseWriter, r *http.Request)
	Schemas(w http.ResponseWriter, r *http.Request)
	ListUsers(w http.ResponseWriter, r *http.Request)
	GetUser(w http.ResponseWriter, r *http.Request)
	CreateUser(w http.ResponseWriter, r *http.Request)
	UpdateUser(w http.ResponseWriter, r *http.Request)
	PatchUser(w http.ResponseWriter, r *http.Request)
	DeleteUser(w http.ResponseWriter, r *http.Request)
	ListGroups(w http.ResponseWriter, r *http.Request)
	GetGroup(w http.ResponseWriter, r *http.Request)
	CreateGroup(w http.ResponseWriter, r *http.Request)
	UpdateGroup(w http.ResponseWriter, r *http.Request)
	PatchGroup(w http.ResponseWriter, r *http.Request)
	DeleteGroup(w http.ResponseWriter, r *http.Request)
}

// RegisterSCIMRoutes mounts SCIM 2.0 provisioning endpoints under /scim/v2/.
// These endpoints require Bearer token authentication.
func RegisterSCIMRoutes(r *chi.Mux, pubKey *rsa.PublicKey, scim SCIMEndpoints) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(pubKey))
		r.Use(middleware.RequireRole("admin"))

		// Discovery
		r.Get("/scim/v2/ServiceProviderConfig", scim.ServiceProviderConfig)
		r.Get("/scim/v2/ResourceTypes", scim.ResourceTypes)
		r.Get("/scim/v2/Schemas", scim.Schemas)

		// Users
		r.Get("/scim/v2/Users", scim.ListUsers)
		r.Post("/scim/v2/Users", scim.CreateUser)
		r.Get("/scim/v2/Users/{id}", scim.GetUser)
		r.Put("/scim/v2/Users/{id}", scim.UpdateUser)
		r.Patch("/scim/v2/Users/{id}", scim.PatchUser)
		r.Delete("/scim/v2/Users/{id}", scim.DeleteUser)

		// Groups
		r.Get("/scim/v2/Groups", scim.ListGroups)
		r.Post("/scim/v2/Groups", scim.CreateGroup)
		r.Get("/scim/v2/Groups/{id}", scim.GetGroup)
		r.Put("/scim/v2/Groups/{id}", scim.UpdateGroup)
		r.Patch("/scim/v2/Groups/{id}", scim.PatchGroup)
		r.Delete("/scim/v2/Groups/{id}", scim.DeleteGroup)
	})
}

// RegisterOIDCRoutes mounts OIDC Discovery and JWKS endpoints (public, no auth).
func RegisterOIDCRoutes(r *chi.Mux, discovery, jwks http.HandlerFunc) {
	r.Get("/.well-known/openid-configuration", discovery)
	r.Get("/.well-known/jwks.json", jwks)
}

// AdminConsoleEndpoints groups the handler methods needed by RegisterAdminConsoleRoutes.
type AdminConsoleEndpoints interface {
	Dashboard(w http.ResponseWriter, r *http.Request)
	ListUsersPage(w http.ResponseWriter, r *http.Request)
	CreateUserPage(w http.ResponseWriter, r *http.Request)
	CreateUserAction(w http.ResponseWriter, r *http.Request)
	UserDetailPage(w http.ResponseWriter, r *http.Request)
	UpdateUserAction(w http.ResponseWriter, r *http.Request)
	DeleteUserAction(w http.ResponseWriter, r *http.Request)
	ResetPasswordAction(w http.ResponseWriter, r *http.Request)
	RevokeSessionsAction(w http.ResponseWriter, r *http.Request)
	ListOrgsPage(w http.ResponseWriter, r *http.Request)
	CreateOrgPage(w http.ResponseWriter, r *http.Request)
	CreateOrgAction(w http.ResponseWriter, r *http.Request)
	OrgDetailPage(w http.ResponseWriter, r *http.Request)
	UpdateOrgAction(w http.ResponseWriter, r *http.Request)
	UpdateOrgSettingsAction(w http.ResponseWriter, r *http.Request)
	DeleteOrgAction(w http.ResponseWriter, r *http.Request)
	ListClientsPage(w http.ResponseWriter, r *http.Request)
	CreateClientPage(w http.ResponseWriter, r *http.Request)
	CreateClientAction(w http.ResponseWriter, r *http.Request)
	ClientDetailPage(w http.ResponseWriter, r *http.Request)
	UpdateClientAction(w http.ResponseWriter, r *http.Request)
	DeleteClientAction(w http.ResponseWriter, r *http.Request)
	RegenerateSecretAction(w http.ResponseWriter, r *http.Request)
	ListRolesPage(w http.ResponseWriter, r *http.Request)
	CreateRolePage(w http.ResponseWriter, r *http.Request)
	CreateRoleAction(w http.ResponseWriter, r *http.Request)
	RoleDetailPage(w http.ResponseWriter, r *http.Request)
	UpdateRoleAction(w http.ResponseWriter, r *http.Request)
	DeleteRoleAction(w http.ResponseWriter, r *http.Request)
	AssignRoleAction(w http.ResponseWriter, r *http.Request)
	UnassignRoleAction(w http.ResponseWriter, r *http.Request)
	ListEventsPage(w http.ResponseWriter, r *http.Request)
	ListSessionsPage(w http.ResponseWriter, r *http.Request)
	RevokeSessionAction(w http.ResponseWriter, r *http.Request)
	RevokeAllSessionsAction(w http.ResponseWriter, r *http.Request)
	ListGroupsPage(w http.ResponseWriter, r *http.Request)
	CreateGroupPage(w http.ResponseWriter, r *http.Request)
	CreateGroupAction(w http.ResponseWriter, r *http.Request)
	GroupDetailPage(w http.ResponseWriter, r *http.Request)
	UpdateGroupAction(w http.ResponseWriter, r *http.Request)
	DeleteGroupAction(w http.ResponseWriter, r *http.Request)
	SearchUsersForGroup(w http.ResponseWriter, r *http.Request)
	AddGroupMemberAction(w http.ResponseWriter, r *http.Request)
	RemoveGroupMemberAction(w http.ResponseWriter, r *http.Request)
	AssignGroupRoleAction(w http.ResponseWriter, r *http.Request)
	UnassignGroupRoleAction(w http.ResponseWriter, r *http.Request)
	ExportOrgAction(w http.ResponseWriter, r *http.Request)
	ImportOrgPage(w http.ResponseWriter, r *http.Request)
	ImportOrgAction(w http.ResponseWriter, r *http.Request)
	OIDCPage(w http.ResponseWriter, r *http.Request)
	SocialProvidersPage(w http.ResponseWriter, r *http.Request)
	UpdateSocialProviderAction(w http.ResponseWriter, r *http.Request)
	ListWebhooksPage(w http.ResponseWriter, r *http.Request)
	CreateWebhookPage(w http.ResponseWriter, r *http.Request)
	CreateWebhookAction(w http.ResponseWriter, r *http.Request)
	WebhookDetailPage(w http.ResponseWriter, r *http.Request)
	UpdateWebhookAction(w http.ResponseWriter, r *http.Request)
	DeleteWebhookAction(w http.ResponseWriter, r *http.Request)
	ListSAMLProvidersPage(w http.ResponseWriter, r *http.Request)
	CreateSAMLProviderPage(w http.ResponseWriter, r *http.Request)
	CreateSAMLProviderAction(w http.ResponseWriter, r *http.Request)
	SAMLProviderDetailPage(w http.ResponseWriter, r *http.Request)
	UpdateSAMLProviderAction(w http.ResponseWriter, r *http.Request)
	DeleteSAMLProviderAction(w http.ResponseWriter, r *http.Request)
	PluginsPage(w http.ResponseWriter, r *http.Request)
}

// AdminLoginEndpoints groups the handler methods needed for admin OAuth login.
type AdminLoginEndpoints interface {
	Login(w http.ResponseWriter, r *http.Request)
	Callback(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
}

// RegisterAdminConsoleRoutes mounts SSR admin console routes under /admin/.
func RegisterAdminConsoleRoutes(r *chi.Mux, pubKey *rsa.PublicKey, hmacKey []byte, staticHandler http.Handler, login AdminLoginEndpoints, console AdminConsoleEndpoints) {
	// Redirect /admin to /admin/ so the dashboard is reachable without trailing slash
	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
	})

	// Static assets (CSS, JS) — no auth required, long cache
	r.Handle("/static/*", http.StripPrefix("/static/", staticHandler))

	// Public admin routes (no session required)
	r.Get("/admin/login", login.Login)
	r.Get("/admin/callback", login.Callback)

	// Protected admin routes (session cookie required + admin role)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AdminSession(pubKey, hmacKey))
		r.Use(middleware.RequireAdminSession())
		r.Use(middleware.CSRFProtect())

		r.Get("/admin/", console.Dashboard)
		r.Post("/admin/logout", login.Logout)

		// Users
		r.Get("/admin/users", console.ListUsersPage)
		r.Get("/admin/users/new", console.CreateUserPage)
		r.Post("/admin/users", console.CreateUserAction)
		r.Get("/admin/users/{id}", console.UserDetailPage)
		r.Post("/admin/users/{id}", console.UpdateUserAction)
		r.Post("/admin/users/{id}/delete", console.DeleteUserAction)
		r.Post("/admin/users/{id}/reset-password", console.ResetPasswordAction)
		r.Post("/admin/users/{id}/revoke-sessions", console.RevokeSessionsAction)

		// Organizations
		r.Get("/admin/organizations", console.ListOrgsPage)
		r.Get("/admin/organizations/new", console.CreateOrgPage)
		r.Get("/admin/organizations/import", console.ImportOrgPage)
		r.Post("/admin/organizations/import", console.ImportOrgAction)
		r.Post("/admin/organizations", console.CreateOrgAction)
		r.Get("/admin/organizations/{id}", console.OrgDetailPage)
		r.Post("/admin/organizations/{id}", console.UpdateOrgAction)
		r.Post("/admin/organizations/{id}/settings", console.UpdateOrgSettingsAction)
		r.Post("/admin/organizations/{id}/delete", console.DeleteOrgAction)
		r.Get("/admin/organizations/{id}/export", console.ExportOrgAction)

		// Roles
		r.Get("/admin/roles", console.ListRolesPage)
		r.Get("/admin/roles/new", console.CreateRolePage)
		r.Post("/admin/roles", console.CreateRoleAction)
		r.Get("/admin/roles/{id}", console.RoleDetailPage)
		r.Post("/admin/roles/{id}", console.UpdateRoleAction)
		r.Post("/admin/roles/{id}/delete", console.DeleteRoleAction)

		// User role management
		r.Post("/admin/users/{id}/roles", console.AssignRoleAction)
		r.Post("/admin/users/{id}/roles/{roleId}/delete", console.UnassignRoleAction)

		// Sessions
		r.Get("/admin/sessions", console.ListSessionsPage)
		r.Post("/admin/sessions/{id}/delete", console.RevokeSessionAction)
		r.Post("/admin/sessions/revoke-all", console.RevokeAllSessionsAction)

		// User search (autocomplete for group/role member addition)
		r.Get("/admin/users/search", console.SearchUsersForGroup)

		// Groups
		r.Get("/admin/groups", console.ListGroupsPage)
		r.Get("/admin/groups/new", console.CreateGroupPage)
		r.Post("/admin/groups", console.CreateGroupAction)
		r.Get("/admin/groups/{id}", console.GroupDetailPage)
		r.Post("/admin/groups/{id}", console.UpdateGroupAction)
		r.Post("/admin/groups/{id}/delete", console.DeleteGroupAction)
		r.Post("/admin/groups/{id}/members", console.AddGroupMemberAction)
		r.Post("/admin/groups/{id}/members/{userId}/delete", console.RemoveGroupMemberAction)
		r.Post("/admin/groups/{id}/roles", console.AssignGroupRoleAction)
		r.Post("/admin/groups/{id}/roles/{roleId}/delete", console.UnassignGroupRoleAction)

		// Webhooks
		r.Get("/admin/webhooks", console.ListWebhooksPage)
		r.Get("/admin/webhooks/new", console.CreateWebhookPage)
		r.Post("/admin/webhooks", console.CreateWebhookAction)
		r.Get("/admin/webhooks/{id}", console.WebhookDetailPage)
		r.Post("/admin/webhooks/{id}", console.UpdateWebhookAction)
		r.Post("/admin/webhooks/{id}/delete", console.DeleteWebhookAction)

		// SAML Providers
		r.Get("/admin/saml-providers", console.ListSAMLProvidersPage)
		r.Get("/admin/saml-providers/new", console.CreateSAMLProviderPage)
		r.Post("/admin/saml-providers", console.CreateSAMLProviderAction)
		r.Get("/admin/saml-providers/{id}", console.SAMLProviderDetailPage)
		r.Post("/admin/saml-providers/{id}", console.UpdateSAMLProviderAction)
		r.Post("/admin/saml-providers/{id}/delete", console.DeleteSAMLProviderAction)

		// Events
		r.Get("/admin/events", console.ListEventsPage)

		// Clients
		r.Get("/admin/clients", console.ListClientsPage)
		r.Get("/admin/clients/new", console.CreateClientPage)
		r.Post("/admin/clients", console.CreateClientAction)
		r.Get("/admin/clients/{id}", console.ClientDetailPage)
		r.Post("/admin/clients/{id}", console.UpdateClientAction)
		r.Post("/admin/clients/{id}/delete", console.DeleteClientAction)
		r.Post("/admin/clients/{id}/regenerate-secret", console.RegenerateSecretAction)

		// Plugins
		r.Get("/admin/plugins", console.PluginsPage)

		// OIDC
		r.Get("/admin/oidc", console.OIDCPage)

		// Social Providers
		r.Get("/admin/social", console.SocialProvidersPage)
		r.Post("/admin/social/{provider}", console.UpdateSocialProviderAction)
	})
}
