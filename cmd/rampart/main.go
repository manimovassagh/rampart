package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/manimovassagh/rampart/internal/config"
	"github.com/manimovassagh/rampart/internal/database"
	"github.com/manimovassagh/rampart/internal/handler"
	"github.com/manimovassagh/rampart/internal/server"
	"github.com/manimovassagh/rampart/internal/session"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(_ *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Reconfigure logger with the loaded log level
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))
	slog.SetDefault(logger)

	ctx := context.Background()
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := database.RunMigrations(cfg.DatabaseURL, "migrations", logger); err != nil {
		return err
	}

	router := server.NewRouter(logger, cfg.AllowedOrigins)
	healthHandler := handler.NewHealthHandler(db)
	server.RegisterHealthRoutes(router, healthHandler.Liveness, healthHandler.Readiness)

	registerHandler := handler.NewRegisterHandler(db, logger)
	server.RegisterAuthRoutes(router, registerHandler.Register)

	sessionStore := session.NewPGStore(db.Pool)
	loginHandler := handler.NewLoginHandler(db, sessionStore, logger, cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	server.RegisterLoginRoutes(router, loginHandler.Login, loginHandler.Refresh, loginHandler.Logout)

	server.RegisterProtectedRoutes(router, cfg.JWTSecret, handler.Me)

	srv := server.New(cfg.Addr(), router, logger)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("received signal, shutting down", "signal", sig)
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	return srv.Shutdown()
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
