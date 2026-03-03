package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/manimovassagh/rampart/internal/middleware"
)

func TestNewRouterMiddlewareChain(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	r := NewRouter(logger)

	RegisterHealthRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"alive"}`))
	}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("healthz status = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Header().Get(middleware.HeaderRequestID) == "" {
		t.Error("expected X-Request-Id header from middleware")
	}
}

func TestNewRouterNotFound(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	r := NewRouter(logger)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestNewServer(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	r := NewRouter(logger)
	s := New(":0", r, logger)

	if s.httpServer.ReadTimeout != readTimeout {
		t.Errorf("ReadTimeout = %v, want %v", s.httpServer.ReadTimeout, readTimeout)
	}
	if s.httpServer.WriteTimeout != writeTimeout {
		t.Errorf("WriteTimeout = %v, want %v", s.httpServer.WriteTimeout, writeTimeout)
	}
	if s.httpServer.IdleTimeout != idleTimeout {
		t.Errorf("IdleTimeout = %v, want %v", s.httpServer.IdleTimeout, idleTimeout)
	}
}
