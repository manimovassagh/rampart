package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/manimovassagh/rampart/internal/config"
	"github.com/manimovassagh/rampart/internal/database"
	"github.com/manimovassagh/rampart/internal/handler"
	"github.com/manimovassagh/rampart/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := database.RunMigrations(cfg.DatabaseURL, "migrations", logger); err != nil {
		return err
	}

	router := server.NewRouter(logger)
	healthHandler := handler.NewHealthHandler(db)
	server.RegisterHealthRoutes(router, healthHandler.Liveness, healthHandler.Readiness)

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
