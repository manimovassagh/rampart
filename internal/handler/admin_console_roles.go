// admin_console_roles.go contains admin console handlers for role management:
// ListRolesPage, CreateRolePage, CreateRoleAction, RoleDetailPage,
// UpdateRoleAction, DeleteRoleAction, AssignRoleAction, UnassignRoleAction.
package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/store"
)

// ListRolesPage handles GET /admin/roles
func (h *AdminConsoleHandler) ListRolesPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	search := r.URL.Query().Get("search")
	page := queryInt(r, "page", 1)
	limit := 20
	offset := (page - 1) * limit

	roles, total, err := h.store.ListRoles(ctx, orgID, search, limit, offset)
	if err != nil {
		h.logger.Error("failed to list roles", "error", err)
		h.render(w, r, "roles_list", &pageData{Title: "Roles", ActiveNav: navRoles, Error: "Failed to load roles."})
		return
	}

	roleResponses := make([]*model.RoleResponse, len(roles))
	for i, role := range roles {
		count, _ := h.store.CountRoleUsers(ctx, role.ID)
		roleResponses[i] = role.ToRoleResponse(count)
	}

	pg := buildPagination(page, limit, total, pathAdminRoles, search)

	if r.Header.Get(headerHXRequest) == formValueTrue {
		h.renderPartial(w, r, "roles_list", "roles_table", &pageData{Roles: roleResponses, Search: search, Pagination: pg})
		return
	}

	h.render(w, r, "roles_list", &pageData{
		Title:      "Roles",
		ActiveNav:  "roles",
		Roles:      roleResponses,
		Search:     search,
		Pagination: pg,
	})
}

// CreateRolePage handles GET /admin/roles/new
func (h *AdminConsoleHandler) CreateRolePage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, tmplRoleCreate, &pageData{Title: titleCreateRole, ActiveNav: navRoles})
}

// CreateRoleAction handles POST /admin/roles
func (h *AdminConsoleHandler) CreateRoleAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, tmplRoleCreate, &pageData{Title: titleCreateRole, ActiveNav: navRoles, Error: msgInvalidForm})
		return
	}

	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	name := strings.ToLower(strings.TrimSpace(r.FormValue("name")))
	description := strings.TrimSpace(r.FormValue("description"))

	formValues := map[string]string{"name": name, "description": description}

	if name == "" {
		h.render(w, r, tmplRoleCreate, &pageData{Title: titleCreateRole, ActiveNav: navRoles, FormErrors: map[string]string{"name": "Role name is required."}, FormValues: formValues})
		return
	}

	role := &model.Role{
		OrgID:       orgID,
		Name:        name,
		Description: description,
	}

	if _, err := h.store.CreateRole(ctx, role); err != nil {
		if errors.Is(err, store.ErrDuplicateKey) {
			h.render(w, r, tmplRoleCreate, &pageData{Title: titleCreateRole, ActiveNav: navRoles, FormErrors: map[string]string{"name": "A role with this name already exists."}, FormValues: formValues})
			return
		}
		h.logger.Error("failed to create role", "error", err)
		h.render(w, r, tmplRoleCreate, &pageData{Title: titleCreateRole, ActiveNav: navRoles, Error: "Failed to create role.", FormValues: formValues})
		return
	}

	middleware.SetFlash(w, "Role created successfully.")
	http.Redirect(w, r, pathAdminRoles, http.StatusFound)
}

// RoleDetailPage handles GET /admin/roles/{id}
func (h *AdminConsoleHandler) RoleDetailPage(w http.ResponseWriter, r *http.Request) {
	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminRoles, http.StatusFound)
		return
	}

	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)

	role, err := h.store.GetRoleByID(ctx, roleID)
	if err != nil || role == nil || role.OrgID != authUser.OrgID {
		middleware.SetFlash(w, "Role not found.")
		http.Redirect(w, r, pathAdminRoles, http.StatusFound)
		return
	}

	userCount, _ := h.store.CountRoleUsers(ctx, roleID)
	roleUsers, _ := h.store.GetRoleUsers(ctx, roleID)

	h.render(w, r, "role_detail", &pageData{
		Title:      fmt.Sprintf("Role: %s", role.Name),
		ActiveNav:  "roles",
		RoleDetail: role.ToRoleResponse(userCount),
		RoleUsers:  roleUsers,
	})
}

// UpdateRoleAction handles POST /admin/roles/{id}
func (h *AdminConsoleHandler) UpdateRoleAction(w http.ResponseWriter, r *http.Request) {
	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminRoles, http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminRoleFmt, roleID), http.StatusFound)
		return
	}

	req := &model.UpdateRoleRequest{
		Name:        strings.ToLower(strings.TrimSpace(r.FormValue("name"))),
		Description: strings.TrimSpace(r.FormValue("description")),
	}

	updateRoleAuthUser := middleware.GetAuthenticatedUser(r.Context())
	if _, err := h.store.UpdateRole(r.Context(), roleID, updateRoleAuthUser.OrgID, req); err != nil {
		h.logger.Error("failed to update role", "error", err)
		middleware.SetFlash(w, "Failed to update role.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminRoleFmt, roleID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Role updated successfully.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminRoleFmt, roleID), http.StatusFound)
}

// DeleteRoleAction handles POST /admin/roles/{id}/delete
func (h *AdminConsoleHandler) DeleteRoleAction(w http.ResponseWriter, r *http.Request) {
	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminRoles, http.StatusFound)
		return
	}

	deleteRoleAuthUser := middleware.GetAuthenticatedUser(r.Context())
	if err := h.store.DeleteRole(r.Context(), roleID, deleteRoleAuthUser.OrgID); err != nil {
		switch {
		case errors.Is(err, store.ErrBuiltinRole):
			middleware.SetFlash(w, "Cannot delete built-in roles.")
		case errors.Is(err, store.ErrNotFound):
			middleware.SetFlash(w, "Role not found.")
		default:
			h.logger.Error("failed to delete role", "error", err)
			middleware.SetFlash(w, "Failed to delete role.")
		}
		http.Redirect(w, r, fmt.Sprintf(pathAdminRoleFmt, roleID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Role deleted.")
	http.Redirect(w, r, pathAdminRoles, http.StatusFound)
}

// AssignRoleAction handles POST /admin/users/{id}/roles
func (h *AdminConsoleHandler) AssignRoleAction(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminUsers, http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
		return
	}

	roleID, err := uuid.Parse(r.FormValue("role_id"))
	if err != nil {
		middleware.SetFlash(w, msgInvalidRole)
		http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
		return
	}

	if err := h.store.AssignRole(r.Context(), userID, roleID); err != nil {
		h.logger.Error("failed to assign role", "error", err)
		middleware.SetFlash(w, "Failed to assign role.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
		return
	}

	authUser := middleware.GetAuthenticatedUser(r.Context())
	h.auditLog(r, authUser.OrgID, model.EventRoleAssigned, "user", userID.String(), roleID.String())

	middleware.SetFlash(w, "Role assigned.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
}

// UnassignRoleAction handles POST /admin/users/{id}/roles/{roleId}/delete
func (h *AdminConsoleHandler) UnassignRoleAction(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminUsers, http.StatusFound)
		return
	}

	roleID, err := uuid.Parse(chi.URLParam(r, "roleId"))
	if err != nil {
		middleware.SetFlash(w, msgInvalidRole)
		http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
		return
	}

	if err := h.store.UnassignRole(r.Context(), userID, roleID); err != nil {
		h.logger.Error("failed to unassign role", "error", err)
		middleware.SetFlash(w, "Failed to remove role.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
		return
	}

	unassignAuthUser := middleware.GetAuthenticatedUser(r.Context())
	h.auditLog(r, unassignAuthUser.OrgID, model.EventRoleUnassigned, "user", userID.String(), roleID.String())

	middleware.SetFlash(w, "Role removed.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
}
