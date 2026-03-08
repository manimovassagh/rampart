// admin_console_plugins.go contains admin console handlers for the plugin system.
package handler

import (
	"net/http"
)

// PluginsPage handles GET /admin/plugins — lists registered plugins.
func (h *AdminConsoleHandler) PluginsPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "plugins_list", &pageData{
		Title:     "Plugins",
		ActiveNav: navPlugins,
		Plugins:   h.plugins.ListPlugins(),
	})
}
