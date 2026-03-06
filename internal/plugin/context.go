package plugin

import "time"

// AuthRequest holds data passed to pre-auth hooks.
type AuthRequest struct {
	Identifier string
	IP         string
	UserAgent  string
	ClientID   string
	Metadata   map[string]any
}

// AuthResponse is the result from a pre-auth hook.
type AuthResponse struct {
	Allow    bool
	Reason   string
	Metadata map[string]any
}

// AuthResult holds the outcome of an authentication attempt.
type AuthResult struct {
	UserID   string
	Username string
	Success  bool
	Metadata map[string]any
}

// RegistrationRequest holds data passed to pre-register hooks.
type RegistrationRequest struct {
	Username string
	Email    string
	Metadata map[string]any
}

// RegistrationResponse is the result from a pre-register hook.
type RegistrationResponse struct {
	Allow    bool
	Reason   string
	Metadata map[string]any
}

// Event represents an audit event passed to event hooks.
type Event struct {
	Type      string
	OrgID     string
	ActorID   string
	TargetID  string
	Timestamp time.Time
	Data      map[string]any
}

// PluginInfo is a safe struct for listing loaded plugins without exposing internals.
type PluginInfo struct {
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	Description string      `json:"description"`
	Hooks       []HookPoint `json:"hooks"`
}
