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

	if name == "" {
		h.render(w, r, tmplRoleCreate, &pageData{Title: titleCreateRole, ActiveNav: navRoles, Error: "Role name is required."})
		return
	}

	role := &model.Role{
		OrgID:       orgID,
		Name:        name,
		Description: description,
	}

	if _, err := h.store.CreateRole(ctx, role); err != nil {
		if strings.Contains(err.Error(), msgDuplicateKey) || strings.Contains(err.Error(), "unique") {
			h.render(w, r, tmplRoleCreate, &pageData{Title: titleCreateRole, ActiveNav: navRoles, Error: "A role with this name already exists."})
			return
		}
		h.logger.Error("failed to create role", "error", err)
		h.render(w, r, tmplRoleCreate, &pageData{Title: titleCreateRole, ActiveNav: navRoles, Error: "Failed to create role."})
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
	role, err := h.store.GetRoleByID(ctx, roleID)
	if err != nil || role == nil {
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
		middleware.SetFlash(w, msgInvalidForm)
		http.Redirect(w, r, fmt.Sprintf(pathAdminRoleFmt, roleID), http.StatusFound)
		return
	}

	req := &model.UpdateRoleRequest{
		Name:        strings.ToLower(strings.TrimSpace(r.FormValue("name"))),
		Description: strings.TrimSpace(r.FormValue("description")),
	}

	if _, err := h.store.UpdateRole(r.Context(), roleID, req); err != nil {
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

	if err := h.store.DeleteRole(r.Context(), roleID); err != nil {
		if strings.Contains(err.Error(), "builtin") {
			middleware.SetFlash(w, "Cannot delete built-in roles.")
		} else {
			h.logger.Error("failed to delete role", "error", err)
			middleware.SetFlash(w, "Failed to delete role.")
		}
		http.Redirect(w, r, fmt.Sprintf(pathAdminRoleFmt, roleID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Role deleted.")
	http.Redirect(w, r, pathAdminRoles, http.StatusFound)
}
