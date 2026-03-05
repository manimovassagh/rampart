package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockPinger struct {
	err error
}

// errResponseWriter simulates a broken connection where Write fails.
type errResponseWriter struct {
	header http.Header
	status int
}

func (w *errResponseWriter) Header() http.Header  { return w.header }
func (w *errResponseWriter) WriteHeader(code int) { w.status = code }
func (w *errResponseWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("broken pipe")
}

func (m *mockPinger) Ping(_ context.Context) error {
	return m.err
}

func TestLiveness(t *testing.T) {
	h := NewHealthHandler(&mockPinger{})
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

func TestReadinessHealthy(t *testing.T) {
	h := NewHealthHandler(&mockPinger{err: nil})
	req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
	w := httptest.NewRecorder()

	h.Readiness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["status"] != "ready" {
		t.Errorf("status = %q, want ready", body["status"])
	}
}

func TestLivenessEncodeError(t *testing.T) {
	h := NewHealthHandler(&mockPinger{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	w := &errResponseWriter{header: make(http.Header)}

	// Should not panic — logs the error internally
	h.Liveness(w, req)
}

func TestReadinessEncodeError(t *testing.T) {
	h := NewHealthHandler(&mockPinger{err: nil})
	req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
	w := &errResponseWriter{header: make(http.Header)}

	// Should not panic — logs the error internally
	h.Readiness(w, req)
}

func TestReadinessUnhealthy(t *testing.T) {
	h := NewHealthHandler(&mockPinger{err: fmt.Errorf("connection refused")})
	req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
	w := httptest.NewRecorder()

	h.Readiness(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
