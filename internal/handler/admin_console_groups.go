// admin_console_groups.go contains admin console handlers for group management:
// ListGroupsPage, CreateGroupPage, CreateGroupAction, GroupDetailPage,
// UpdateGroupAction, DeleteGroupAction, AddGroupMemberAction,
// RemoveGroupMemberAction, AssignGroupRoleAction, UnassignGroupRoleAction.
package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
)

// ListGroupsPage handles GET /admin/groups
func (h *AdminConsoleHandler) ListGroupsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	search := r.URL.Query().Get("search")
	page := queryInt(r, "page", 1)
	limit := 20
	offset := (page - 1) * limit

	groups, total, err := h.store.ListGroups(ctx, orgID, search, limit, offset)
	if err != nil {
		h.logger.Error("failed to list groups", "error", err)
		h.render(w, r, "groups_list", &pageData{Title: "Groups", ActiveNav: navGroups, Error: "Failed to load groups."})
		return
	}

	groupResponses := make([]*model.GroupResponse, len(groups))
	for i, g := range groups {
		memberCount, _ := h.store.CountGroupMembers(ctx, g.ID)
		roleCount, _ := h.store.CountGroupRoles(ctx, g.ID)
		groupResponses[i] = g.ToGroupResponse(memberCount, roleCount)
	}

	pg := buildPagination(page, limit, total, pathAdminGroups, search)

	if r.Header.Get(headerHXRequest) == formValueTrue {
		h.renderPartial(w, r, "groups_list", "groups_table", &pageData{Groups: groupResponses, Search: search, Pagination: pg})
		return
	}

	h.render(w, r, "groups_list", &pageData{
		Title:      "Groups",
		ActiveNav:  "groups",
		Groups:     groupResponses,
		Search:     search,
		Pagination: pg,
	})
}

// CreateGroupPage handles GET /admin/groups/new
func (h *AdminConsoleHandler) CreateGroupPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, tmplGroupCreate, &pageData{Title: titleCreateGroup, ActiveNav: navGroups})
}

// CreateGroupAction handles POST /admin/groups
func (h *AdminConsoleHandler) CreateGroupAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, tmplGroupCreate, &pageData{Title: titleCreateGroup, ActiveNav: navGroups, Error: msgInvalidForm})
		return
	}

	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))

	if name == "" {
		h.render(w, r, tmplGroupCreate, &pageData{Title: titleCreateGroup, ActiveNav: navGroups, Error: "Group name is required."})
		return
	}

	group := &model.Group{
		OrgID:       orgID,
		Name:        name,
		Description: description,
	}

	if _, err := h.store.CreateGroup(ctx, group); err != nil {
		if strings.Contains(err.Error(), msgDuplicateKey) || strings.Contains(err.Error(), "unique") {
			h.render(w, r, tmplGroupCreate, &pageData{Title: titleCreateGroup, ActiveNav: navGroups, Error: "A group with this name already exists."})
			return
		}
		h.logger.Error("failed to create group", "error", err)
		h.render(w, r, tmplGroupCreate, &pageData{Title: titleCreateGroup, ActiveNav: navGroups, Error: "Failed to create group."})
		return
	}

	middleware.SetFlash(w, "Group created successfully.")
	http.Redirect(w, r, pathAdminGroups, http.StatusFound)
}

// GroupDetailPage handles GET /admin/groups/{id}
func (h *AdminConsoleHandler) GroupDetailPage(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminGroups, http.StatusFound)
		return
	}

	ctx := r.Context()
	group, err := h.store.GetGroupByID(ctx, groupID)
	if err != nil || group == nil {
		middleware.SetFlash(w, "Group not found.")
		http.Redirect(w, r, pathAdminGroups, http.StatusFound)
		return
	}

	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	memberCount, _ := h.store.CountGroupMembers(ctx, groupID)
	roleCount, _ := h.store.CountGroupRoles(ctx, groupID)
	members, _ := h.store.GetGroupMembers(ctx, groupID)
	groupRoles, _ := h.store.GetGroupRoles(ctx, groupID)
	allRoles, _, _ := h.store.ListRoles(ctx, orgID, "", 100, 0)

	h.render(w, r, "group_detail", &pageData{
		Title:        fmt.Sprintf("Group: %s", group.Name),
		ActiveNav:    "groups",
		GroupDetail:  group.ToGroupResponse(memberCount, roleCount),
		GroupMembers: members,
		GroupRoles:   groupRoles,
		AllRoles:     allRoles,
	})
}

// UpdateGroupAction handles POST /admin/groups/{id}
func (h *AdminConsoleHandler) UpdateGroupAction(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminGroups, http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
		return
	}

	req := &model.UpdateGroupRequest{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
	}

	if _, err := h.store.UpdateGroup(r.Context(), groupID, req); err != nil {
		h.logger.Error("failed to update group", "error", err)
		middleware.SetFlash(w, "Failed to update group.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Group updated successfully.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
}

// DeleteGroupAction handles POST /admin/groups/{id}/delete
func (h *AdminConsoleHandler) DeleteGroupAction(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminGroups, http.StatusFound)
		return
	}

	if err := h.store.DeleteGroup(r.Context(), groupID); err != nil {
		h.logger.Error("failed to delete group", "error", err)
		middleware.SetFlash(w, "Failed to delete group.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Group deleted.")
	http.Redirect(w, r, pathAdminGroups, http.StatusFound)
}

// AddGroupMemberAction handles POST /admin/groups/{id}/members
func (h *AdminConsoleHandler) AddGroupMemberAction(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminGroups, http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
		return
	}

	userID, err := uuid.Parse(r.FormValue("user_id"))
	if err != nil {
		middleware.SetFlash(w, "Invalid user.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
		return
	}

	if err := h.store.AddUserToGroup(r.Context(), userID, groupID); err != nil {
		h.logger.Error("failed to add member to group", "error", err)
		middleware.SetFlash(w, "Failed to add member.")
	} else {
		middleware.SetFlash(w, "Member added.")
	}

	http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
}

// RemoveGroupMemberAction handles POST /admin/groups/{id}/members/{userId}/delete
func (h *AdminConsoleHandler) RemoveGroupMemberAction(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminGroups, http.StatusFound)
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
		return
	}

	if err := h.store.RemoveUserFromGroup(r.Context(), userID, groupID); err != nil {
		h.logger.Error("failed to remove member from group", "error", err)
		middleware.SetFlash(w, "Failed to remove member.")
	} else {
		middleware.SetFlash(w, "Member removed.")
	}

	http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
}

// AssignGroupRoleAction handles POST /admin/groups/{id}/roles
func (h *AdminConsoleHandler) AssignGroupRoleAction(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminGroups, http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
		return
	}

	roleID, err := uuid.Parse(r.FormValue("role_id"))
	if err != nil {
		middleware.SetFlash(w, msgInvalidRole)
		http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
		return
	}

	if err := h.store.AssignRoleToGroup(r.Context(), groupID, roleID); err != nil {
		h.logger.Error("failed to assign role to group", "error", err)
		middleware.SetFlash(w, "Failed to assign role.")
	} else {
		middleware.SetFlash(w, "Role assigned to group.")
	}

	http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
}

// UnassignGroupRoleAction handles POST /admin/groups/{id}/roles/{roleId}/delete
func (h *AdminConsoleHandler) UnassignGroupRoleAction(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminGroups, http.StatusFound)
		return
	}

	roleID, err := uuid.Parse(chi.URLParam(r, "roleId"))
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
		return
	}

	if err := h.store.UnassignRoleFromGroup(r.Context(), groupID, roleID); err != nil {
		h.logger.Error("failed to unassign role from group", "error", err)
		middleware.SetFlash(w, "Failed to unassign role.")
	} else {
		middleware.SetFlash(w, "Role unassigned from group.")
	}

	http.Redirect(w, r, fmt.Sprintf(pathAdminGroupFmt, groupID), http.StatusFound)
}
