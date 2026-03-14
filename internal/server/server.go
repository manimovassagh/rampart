package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// Server wraps the HTTP server with graceful shutdown.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// New creates a Server bound to the given address with the provided router.
func New(addr string, router *chi.Mux, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           router,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		logger: logger,
	}
}

// Start begins listening. It blocks until the server is shut down.
func (s *Server) Start() error {
	s.logger.Info("server starting", "addr", s.httpServer.Addr)
	err := s.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server listen: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the server with a 10-second deadline.
func (s *Server) Shutdown() error {
	s.logger.Info("server shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	s.logger.Info("server stopped")
	return nil
}
