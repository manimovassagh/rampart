package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/manimovassagh/rampart/internal/middleware"
)

func TestNewRouterMiddlewareChain(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	r := NewRouter(logger, []string{"http://localhost:3000"})

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
	r := NewRouter(logger, []string{"http://localhost:3000"})

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestNewServer(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	r := NewRouter(logger, []string{"http://localhost:3000"})
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

func TestServerStartAndShutdown(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"http://localhost:3000"})

	RegisterHealthRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"alive"}`))
	}, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	// Use port 0 for OS-assigned free port
	srv := New("127.0.0.1:0", r, logger)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Graceful shutdown
	if err := srv.Shutdown(); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}

	// Start should have returned nil (ErrServerClosed is swallowed)
	if err := <-errCh; err != nil {
		t.Fatalf("start returned unexpected error: %v", err)
	}
}

func TestNewRouterCORSHeaders(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"http://localhost:3000"})

	RegisterHealthRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Preflight OPTIONS request
	req := httptest.NewRequest(http.MethodOptions, "/healthz", http.NoBody)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected Access-Control-Allow-Origin header from CORS middleware")
	}
}

func TestNewRouterReadyzEndpoint(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"http://localhost:3000"})

	RegisterHealthRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("readyz status = %d, want %d", w.Code, http.StatusOK)
	}
}
