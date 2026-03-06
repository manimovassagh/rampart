package server

import (
	"crypto/rsa"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/manimovassagh/rampart/internal/apierror"
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
	if rl != nil {
		r.With(middleware.RequireJSON, rl.Middleware()).Post("/login", loginHandler)
	} else {
		r.With(middleware.RequireJSON).Post("/login", loginHandler)
	}
	r.With(middleware.RequireJSON).Post("/token/refresh", refreshHandler)
	r.With(middleware.RequireJSON).Post("/logout", logoutHandler)
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

		r.Get("/api/v1/admin/organizations/{id}/export", exportHandler)
		r.Post("/api/v1/admin/organizations/{id}/import", importHandler)
	})
}

// RegisterOAuthRoutes mounts the OAuth 2.0 authorization and token endpoints.
// If rl is non-nil, it is applied as rate limiting middleware to /oauth/token.
func RegisterOAuthRoutes(r *chi.Mux, authorize, token http.HandlerFunc, rl *middleware.RateLimiter) {
	r.Get("/oauth/authorize", authorize)
	r.Post("/oauth/authorize", authorize)
	if rl != nil {
		r.With(rl.Middleware()).Post("/oauth/token", token)
	} else {
		r.Post("/oauth/token", token)
	}
}

// RegisterSocialRoutes mounts social login initiation and callback endpoints.
func RegisterSocialRoutes(r *chi.Mux, initiate, callback http.HandlerFunc) {
	r.Get("/oauth/social/{provider}", initiate)
	r.Get("/oauth/social/{provider}/callback", callback)
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

		// OIDC
		r.Get("/admin/oidc", console.OIDCPage)

		// Social Providers
		r.Get("/admin/social", console.SocialProvidersPage)
		r.Post("/admin/social/{provider}", console.UpdateSocialProviderAction)
	})
}
