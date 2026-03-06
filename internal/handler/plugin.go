package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/plugin"
)

// PluginLister provides plugin metadata.
type PluginLister interface {
	Plugins() []plugin.PluginInfo
}

// PluginHandler serves plugin admin endpoints.
type PluginHandler struct {
	registry PluginLister
}

// NewPluginHandler creates a new plugin handler.
func NewPluginHandler(registry PluginLister) *PluginHandler {
	return &PluginHandler{registry: registry}
}

// ListPlugins returns metadata for all loaded plugins.
// GET /api/v1/admin/plugins
func (h *PluginHandler) ListPlugins(w http.ResponseWriter, _ *http.Request) {
	plugins := h.registry.Plugins()
	if plugins == nil {
		plugins = []plugin.PluginInfo{}
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(plugins); err != nil {
		slog.Error("failed to encode plugins response", "error", err)
	}
}
