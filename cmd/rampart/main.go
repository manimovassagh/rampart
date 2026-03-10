package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/manimovassagh/rampart/internal/audit"
	"github.com/manimovassagh/rampart/internal/cluster"
	"github.com/manimovassagh/rampart/internal/plugin"
	webhookpkg "github.com/manimovassagh/rampart/internal/webhook"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/manimovassagh/rampart/internal/config"
	"github.com/manimovassagh/rampart/internal/crypto"
	"github.com/manimovassagh/rampart/internal/database"
	"github.com/manimovassagh/rampart/internal/email"
	"github.com/manimovassagh/rampart/internal/handler"
	"github.com/manimovassagh/rampart/internal/logging"
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

	// Configure cookie security based on environment
	middleware.SetSecureCookies(cfg.SecureCookies)

	// Reconfigure logger with the loaded log level and format
	logOpts := &slog.HandlerOptions{Level: parseLogLevel(string(cfg.LogLevel))}
	var logHandler slog.Handler
	switch cfg.LogFormat {
	case config.LogFormatJSON:
		logHandler = slog.NewJSONHandler(os.Stdout, logOpts)
	case config.LogFormatText:
		logHandler = slog.NewTextHandler(os.Stdout, logOpts)
	case config.LogFormatPretty:
		logHandler = logging.NewPrettyHandler(os.Stdout, logOpts)
	}
	logger := slog.New(logHandler)
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

	// Set up encryption for secrets at rest (social tokens, client secrets)
	if cfg.EncryptionKey != "" {
		keyBytes, err := hex.DecodeString(cfg.EncryptionKey)
		if err != nil {
			return fmt.Errorf("RAMPART_ENCRYPTION_KEY must be valid hex: %w", err)
		}
		enc, err := crypto.NewEncryptor(keyBytes)
		if err != nil {
			return fmt.Errorf("invalid encryption key: %w", err)
		}
		db.Encryptor = enc
		logger.Info("encryption at rest enabled for social tokens and secrets")
	}

	if err := database.RunMigrations(cfg.DatabaseURL, "migrations", logger); err != nil {
		return err
	}

	router := server.NewRouter(logger, cfg.AllowedOrigins, cfg.HSTSEnabled)
	healthHandler := handler.NewHealthHandler(db)
	server.RegisterHealthRoutes(router, healthHandler.Liveness, healthHandler.Readiness)
	server.RegisterMetricsRoutes(router)

	// Rate limiters for auth endpoints
	loginRL := middleware.NewRateLimiter(cfg.RateLimit.LoginPerMinute, time.Minute)
	defer loginRL.Close()
	registerRL := middleware.NewRateLimiter(cfg.RateLimit.RegisterPerMinute, time.Minute)
	defer registerRL.Close()
	tokenRL := middleware.NewRateLimiter(cfg.RateLimit.TokenPerMinute, time.Minute)
	defer tokenRL.Close()
	logger.Info("rate limiting enabled",
		"login_per_min", cfg.RateLimit.LoginPerMinute,
		"register_per_min", cfg.RateLimit.RegisterPerMinute,
		"token_per_min", cfg.RateLimit.TokenPerMinute,
	)

	registerHandler := handler.NewRegisterHandler(db, logger)
	server.RegisterAuthRoutes(router, registerHandler.Register, registerRL)

	sessionStore := session.NewPGStore(db.Pool)
	auditLogger := audit.NewLogger(db, logger)

	// Plugin registry — extensibility layer for event hooks, claim enrichers, etc.
	pluginRegistry := plugin.NewRegistry(logger)
	defer func() { _ = pluginRegistry.Close() }()

	// Webhook dispatcher — delivers audit events to registered webhook endpoints
	webhookDispatcher := webhookpkg.NewDispatcher(db, logger)
	auditLogger.SetDispatcher(webhookDispatcher)
	auditLogger.SetPluginDispatcher(pluginRegistry)
	loginHandler := handler.NewLoginHandler(db, sessionStore, logger, auditLogger, kp.PrivateKey, kp.KID, cfg.Issuer, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	server.RegisterLoginRoutes(router, loginHandler.Login, loginHandler.Refresh, loginHandler.Logout, loginRL)

	// Password reset (forgot-password / reset-password)
	var emailSender handler.EmailSender
	if cfg.SMTPHost != "" && cfg.SMTPFrom != "" {
		emailSender = email.NewSender(email.Config{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
		})
		logger.Info("SMTP configured for transactional emails", "host", cfg.SMTPHost)
	} else {
		emailSender = &email.NoOpSender{}
		logger.Warn("SMTP not configured — password reset tokens will be logged instead of emailed")
	}
	resetHandler := handler.NewPasswordResetHandler(db, emailSender, logger, cfg.Issuer)
	server.RegisterPasswordResetRoutes(router, resetHandler.ForgotPassword, resetHandler.ResetPassword, loginRL)

	// Email verification
	emailVerifyHandler := handler.NewEmailVerificationHandler(db, emailSender, logger, cfg.Issuer)
	registerHandler.SetEmailVerifier(emailVerifyHandler)
	server.RegisterEmailVerificationRoutes(router, emailVerifyHandler.SendVerification, emailVerifyHandler.VerifyEmail, loginRL)

	// MFA endpoints (enrollment + login verification)
	mfaHandler := handler.NewMFAHandler(db, logger, cfg.Issuer)
	mfaVerifyHandler := handler.NewMFAVerifyHandler(db, sessionStore, logger, auditLogger, kp.PrivateKey, kp.PublicKey, kp.KID, cfg.Issuer, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)

	// WebAuthn/Passkey configuration — derive RPID and origins from issuer URL
	rpID := extractHost(cfg.Issuer)
	wa, err := gowebauthn.New(&gowebauthn.Config{
		RPID:          rpID,
		RPDisplayName: "Rampart",
		RPOrigins:     append([]string{cfg.Issuer}, cfg.AllowedOrigins...),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize WebAuthn: %w", err)
	}
	webauthnHandler := handler.NewWebAuthnHandler(db, sessionStore, logger, auditLogger, wa, kp.PrivateKey, kp.PublicKey, kp.KID, cfg.Issuer, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	server.RegisterMFARoutes(router, kp.PublicKey, mfaHandler, mfaVerifyHandler.VerifyTOTP, webauthnHandler, loginRL)

	meHandler := handler.NewMeHandler(db)
	server.RegisterProtectedRoutes(router, kp.PublicKey, meHandler.Me)

	adminHandler := handler.NewAdminHandler(db, sessionStore, logger)
	server.RegisterAdminRoutes(router, kp.PublicKey, adminHandler)

	orgHandler := handler.NewOrgHandler(db, db, logger)
	server.RegisterOrgRoutes(router, kp.PublicKey, orgHandler)

	// Social login providers — load from env vars first, then override with DB configs
	socialRegistry := social.NewRegistry()
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		socialRegistry.Register(&social.GoogleProvider{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
		})
		logger.Info("social provider registered", "provider", "google", "source", "env")
	}
	if cfg.GitHubClientID != "" && cfg.GitHubClientSecret != "" {
		socialRegistry.Register(&social.GitHubProvider{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
		})
		logger.Info("social provider registered", "provider", "github", "source", "env")
	}
	if cfg.AppleClientID != "" {
		socialRegistry.Register(&social.AppleProvider{
			ClientID: cfg.AppleClientID,
			TeamID:   cfg.AppleTeamID,
			KeyID:    cfg.AppleKeyID,
		})
		logger.Info("social provider registered", "provider", "apple", "source", "env")
	}

	// Load social provider configs from database (admin dashboard settings)
	defaultOrgID, err := db.GetDefaultOrganizationID(context.Background())
	if err != nil {
		logger.Warn("failed to get default org for social provider loading", "error", err)
	} else {
		dbConfigs, err := db.ListSocialProviderConfigs(context.Background(), defaultOrgID)
		if err != nil {
			logger.Warn("failed to load social provider configs from database", "error", err)
		} else {
			for _, sc := range dbConfigs {
				if !sc.Enabled || sc.ClientID == "" || sc.ClientSecret == "" {
					continue
				}
				switch sc.Provider {
				case "google":
					socialRegistry.Register(&social.GoogleProvider{
						ClientID:     sc.ClientID,
						ClientSecret: sc.ClientSecret,
					})
				case "github":
					socialRegistry.Register(&social.GitHubProvider{
						ClientID:     sc.ClientID,
						ClientSecret: sc.ClientSecret,
					})
				case "apple":
					socialRegistry.Register(&social.AppleProvider{
						ClientID: sc.ClientID,
						TeamID:   sc.ExtraConfig["team_id"],
						KeyID:    sc.ExtraConfig["key_id"],
					})
				}
				logger.Info("social provider registered", "provider", sc.Provider, "source", "database")
			}
		}
	}

	// OAuth 2.0 Authorization Code + PKCE endpoints
	authorizeHandler := handler.NewAuthorizeHandler(db, logger, auditLogger, socialRegistry)
	tokenHandler := handler.NewTokenHandler(db, sessionStore, logger, kp.PrivateKey, kp.KID, cfg.Issuer, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	server.RegisterOAuthRoutes(router, authorizeHandler.Authorize, authorizeHandler.Consent, tokenHandler.Token, tokenHandler.Revoke, tokenRL)

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
	adminConsoleHandler := handler.NewAdminConsoleHandler(db, sessionStore, logger, cfg.Issuer, auditLogger, socialRegistry, pluginRegistry)
	server.RegisterAdminConsoleRoutes(router, kp.PublicKey, hmacKey, handler.StaticHandler(), adminLoginHandler, adminConsoleHandler)

	// SCIM 2.0 provisioning endpoints (for Okta, Azure AD, etc.)
	scimHandler := handler.NewSCIMHandler(db, logger)
	server.RegisterSCIMRoutes(router, kp.PublicKey, scimHandler)

	// Compliance reporting endpoints (SOC2, GDPR, HIPAA)
	complianceHandler := handler.NewComplianceHandler(db, logger)
	server.RegisterComplianceRoutes(router, kp.PublicKey, complianceHandler)

	// SAML 2.0 SP endpoints for enterprise SSO
	samlCert, err := handler.ParseCertFromKey(kp.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to generate SAML SP certificate: %w", err)
	}
	samlHandler := handler.NewSAMLHandler(db, sessionStore, logger, auditLogger, kp.PrivateKey, samlCert, kp.KID, cfg.Issuer, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	server.RegisterSAMLRoutes(router, samlHandler)

	// Social login handler
	socialHandler := handler.NewSocialHandler(db, socialRegistry, logger, auditLogger, hmacKey, cfg.Issuer)
	server.RegisterSocialRoutes(router, socialHandler.InitiateLogin, socialHandler.Callback)

	// Leader election — ensures only one instance runs background workers in HA deployments.
	// Uses PostgreSQL advisory locks; safe for single-instance too (always becomes leader).
	leaderElector := cluster.NewLeader(db.Pool, logger)
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		leaderElector.Run(cleanupCtx)
	}()

	// Background cleanup for expired authorization codes (leader-only)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-cleanupCtx.Done():
				return
			case <-ticker.C:
				if !leaderElector.IsLeader() {
					continue
				}
				if n, err := db.DeleteExpiredAuthorizationCodes(cleanupCtx); err != nil {
					logger.Warn("failed to clean up expired auth codes", "error", err)
				} else if n > 0 {
					logger.Info("cleaned up expired authorization codes", "count", n)
				}
				if n, err := db.DeleteExpiredPasswordResetTokens(cleanupCtx); err != nil {
					logger.Warn("failed to clean up expired reset tokens", "error", err)
				} else if n > 0 {
					logger.Info("cleaned up expired password reset tokens", "count", n)
				}
				if n, err := db.DeleteExpiredEmailVerificationTokens(cleanupCtx); err != nil {
					logger.Warn("failed to clean up expired verification tokens", "error", err)
				} else if n > 0 {
					logger.Info("cleaned up expired email verification tokens", "count", n)
				}
				if n, err := db.DeleteExpiredWebAuthnSessions(cleanupCtx); err != nil {
					logger.Warn("failed to clean up expired webauthn sessions", "error", err)
				} else if n > 0 {
					logger.Info("cleaned up expired webauthn sessions", "count", n)
				}
				if n, err := db.DeleteOldDeliveries(cleanupCtx, 7*24*time.Hour); err != nil {
					logger.Warn("failed to clean up old webhook deliveries", "error", err)
				} else if n > 0 {
					logger.Info("cleaned up old webhook deliveries", "count", n)
				}
			}
		}
	}()

	// Background worker for webhook delivery retries (leader-only)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-cleanupCtx.Done():
				return
			case <-ticker.C:
				if !leaderElector.IsLeader() {
					continue
				}
				webhookDispatcher.ProcessPending(cleanupCtx)
			}
		}
	}()

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

	// Graceful shutdown sequence:
	// 1. Cancel background worker context so goroutines stop issuing DB queries
	cleanupCancel()
	// 2. Wait for all background goroutines to finish
	wg.Wait()
	logger.Info("background workers stopped")
	// 3. Shutdown HTTP server (drains in-flight requests)
	// 4. db.Close() fires via defer — safe now that all workers have exited
	return srv.Shutdown()
}

// extractHost parses a URL and returns just the hostname (without scheme or port).
func extractHost(issuer string) string {
	u, err := url.Parse(issuer)
	if err != nil {
		return "localhost"
	}
	return u.Hostname()
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
