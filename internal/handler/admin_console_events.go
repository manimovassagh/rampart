// admin_console_events.go contains admin console handlers for audit event viewing:
// ListEventsPage.
package handler

import (
	"net/http"
	"net/url"

	"github.com/manimovassagh/rampart/internal/middleware"
)

// ListEventsPage handles GET /admin/events
func (h *AdminConsoleHandler) ListEventsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	search := r.URL.Query().Get("search")
	eventFilter := r.URL.Query().Get("event_type")
	page := queryInt(r, "page", 1)
	limit := 50
	offset := (page - 1) * limit

	events, total, err := h.store.ListAuditEvents(ctx, orgID, eventFilter, search, limit, offset)
	if err != nil {
		h.logger.Error("failed to list events", "error", err)
		h.render(w, r, "events_list", &pageData{Title: "Audit Events", ActiveNav: navEvents, Error: "Failed to load events."})
		return
	}

	pg := buildPagination(page, limit, total, pathAdminEvents, search)
	if eventFilter != "" {
		pg.QueryExtra += "&event_type=" + url.QueryEscape(eventFilter)
	}

	if r.Header.Get(headerHXRequest) == formValueTrue {
		h.renderPartial(w, r, "events_list", "events_table", &pageData{Events: events, EventFilter: eventFilter, Search: search, Pagination: pg})
		return
	}

	h.render(w, r, "events_list", &pageData{
		Title:       "Audit Events",
		ActiveNav:   "events",
		Events:      events,
		EventFilter: eventFilter,
		Search:      search,
		Pagination:  pg,
	})
}
