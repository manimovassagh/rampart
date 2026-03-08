// admin_console_users.go contains admin console handlers for user management:
// ListUsersPage, CreateUserPage, CreateUserAction, UserDetailPage,
// UpdateUserAction, DeleteUserAction, ResetPasswordAction, RevokeSessionsAction.
package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/metrics"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
)

// ListUsersPage handles GET /admin/users
func (h *AdminConsoleHandler) ListUsersPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	page := queryInt(r, "page", 1)
	limit := 20
	offset := (page - 1) * limit

	users, total, err := h.store.ListUsers(ctx, orgID, search, status, limit, offset)
	if err != nil {
		h.logger.Error("failed to list users", "error", err)
		h.render(w, r, "users_list", &pageData{Title: "Users", ActiveNav: navUsers, Error: "Failed to load users."})
		return
	}

	adminUsers := make([]*model.AdminUserResponse, len(users))
	for i, u := range users {
		count, _ := h.sessions.CountByUserID(ctx, u.ID)
		adminUsers[i] = u.ToAdminResponse(count)
	}

	pg := buildPaginationWithExtra(page, limit, total, pathAdminUsers, search, status, "status")

	if r.Header.Get(headerHXRequest) == formValueTrue {
		h.renderPartial(w, r, "users_list", "users_table", &pageData{Users: adminUsers, Search: search, StatusFilter: status, Pagination: pg})
		return
	}

	h.render(w, r, "users_list", &pageData{
		Title:        "Users",
		ActiveNav:    navUsers,
		Users:        adminUsers,
		Search:       search,
		StatusFilter: status,
		Pagination:   pg,
	})
}

// CreateUserPage handles GET /admin/users/new
func (h *AdminConsoleHandler) CreateUserPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers})
}

// CreateUserAction handles POST /admin/users
func (h *AdminConsoleHandler) CreateUserAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, Error: msgInvalidForm})
		return
	}

	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	givenName := strings.TrimSpace(r.FormValue("given_name"))
	familyName := strings.TrimSpace(r.FormValue("family_name"))
	enabled := r.FormValue("enabled") == formValueTrue
	emailVerified := r.FormValue("email_verified") == formValueTrue

	// Preserve form values for re-rendering on error
	formValues := map[string]string{
		"username":    username,
		"email":       email,
		"given_name":  givenName,
		"family_name": familyName,
	}

	// Validate with per-field errors
	formErrors := make(map[string]string)
	if fe := auth.ValidateEmail(email); fe != nil {
		formErrors[fe.Field] = fe.Message
	}
	if fe := auth.ValidatePassword(password); fe != nil {
		formErrors[fe.Field] = fe.Message
	}
	if fe := auth.ValidateUsername(username); fe != nil {
		formErrors[fe.Field] = fe.Message
	}
	if len(formErrors) > 0 {
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, FormErrors: formErrors, FormValues: formValues})
		return
	}

	// Check duplicates
	if existing, err := h.store.GetUserByEmail(ctx, email, orgID); err != nil {
		h.logger.Error("failed to check email uniqueness", "error", err)
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, Error: msgInternalErr, FormValues: formValues})
		return
	} else if existing != nil {
		formErrors["email"] = "A user with this email already exists."
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, FormErrors: formErrors, FormValues: formValues})
		return
	}
	if existing, err := h.store.GetUserByUsername(ctx, username, orgID); err != nil {
		h.logger.Error("failed to check username uniqueness", "error", err)
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, Error: msgInternalErr, FormValues: formValues})
		return
	} else if existing != nil {
		formErrors["username"] = "A user with this username already exists."
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, FormErrors: formErrors, FormValues: formValues})
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, Error: msgInternalErr, FormValues: formValues})
		return
	}

	user := &model.User{
		OrgID:         orgID,
		Username:      username,
		Email:         email,
		GivenName:     givenName,
		FamilyName:    familyName,
		PasswordHash:  []byte(hash),
		Enabled:       enabled,
		EmailVerified: emailVerified,
	}

	created, err := h.store.CreateUser(ctx, user)
	if err != nil {
		h.logger.Error("failed to create user", "error", err)
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, Error: "Failed to create user.", FormValues: formValues})
		return
	}

	h.auditLog(r, orgID, model.EventUserCreated, "user", created.ID.String(), username)
	middleware.SetFlash(w, "User created successfully.")
	http.Redirect(w, r, pathAdminUsers, http.StatusFound)
}

// UserDetailPage handles GET /admin/users/{id}
func (h *AdminConsoleHandler) UserDetailPage(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminUsers, http.StatusFound)
		return
	}

	ctx := r.Context()

	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		middleware.SetFlash(w, "User not found.")
		http.Redirect(w, r, pathAdminUsers, http.StatusFound)
		return
	}

	sessionCount, _ := h.sessions.CountByUserID(ctx, userID)
	sessions, _ := h.sessions.ListByUserID(ctx, userID)
	userRoles, _ := h.store.GetUserRoles(ctx, userID)
	userGroups, _ := h.store.GetUserGroups(ctx, userID)

	authUser := middleware.GetAuthenticatedUser(ctx)
	allRoles, _, _ := h.store.ListRoles(ctx, authUser.OrgID, "", 100, 0)

	h.render(w, r, "user_detail", &pageData{
		Title:      fmt.Sprintf("User: %s", user.Username),
		ActiveNav:  "users",
		UserDetail: user.ToAdminResponse(sessionCount),
		Sessions:   sessions,
		UserRoles:  userRoles,
		AllRoles:   allRoles,
		UserGroups: userGroups,
	})
}

// UpdateUserAction handles POST /admin/users/{id}
func (h *AdminConsoleHandler) UpdateUserAction(w http.ResponseWriter, r *http.Request) {
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

	req := &model.UpdateUserRequest{
		Username:      strings.TrimSpace(r.FormValue("username")),
		Email:         strings.ToLower(strings.TrimSpace(r.FormValue("email"))),
		GivenName:     strings.TrimSpace(r.FormValue("given_name")),
		FamilyName:    strings.TrimSpace(r.FormValue("family_name")),
		Enabled:       r.FormValue("enabled") == formValueTrue,
		EmailVerified: r.FormValue("email_verified") == formValueTrue,
	}

	if _, err := h.store.UpdateUser(r.Context(), userID, req); err != nil {
		h.logger.Error("failed to update user", "error", err)
		middleware.SetFlash(w, "Failed to update user.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
		return
	}

	authUser := middleware.GetAuthenticatedUser(r.Context())
	h.auditLog(r, authUser.OrgID, model.EventUserUpdated, "user", userID.String(), req.Username)
	middleware.SetFlash(w, "User updated successfully.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
}

// DeleteUserAction handles POST /admin/users/{id}/delete
func (h *AdminConsoleHandler) DeleteUserAction(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminUsers, http.StatusFound)
		return
	}

	authUser := middleware.GetAuthenticatedUser(r.Context())
	if authUser != nil && authUser.UserID == userID {
		middleware.SetFlash(w, "You cannot delete your own account.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
		return
	}

	ctx := r.Context()

	sessionCount, _ := h.sessions.CountByUserID(ctx, userID)
	if err := h.sessions.DeleteByUserID(ctx, userID); err != nil {
		h.logger.Error("failed to delete user sessions", "error", err)
	} else {
		metrics.ActiveSessions.Sub(float64(sessionCount))
	}
	if err := h.store.DeleteUser(ctx, userID); err != nil {
		h.logger.Error("failed to delete user", "error", err)
		middleware.SetFlash(w, "Failed to delete user.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
		return
	}

	h.auditLog(r, authUser.OrgID, model.EventUserDeleted, "user", userID.String(), "")
	middleware.SetFlash(w, "User deleted.")
	http.Redirect(w, r, pathAdminUsers, http.StatusFound)
}

// ResetPasswordAction handles POST /admin/users/{id}/reset-password
func (h *AdminConsoleHandler) ResetPasswordAction(w http.ResponseWriter, r *http.Request) {
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

	password := r.FormValue("password")
	if fe := auth.ValidatePassword(password); fe != nil {
		middleware.SetFlash(w, fe.Message)
		http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		middleware.SetFlash(w, "Internal error.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
		return
	}

	if err := h.store.UpdatePassword(r.Context(), userID, []byte(hash)); err != nil {
		h.logger.Error("failed to update password", "error", err)
		middleware.SetFlash(w, "Failed to reset password.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
		return
	}

	pwAuthUser := middleware.GetAuthenticatedUser(r.Context())
	h.auditLog(r, pwAuthUser.OrgID, model.EventUserPasswordReset, "user", userID.String(), "")
	middleware.SetFlash(w, "Password reset successfully.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
}

// RevokeSessionsAction handles POST /admin/users/{id}/revoke-sessions
func (h *AdminConsoleHandler) RevokeSessionsAction(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminUsers, http.StatusFound)
		return
	}

	revokeCtx := r.Context()
	revokeCount, _ := h.sessions.CountByUserID(revokeCtx, userID)
	if err := h.sessions.DeleteByUserID(revokeCtx, userID); err != nil {
		h.logger.Error("failed to revoke sessions", "error", err)
		middleware.SetFlash(w, "Failed to revoke sessions.")
	} else {
		metrics.ActiveSessions.Sub(float64(revokeCount))
		sessAuthUser := middleware.GetAuthenticatedUser(revokeCtx)
		h.auditLog(r, sessAuthUser.OrgID, model.EventSessionRevoked, "user", userID.String(), "")
		middleware.SetFlash(w, "All sessions revoked.")
	}

	http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
}
