package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/manimovassagh/rampart/internal/apierror"
)

const statusAlive = "alive"
const statusReady = "ready"

// Pinger checks database connectivity.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler provides liveness and readiness endpoints.
type HealthHandler struct {
	db Pinger
}

// NewHealthHandler creates a handler with a database dependency for readiness checks.
func NewHealthHandler(db Pinger) *HealthHandler {
	return &HealthHandler{db: db}
}

// Liveness returns 200 OK if the process is alive.
// GET /healthz
func (h *HealthHandler) Liveness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": statusAlive}); err != nil {
		slog.Error("failed to encode liveness response", "error", err)
	}
}

// Readiness returns 200 OK if the server is ready to handle requests.
// Checks database connectivity.
// GET /readyz
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		apierror.ServiceUnavailable(w, "database is not reachable")
		return
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": statusReady}); err != nil {
		slog.Error("failed to encode readiness response", "error", err)
	}
}
