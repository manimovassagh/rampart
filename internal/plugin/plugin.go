// Package plugin defines the extensibility interfaces and registry for Rampart plugins.
//
// Plugins are Go types that implement one or more plugin contracts (EventHook,
// ClaimEnricher, AuthMethod, MiddlewarePlugin). They are registered with the
// central Registry, which dispatches calls at the appropriate hook points.
package plugin

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// EventHook reacts to audit events after they are persisted.
// Implementations receive events asynchronously and should not block.
type EventHook interface {
	Name() string
	HandleEvent(ctx context.Context, event *model.AuditEvent) error
	// SupportedEvents returns the event types this hook listens to.
	// An empty slice means all events.
	SupportedEvents() []string
}

// ClaimEnricher adds custom claims to JWT access tokens during generation.
// The claims map is populated with standard claims before enrichers run.
type ClaimEnricher interface {
	Name() string
	EnrichClaims(ctx context.Context, userID uuid.UUID, claims map[string]any) error
}

// Route describes an HTTP route a plugin needs registered.
type Route struct {
	Method  string // "GET", "POST", etc.
	Pattern string // chi-compatible pattern, e.g. "/auth/magic-link"
	Handler http.HandlerFunc
}

// AuthResult is returned by an AuthMethod after successful authentication.
type AuthResult struct {
	UserID    uuid.UUID
	OrgID     uuid.UUID
	Username  string
	Email     string
	NewUser   bool // true if the plugin auto-provisioned a new user
	ExtraData map[string]any
}

// AuthMethod implements a custom authentication flow.
type AuthMethod interface {
	Name() string
	Authenticate(ctx context.Context, r *http.Request) (*AuthResult, error)
	Routes() []Route
}

// MiddlewarePlugin injects HTTP middleware into the request pipeline.
type MiddlewarePlugin interface {
	Name() string
	Middleware() func(http.Handler) http.Handler
	Priority() int // lower = runs first
}

// Info holds metadata about a registered plugin for admin display.
type Info struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "event_hook", "claim_enricher", "auth_method", "middleware"
	Enabled bool   `json:"enabled"`
}

// Context provides dependencies to plugins during initialization.
type Context struct {
	Logger *slog.Logger
	Config map[string]string
}
