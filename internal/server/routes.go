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
// Middleware order: RequestID → Recovery → CORS → Logging → RealIP
func NewRouter(logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Middleware chain — order matters
	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(logger))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", middleware.HeaderRequestID},
		ExposedHeaders:   []string{middleware.HeaderRequestID},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.Logging(logger))
	r.Use(chimw.RealIP)

	return r
}

// RegisterHealthRoutes mounts the health check endpoints.
func RegisterHealthRoutes(r *chi.Mux, healthHandler, readyHandler http.HandlerFunc) {
	r.Get("/healthz", healthHandler)
	r.Get("/readyz", readyHandler)
}
