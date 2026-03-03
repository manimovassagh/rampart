package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manimovassagh/rampart/internal/database"
)

func TestLiveness(t *testing.T) {
	h := NewHealthHandler(&database.DB{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	w := httptest.NewRecorder()

	h.Liveness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["status"] != "alive" {
		t.Errorf("status = %q, want alive", body["status"])
	}
}

func TestReadinessNoPool(t *testing.T) {
	// DB with nil pool simulates a disconnected database
	h := NewHealthHandler(&database.DB{})
	req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
	w := httptest.NewRecorder()

	h.Readiness(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
