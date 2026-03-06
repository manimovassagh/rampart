package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Registry manages plugin lifecycle and hook dispatch.
type Registry struct {
	plugins    []Plugin
	authHooks  []AuthHook
	tokenHooks []TokenHook
	regHooks   []RegistrationHook
	eventHooks []EventHook
	routeHooks []RouteHook
	logger     *slog.Logger
	mu         sync.RWMutex
}

// NewRegistry creates a new plugin registry.
func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		logger: logger,
	}
}

// Register initializes a plugin and adds it to the registry.
// The plugin is categorized by the hook interfaces it implements.
func (r *Registry) Register(p Plugin, config map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate names
	for _, existing := range r.plugins {
		if existing.Name() == p.Name() {
			return fmt.Errorf("plugin %q already registered", p.Name())
		}
	}

	if err := p.Init(context.Background(), config); err != nil {
		return fmt.Errorf("initializing plugin %q: %w", p.Name(), err)
	}

	r.plugins = append(r.plugins, p)

	if h, ok := p.(AuthHook); ok {
		r.authHooks = append(r.authHooks, h)
	}
	if h, ok := p.(TokenHook); ok {
		r.tokenHooks = append(r.tokenHooks, h)
	}
	if h, ok := p.(RegistrationHook); ok {
		r.regHooks = append(r.regHooks, h)
	}
	if h, ok := p.(EventHook); ok {
		r.eventHooks = append(r.eventHooks, h)
	}
	if h, ok := p.(RouteHook); ok {
		r.routeHooks = append(r.routeHooks, h)
	}

	r.logger.Info("plugin registered",
		"name", p.Name(),
		"version", p.Version(),
		"hooks", p.Hooks(),
	)

	return nil
}

// Shutdown gracefully shuts down all registered plugins in reverse order.
func (r *Registry) Shutdown(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var errs []error
	for i := len(r.plugins) - 1; i >= 0; i-- {
		p := r.plugins[i]
		if err := safeShutdown(ctx, p); err != nil {
			errs = append(errs, fmt.Errorf("shutting down plugin %q: %w", p.Name(), err))
			r.logger.Error("plugin shutdown failed", "name", p.Name(), "error", err)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// RunPreAuth runs all pre-auth hooks. If any hook denies, returns immediately.
func (r *Registry) RunPreAuth(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, h := range r.authHooks {
		resp, err := safePreAuth(ctx, h, req)
		if err != nil {
			r.logger.Error("pre-auth hook error", "error", err)
			continue
		}
		if resp != nil && !resp.Allow {
			return resp, nil
		}
	}
	return &AuthResponse{Allow: true}, nil
}

// RunPostAuth runs all post-auth hooks.
func (r *Registry) RunPostAuth(ctx context.Context, req *AuthRequest, result *AuthResult) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, h := range r.authHooks {
		if err := safePostAuth(ctx, h, req, result); err != nil {
			r.logger.Error("post-auth hook error", "error", err)
		}
	}
	return nil
}

// RunCustomClaims runs all custom claims hooks, merging additional claims.
func (r *Registry) RunCustomClaims(ctx context.Context, userID string, claims map[string]any) (map[string]any, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]any, len(claims))
	for k, v := range claims {
		result[k] = v
	}

	for _, h := range r.tokenHooks {
		extra, err := safeCustomClaims(ctx, h, userID, result)
		if err != nil {
			r.logger.Error("custom-claims hook error", "error", err)
			continue
		}
		for k, v := range extra {
			result[k] = v
		}
	}
	return result, nil
}

// RunPreRegister runs all pre-register hooks. If any hook denies, returns immediately.
func (r *Registry) RunPreRegister(ctx context.Context, req *RegistrationRequest) (*RegistrationResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, h := range r.regHooks {
		resp, err := safePreRegister(ctx, h, req)
		if err != nil {
			r.logger.Error("pre-register hook error", "error", err)
			continue
		}
		if resp != nil && !resp.Allow {
			return resp, nil
		}
	}
	return &RegistrationResponse{Allow: true}, nil
}

// RunPostRegister runs all post-register hooks.
func (r *Registry) RunPostRegister(ctx context.Context, userID string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, h := range r.regHooks {
		if err := safePostRegister(ctx, h, userID); err != nil {
			r.logger.Error("post-register hook error", "error", err)
		}
	}
	return nil
}

// RunEvent dispatches an event to all event hooks.
func (r *Registry) RunEvent(ctx context.Context, event *Event) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, h := range r.eventHooks {
		if err := safeEvent(ctx, h, event); err != nil {
			r.logger.Error("event hook error", "error", err)
		}
	}
	return nil
}

// Routes returns all custom routes from route hook plugins.
func (r *Registry) Routes() []Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var routes []Route
	for _, h := range r.routeHooks {
		routes = append(routes, h.Routes()...)
	}
	return routes
}

// Plugins returns metadata for all loaded plugins.
func (r *Registry) Plugins() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]PluginInfo, 0, len(r.plugins))
	for _, p := range r.plugins {
		infos = append(infos, PluginInfo{
			Name:        p.Name(),
			Version:     p.Version(),
			Description: p.Description(),
			Hooks:       p.Hooks(),
		})
	}
	return infos
}

// safeShutdown calls Shutdown recovering from panics.
func safeShutdown(ctx context.Context, p Plugin) (err error) {
	defer func() {
		if rv := recover(); rv != nil {
			err = fmt.Errorf("panic in plugin shutdown: %v", rv)
		}
	}()
	return p.Shutdown(ctx)
}

// safePreAuth calls OnPreAuth recovering from panics.
func safePreAuth(ctx context.Context, h AuthHook, req *AuthRequest) (resp *AuthResponse, err error) {
	defer func() {
		if rv := recover(); rv != nil {
			err = fmt.Errorf("panic in pre-auth hook: %v", rv)
		}
	}()
	return h.OnPreAuth(ctx, req)
}

// safePostAuth calls OnPostAuth recovering from panics.
func safePostAuth(ctx context.Context, h AuthHook, req *AuthRequest, result *AuthResult) (err error) {
	defer func() {
		if rv := recover(); rv != nil {
			err = fmt.Errorf("panic in post-auth hook: %v", rv)
		}
	}()
	return h.OnPostAuth(ctx, req, result)
}

// safeCustomClaims calls OnCustomClaims recovering from panics.
func safeCustomClaims(ctx context.Context, h TokenHook, userID string, claims map[string]any) (extra map[string]any, err error) {
	defer func() {
		if rv := recover(); rv != nil {
			err = fmt.Errorf("panic in custom-claims hook: %v", rv)
		}
	}()
	return h.OnCustomClaims(ctx, userID, claims)
}

// safePreRegister calls OnPreRegister recovering from panics.
func safePreRegister(ctx context.Context, h RegistrationHook, req *RegistrationRequest) (resp *RegistrationResponse, err error) {
	defer func() {
		if rv := recover(); rv != nil {
			err = fmt.Errorf("panic in pre-register hook: %v", rv)
		}
	}()
	return h.OnPreRegister(ctx, req)
}

// safePostRegister calls OnPostRegister recovering from panics.
func safePostRegister(ctx context.Context, h RegistrationHook, userID string) (err error) {
	defer func() {
		if rv := recover(); rv != nil {
			err = fmt.Errorf("panic in post-register hook: %v", rv)
		}
	}()
	return h.OnPostRegister(ctx, userID)
}

// safeEvent calls OnEvent recovering from panics.
func safeEvent(ctx context.Context, h EventHook, event *Event) (err error) {
	defer func() {
		if rv := recover(); rv != nil {
			err = fmt.Errorf("panic in event hook: %v", rv)
		}
	}()
	return h.OnEvent(ctx, event)
}
