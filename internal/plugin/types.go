package plugin

import (
	"context"
	"net/http"
)

// HookPoint defines when a plugin hook fires.
type HookPoint string

const (
	// HookPreAuth fires before authentication.
	HookPreAuth HookPoint = "pre_auth"
	// HookPostAuth fires after successful authentication.
	HookPostAuth HookPoint = "post_auth"
	// HookPreTokenIssue fires before token generation.
	HookPreTokenIssue HookPoint = "pre_token_issue"
	// HookPostTokenIssue fires after token generation.
	HookPostTokenIssue HookPoint = "post_token_issue"
	// HookPreRegister fires before user registration.
	HookPreRegister HookPoint = "pre_register"
	// HookPostRegister fires after user registration.
	HookPostRegister HookPoint = "post_register"
	// HookCustomClaims fires to add custom claims to tokens.
	HookCustomClaims HookPoint = "custom_claims"
	// HookEvent fires on audit events.
	HookEvent HookPoint = "event"
)

// Plugin is the main plugin interface. All plugins must implement this.
type Plugin interface {
	// Name returns the unique plugin name.
	Name() string
	// Version returns the plugin version string.
	Version() string
	// Description returns a human-readable description.
	Description() string

	// Init initializes the plugin with the given configuration.
	Init(ctx context.Context, config map[string]any) error
	// Shutdown gracefully shuts down the plugin.
	Shutdown(ctx context.Context) error

	// Hooks returns the hook points this plugin handles.
	Hooks() []HookPoint
}

// AuthHook is called during authentication flows.
type AuthHook interface {
	OnPreAuth(ctx context.Context, req *AuthRequest) (*AuthResponse, error)
	OnPostAuth(ctx context.Context, req *AuthRequest, result *AuthResult) error
}

// TokenHook is called during token issuance.
type TokenHook interface {
	OnCustomClaims(ctx context.Context, userID string, existingClaims map[string]any) (map[string]any, error)
	OnPreTokenIssue(ctx context.Context, claims map[string]any) (map[string]any, error)
}

// RegistrationHook is called during user registration.
type RegistrationHook interface {
	OnPreRegister(ctx context.Context, req *RegistrationRequest) (*RegistrationResponse, error)
	OnPostRegister(ctx context.Context, userID string) error
}

// EventHook receives audit events.
type EventHook interface {
	OnEvent(ctx context.Context, event *Event) error
}

// RouteHook allows plugins to register custom HTTP routes.
type RouteHook interface {
	Routes() []Route
}

// Route defines a custom HTTP route registered by a plugin.
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}
