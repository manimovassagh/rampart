package handler

import (
	"net/http"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
)

// Dashboard handles GET /admin/
func (h *AdminConsoleHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	totalUsers, _ := h.store.CountUsers(ctx, orgID)
	activeSessions, _ := h.sessions.CountActive(ctx)
	recentUsers, _ := h.store.CountRecentUsers(ctx, orgID, 7)
	totalOrgs, _ := h.store.CountOrganizations(ctx)
	totalClients, _ := h.store.CountOAuthClients(ctx, orgID)
	totalRoles, _ := h.store.CountRoles(ctx, orgID)
	totalGroups, _ := h.store.CountGroups(ctx, orgID)
	recentEvents, _ := h.store.CountRecentEvents(ctx, orgID, 24)

	h.render(w, r, "dashboard", &pageData{
		Title:     "Dashboard",
		ActiveNav: "dashboard",
		Stats: &model.DashboardStats{
			TotalUsers:         totalUsers,
			ActiveSessions:     activeSessions,
			RecentUsers:        recentUsers,
			TotalOrganizations: totalOrgs,
			TotalClients:       totalClients,
			TotalRoles:         totalRoles,
			TotalGroups:        totalGroups,
			RecentEvents:       recentEvents,
		},
	})
}
