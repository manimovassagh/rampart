// admin_console_sessions.go contains admin console handlers for session management:
// ListSessionsPage, RevokeSessionAction, RevokeAllSessionsAction.
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/metrics"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
)

// ListSessionsPage handles GET /admin/sessions
func (h *AdminConsoleHandler) ListSessionsPage(w http.ResponseWriter, r *http.Request) {
	authUser := middleware.GetAuthenticatedUser(r.Context())
	search := r.URL.Query().Get("search")
	page := queryInt(r, "page", 1)
	limit := 50
	offset := (page - 1) * limit

	sessions, total, err := h.sessions.ListAll(r.Context(), authUser.OrgID, search, limit, offset)
	if err != nil {
		h.logger.Error("failed to list sessions", "error", err)
		h.render(w, r, "sessions_list", &pageData{Title: "Sessions", ActiveNav: navSessions, Error: "Failed to load sessions."})
		return
	}

	pg := buildPagination(page, limit, total, pathAdminSessions, search)

	if r.Header.Get(headerHXRequest) == formValueTrue {
		h.renderPartial(w, r, "sessions_list", "sessions_table", &pageData{GlobalSessions: sessions, Search: search, Pagination: pg})
		return
	}

	h.render(w, r, "sessions_list", &pageData{
		Title:          "Sessions",
		ActiveNav:      "sessions",
		GlobalSessions: sessions,
		Search:         search,
		Pagination:     pg,
	})
}

// RevokeSessionAction handles POST /admin/sessions/{id}/delete
func (h *AdminConsoleHandler) RevokeSessionAction(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminSessions, http.StatusFound)
		return
	}

	if err := h.sessions.Delete(r.Context(), sessionID); err != nil {
		h.logger.Error("failed to revoke session", "error", err)
		middleware.SetFlash(w, "Failed to revoke session.")
	} else {
		metrics.ActiveSessions.Dec()
		authUser := middleware.GetAuthenticatedUser(r.Context())
		h.auditLog(r, authUser.OrgID, model.EventSessionRevoked, "session", sessionID.String(), "")
		middleware.SetFlash(w, "Session revoked.")
	}

	http.Redirect(w, r, pathAdminSessions, http.StatusFound)
}

// RevokeAllSessionsAction handles POST /admin/sessions/revoke-all
func (h *AdminConsoleHandler) RevokeAllSessionsAction(w http.ResponseWriter, r *http.Request) {
	revokeAllAuthUser := middleware.GetAuthenticatedUser(r.Context())
	if err := h.sessions.DeleteAll(r.Context(), revokeAllAuthUser.OrgID); err != nil {
		h.logger.Error("failed to revoke all sessions", "error", err)
		middleware.SetFlash(w, "Failed to revoke sessions.")
	} else {
		metrics.ActiveSessions.Set(0)
		authUser := middleware.GetAuthenticatedUser(r.Context())
		h.auditLog(r, authUser.OrgID, model.EventSessionsRevokedAll, "session", "all", "")
		middleware.SetFlash(w, "All sessions revoked.")
	}

	http.Redirect(w, r, pathAdminSessions, http.StatusFound)
}
