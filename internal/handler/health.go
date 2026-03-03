package handler

import (
	"encoding/json"
	"net/http"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/database"
)

// HealthHandler provides liveness and readiness endpoints.
type HealthHandler struct {
	db *database.DB
}

// NewHealthHandler creates a handler with a database dependency for readiness checks.
func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Liveness returns 200 OK if the process is alive.
// GET /healthz
func (h *HealthHandler) Liveness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

// Readiness returns 200 OK if the server is ready to handle requests.
// Checks database connectivity.
// GET /readyz
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		apierror.ServiceUnavailable(w, "database is not reachable")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
