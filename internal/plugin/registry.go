package plugin

import (
	"context"
	"io"
	"log/slog"
	"sort"
	"sync"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// Registry manages all registered plugins and dispatches calls to them.
type Registry struct {
	mu          sync.RWMutex
	eventHooks  []EventHook
	enrichers   []ClaimEnricher
	authMethods map[string]AuthMethod
	middlewares []MiddlewarePlugin
	logger      *slog.Logger
}

// NewRegistry creates a new plugin registry.
func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		authMethods: make(map[string]AuthMethod),
		logger:      logger,
	}
}

// RegisterEventHook adds an event hook plugin.
func (r *Registry) RegisterEventHook(hook EventHook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.eventHooks = append(r.eventHooks, hook)
	r.logger.Info("plugin registered", "type", "event_hook", "name", hook.Name())
}

// RegisterClaimEnricher adds a claim enricher plugin.
func (r *Registry) RegisterClaimEnricher(enricher ClaimEnricher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enrichers = append(r.enrichers, enricher)
	r.logger.Info("plugin registered", "type", "claim_enricher", "name", enricher.Name())
}

// RegisterAuthMethod adds a custom auth method plugin.
func (r *Registry) RegisterAuthMethod(method AuthMethod) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.authMethods[method.Name()] = method
	r.logger.Info("plugin registered", "type", "auth_method", "name", method.Name())
}

// RegisterMiddleware adds a middleware plugin.
func (r *Registry) RegisterMiddleware(mw MiddlewarePlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewares = append(r.middlewares, mw)
	sort.Slice(r.middlewares, func(i, j int) bool {
		return r.middlewares[i].Priority() < r.middlewares[j].Priority()
	})
	r.logger.Info("plugin registered", "type", "middleware", "name", mw.Name())
}

// DispatchEvent sends an audit event to all registered event hooks.
// Errors are logged but do not stop dispatch to other hooks.
func (r *Registry) DispatchEvent(ctx context.Context, event *model.AuditEvent) {
	r.mu.RLock()
	hooks := make([]EventHook, len(r.eventHooks))
	copy(hooks, r.eventHooks)
	r.mu.RUnlock()

	for _, hook := range hooks {
		supported := hook.SupportedEvents()
		if len(supported) > 0 && !containsStr(supported, event.EventType) {
			continue
		}
		if err := hook.HandleEvent(ctx, event); err != nil {
			r.logger.Warn("plugin event hook failed",
				"plugin", hook.Name(),
				"event_type", event.EventType,
				"error", err,
			)
		}
	}
}

// EnrichClaims runs all registered claim enrichers against the claims map.
func (r *Registry) EnrichClaims(ctx context.Context, userID uuid.UUID, claims map[string]any) error {
	r.mu.RLock()
	enrichers := make([]ClaimEnricher, len(r.enrichers))
	copy(enrichers, r.enrichers)
	r.mu.RUnlock()

	for _, enricher := range enrichers {
		if err := enricher.EnrichClaims(ctx, userID, claims); err != nil {
			r.logger.Warn("plugin claim enricher failed",
				"plugin", enricher.Name(),
				"error", err,
			)
			// Continue with other enrichers — don't block token generation
		}
	}
	return nil
}

// GetAuthMethod returns a registered auth method by name.
func (r *Registry) GetAuthMethod(name string) (AuthMethod, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.authMethods[name]
	return m, ok
}

// AuthMethodNames returns the names of all registered auth methods.
func (r *Registry) AuthMethodNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.authMethods))
	for name := range r.authMethods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Middlewares returns all middleware plugins sorted by priority.
func (r *Registry) Middlewares() []MiddlewarePlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]MiddlewarePlugin, len(r.middlewares))
	copy(result, r.middlewares)
	return result
}

// ListPlugins returns info about all registered plugins.
func (r *Registry) ListPlugins() []Info {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]Info, 0, len(r.eventHooks)+len(r.enrichers)+len(r.authMethods)+len(r.middlewares))
	for _, h := range r.eventHooks {
		infos = append(infos, Info{Name: h.Name(), Type: "event_hook", Enabled: true})
	}
	for _, e := range r.enrichers {
		infos = append(infos, Info{Name: e.Name(), Type: "claim_enricher", Enabled: true})
	}
	for _, m := range r.authMethods {
		infos = append(infos, Info{Name: m.Name(), Type: "auth_method", Enabled: true})
	}
	for _, mw := range r.middlewares {
		infos = append(infos, Info{Name: mw.Name(), Type: "middleware", Enabled: true})
	}
	return infos
}

// Close shuts down all plugins that implement io.Closer.
func (r *Registry) Close() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, h := range r.eventHooks {
		if c, ok := h.(io.Closer); ok {
			if err := c.Close(); err != nil {
				r.logger.Warn("plugin close failed", "plugin", h.Name(), "error", err)
			}
		}
	}
	for _, e := range r.enrichers {
		if c, ok := e.(io.Closer); ok {
			if err := c.Close(); err != nil {
				r.logger.Warn("plugin close failed", "plugin", e.Name(), "error", err)
			}
		}
	}
	for _, m := range r.authMethods {
		if c, ok := m.(io.Closer); ok {
			if err := c.Close(); err != nil {
				r.logger.Warn("plugin close failed", "plugin", m.Name(), "error", err)
			}
		}
	}
	for _, mw := range r.middlewares {
		if c, ok := mw.(io.Closer); ok {
			if err := c.Close(); err != nil {
				r.logger.Warn("plugin close failed", "plugin", mw.Name(), "error", err)
			}
		}
	}
	return nil
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
