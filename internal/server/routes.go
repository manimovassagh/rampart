package server

import (
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
func RegisterProtectedRoutes(r *chi.Mux, jwtSecret string, meHandler http.HandlerFunc) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtSecret))
		r.Get("/me", meHandler)
	})
}
