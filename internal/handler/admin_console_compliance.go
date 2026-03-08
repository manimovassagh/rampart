// admin_console_compliance.go contains admin console handlers for compliance reporting.
package handler

import (
	"net/http"
)

// CompliancePage handles GET /admin/compliance — shows compliance dashboard.
func (h *AdminConsoleHandler) CompliancePage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "compliance", &pageData{
		Title:     "Compliance",
		ActiveNav: navCompliance,
	})
}
