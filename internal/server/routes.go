package server

import (
	"crypto/rsa"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/manimovassagh/rampart/internal/middleware"
)

// NewRouter creates and configures the chi router with middleware chain.
// Middleware order: RequestID → RealIP → Recovery → CORS → Logging
func NewRouter(logger *slog.Logger, allowedOrigins []string) *chi.Mux {
	r := chi.NewRouter()

	// Middleware chain — order matters
	r.Use(middleware.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Recovery(logger))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", middleware.HeaderRequestID},
		ExposedHeaders:   []string{middleware.HeaderRequestID},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.Logging(logger))

	return r
}

// RegisterHealthRoutes mounts the health check endpoints.
func RegisterHealthRoutes(r *chi.Mux, healthHandler, readyHandler http.HandlerFunc) {
	r.Get("/healthz", healthHandler)
	r.Get("/readyz", readyHandler)
}

// RegisterAuthRoutes mounts authentication-related endpoints.
func RegisterAuthRoutes(r *chi.Mux, registerHandler http.HandlerFunc) {
	r.Post("/register", registerHandler)
}

// RegisterLoginRoutes mounts login, token refresh, and logout endpoints.
func RegisterLoginRoutes(r *chi.Mux, loginHandler, refreshHandler, logoutHandler http.HandlerFunc) {
	r.Post("/login", loginHandler)
	r.Post("/token/refresh", refreshHandler)
	r.Post("/logout", logoutHandler)
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

		r.Get("/api/v1/admin/organizations", org.ListOrgs)
		r.Post("/api/v1/admin/organizations", org.CreateOrg)
		r.Get("/api/v1/admin/organizations/{id}", org.GetOrg)
		r.Put("/api/v1/admin/organizations/{id}", org.UpdateOrg)
		r.Delete("/api/v1/admin/organizations/{id}", org.DeleteOrg)
		r.Get("/api/v1/admin/organizations/{id}/settings", org.GetOrgSettings)
		r.Put("/api/v1/admin/organizations/{id}/settings", org.UpdateOrgSettings)
	})
}

// RegisterOIDCRoutes mounts OIDC Discovery and JWKS endpoints (public, no auth).
func RegisterOIDCRoutes(r *chi.Mux, discovery, jwks http.HandlerFunc) {
	r.Get("/.well-known/openid-configuration", discovery)
	r.Get("/.well-known/jwks.json", jwks)
}
