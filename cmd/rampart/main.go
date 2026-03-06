package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/manimovassagh/rampart/internal/audit"
	"github.com/manimovassagh/rampart/internal/config"
	"github.com/manimovassagh/rampart/internal/database"
	"github.com/manimovassagh/rampart/internal/handler"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/server"
	"github.com/manimovassagh/rampart/internal/session"
	"github.com/manimovassagh/rampart/internal/signing"
	"github.com/manimovassagh/rampart/internal/social"
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

	// Load or generate RSA signing key pair
	kp, err := signing.LoadOrGenerate(cfg.SigningKeyPath)
	if err != nil {
		return err
	}
	logger.Info("signing key loaded", "kid", kp.KID, "path", cfg.SigningKeyPath)

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
	auditLogger := audit.NewLogger(db, logger)
	loginHandler := handler.NewLoginHandler(db, sessionStore, logger, auditLogger, kp.PrivateKey, kp.KID, cfg.Issuer, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	server.RegisterLoginRoutes(router, loginHandler.Login, loginHandler.Refresh, loginHandler.Logout)

	meHandler := handler.NewMeHandler(db)
	server.RegisterProtectedRoutes(router, kp.PublicKey, meHandler.Me)

	adminHandler := handler.NewAdminHandler(db, sessionStore, logger)
	server.RegisterAdminRoutes(router, kp.PublicKey, adminHandler)

	orgHandler := handler.NewOrgHandler(db, db, logger)
	server.RegisterOrgRoutes(router, kp.PublicKey, orgHandler)

	// Social login providers
	socialRegistry := social.NewRegistry()
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		socialRegistry.Register(&social.GoogleProvider{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
		})
		logger.Info("social provider registered", "provider", "google")
	}
	if cfg.GitHubClientID != "" && cfg.GitHubClientSecret != "" {
		socialRegistry.Register(&social.GitHubProvider{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
		})
		logger.Info("social provider registered", "provider", "github")
	}
	if cfg.AppleClientID != "" {
		socialRegistry.Register(&social.AppleProvider{
			ClientID: cfg.AppleClientID,
			TeamID:   cfg.AppleTeamID,
			KeyID:    cfg.AppleKeyID,
		})
		logger.Info("social provider registered", "provider", "apple")
	}

	// OAuth 2.0 Authorization Code + PKCE endpoints
	authorizeHandler := handler.NewAuthorizeHandler(db, logger, auditLogger, socialRegistry)
	tokenHandler := handler.NewTokenHandler(db, sessionStore, logger, kp.PrivateKey, kp.KID, cfg.Issuer, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	server.RegisterOAuthRoutes(router, authorizeHandler.Authorize, tokenHandler.Token)

	// OIDC Discovery + JWKS (public endpoints, no auth)
	discoveryHandler := handler.DiscoveryHandler(cfg.Issuer, logger)
	jwksHandler := handler.JWKSHandler(kp, logger)
	server.RegisterOIDCRoutes(router, discoveryHandler, jwksHandler)

	// Admin Console (SSR) — generate HMAC key early so social handler can use it too
	hmacKey, err := middleware.GenerateHMACKey()
	if err != nil {
		return err
	}
	adminLoginHandler := handler.NewAdminLoginHandler(
		db, sessionStore, logger, auditLogger,
		kp.PrivateKey, kp.PublicKey, kp.KID, cfg.Issuer,
		cfg.AccessTokenTTL, cfg.RefreshTokenTTL, hmacKey,
	)
	adminConsoleHandler := handler.NewAdminConsoleHandler(db, sessionStore, logger, cfg.Issuer, auditLogger, socialRegistry)
	server.RegisterAdminConsoleRoutes(router, kp.PublicKey, hmacKey, handler.StaticHandler(), adminLoginHandler, adminConsoleHandler)

	// Social login handler
	socialHandler := handler.NewSocialHandler(db, socialRegistry, logger, auditLogger, hmacKey, cfg.Issuer)
	server.RegisterSocialRoutes(router, socialHandler.InitiateLogin, socialHandler.Callback)

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
