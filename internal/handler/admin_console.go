package handler

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/manimovassagh/rampart/internal/audit"
	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
)

//go:embed templates/admin/*.html templates/admin/partials/*.html
var adminTemplateFS embed.FS

// AdminConsoleStore defines the database operations required by AdminConsoleHandler.
type AdminConsoleStore interface {
	GetUserByEmail(ctx context.Context, email string, orgID uuid.UUID) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string, orgID uuid.UUID) (*model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	ListUsers(ctx context.Context, orgID uuid.UUID, search, status string, limit, offset int) ([]*model.User, int, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req *model.UpdateUserRequest) (*model.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash []byte) error
	CountUsers(ctx context.Context, orgID uuid.UUID) (int, error)
	CountRecentUsers(ctx context.Context, orgID uuid.UUID, days int) (int, error)
	CountOrganizations(ctx context.Context) (int, error)
	GetOrgSettings(ctx context.Context, orgID uuid.UUID) (*model.OrgSettings, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*model.Organization, error)
	ListOrganizations(ctx context.Context, search string, limit, offset int) ([]*model.Organization, int, error)
	CreateOrganization(ctx context.Context, req *model.CreateOrgRequest) (*model.Organization, error)
	UpdateOrganization(ctx context.Context, id uuid.UUID, req *model.UpdateOrgRequest) (*model.Organization, error)
	DeleteOrganization(ctx context.Context, id uuid.UUID) error
	UpdateOrgSettings(ctx context.Context, orgID uuid.UUID, req *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error)

	// OAuth Client operations
	GetOAuthClient(ctx context.Context, clientID string) (*model.OAuthClient, error)
	ListOAuthClients(ctx context.Context, orgID uuid.UUID, search string, limit, offset int) ([]*model.OAuthClient, int, error)
	CreateOAuthClient(ctx context.Context, client *model.OAuthClient) (*model.OAuthClient, error)
	UpdateOAuthClient(ctx context.Context, clientID string, req *model.UpdateClientRequest) (*model.OAuthClient, error)
	DeleteOAuthClient(ctx context.Context, clientID string) error
	UpdateClientSecret(ctx context.Context, clientID string, secretHash []byte) error
	CountOAuthClients(ctx context.Context, orgID uuid.UUID) (int, error)

	// Role operations
	ListRoles(ctx context.Context, orgID uuid.UUID, search string, limit, offset int) ([]*model.Role, int, error)
	GetRoleByID(ctx context.Context, id uuid.UUID) (*model.Role, error)
	CreateRole(ctx context.Context, role *model.Role) (*model.Role, error)
	UpdateRole(ctx context.Context, id uuid.UUID, req *model.UpdateRoleRequest) (*model.Role, error)
	DeleteRole(ctx context.Context, id uuid.UUID) error
	CountRoles(ctx context.Context, orgID uuid.UUID) (int, error)
	CountRoleUsers(ctx context.Context, roleID uuid.UUID) (int, error)
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
	UnassignRole(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*model.Role, error)
	GetRoleUsers(ctx context.Context, roleID uuid.UUID) ([]*model.UserRoleAssignment, error)

	// Audit event operations
	CreateAuditEvent(ctx context.Context, event *model.AuditEvent) error
	ListAuditEvents(ctx context.Context, orgID uuid.UUID, eventType, search string, limit, offset int) ([]*model.AuditEvent, int, error)
	CountRecentEvents(ctx context.Context, orgID uuid.UUID, hours int) (int, error)

	// Group operations
	ListGroups(ctx context.Context, orgID uuid.UUID, search string, limit, offset int) ([]*model.Group, int, error)
	GetGroupByID(ctx context.Context, id uuid.UUID) (*model.Group, error)
	CreateGroup(ctx context.Context, group *model.Group) (*model.Group, error)
	UpdateGroup(ctx context.Context, id uuid.UUID, req *model.UpdateGroupRequest) (*model.Group, error)
	DeleteGroup(ctx context.Context, id uuid.UUID) error
	CountGroups(ctx context.Context, orgID uuid.UUID) (int, error)
	CountGroupMembers(ctx context.Context, groupID uuid.UUID) (int, error)
	CountGroupRoles(ctx context.Context, groupID uuid.UUID) (int, error)
	AddUserToGroup(ctx context.Context, userID, groupID uuid.UUID) error
	RemoveUserFromGroup(ctx context.Context, userID, groupID uuid.UUID) error
	GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]*model.GroupMember, error)
	GetGroupRoles(ctx context.Context, groupID uuid.UUID) ([]*model.GroupRoleAssignment, error)
	AssignRoleToGroup(ctx context.Context, groupID, roleID uuid.UUID) error
	UnassignRoleFromGroup(ctx context.Context, groupID, roleID uuid.UUID) error
	GetUserGroups(ctx context.Context, userID uuid.UUID) ([]*model.Group, error)
}

// AdminConsoleSessionStore defines session operations for the admin console.
type AdminConsoleSessionStore interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*session.Session, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	CountActive(ctx context.Context) (int, error)
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	Delete(ctx context.Context, sessionID uuid.UUID) error
	ListAll(ctx context.Context, search string, limit, offset int) ([]*session.SessionWithUser, int, error)
	DeleteAll(ctx context.Context) error
}

// AdminConsoleHandler serves SSR admin pages.
type AdminConsoleHandler struct {
	store    AdminConsoleStore
	sessions AdminConsoleSessionStore
	audit    *audit.Logger
	logger   *slog.Logger
	issuer   string
	pages    map[string]*template.Template
}

var adminFuncMap = template.FuncMap{
	"add":      func(a, b int) int { return a + b },
	"subtract": func(a, b int) int { return a - b },
	"slice": func(s string, start, end int) string {
		if end > len(s) {
			end = len(s)
		}
		if start > len(s) {
			start = len(s)
		}
		return s[start:end]
	},
}

// parseAdminPage builds a template set for a single page: base + partials + page.
func parseAdminPage(pageFile string) *template.Template {
	return template.Must(
		template.New("").Funcs(adminFuncMap).ParseFS(adminTemplateFS,
			"templates/admin/base.html",
			"templates/admin/partials/*.html",
			"templates/admin/"+pageFile,
		),
	)
}

// NewAdminConsoleHandler creates a handler for SSR admin pages.
func NewAdminConsoleHandler(store AdminConsoleStore, sessions AdminConsoleSessionStore, logger *slog.Logger, issuer string, auditLogger *audit.Logger) *AdminConsoleHandler {
	pages := map[string]*template.Template{
		"dashboard":     parseAdminPage("dashboard.html"),
		"users_list":    parseAdminPage("users_list.html"),
		"user_create":   parseAdminPage("user_create.html"),
		"user_detail":   parseAdminPage("user_detail.html"),
		"orgs_list":     parseAdminPage("orgs_list.html"),
		"org_create":    parseAdminPage("org_create.html"),
		"org_detail":    parseAdminPage("org_detail.html"),
		"clients_list":  parseAdminPage("clients_list.html"),
		"client_create": parseAdminPage("client_create.html"),
		"client_detail": parseAdminPage("client_detail.html"),
		"roles_list":    parseAdminPage("roles_list.html"),
		"role_create":   parseAdminPage("role_create.html"),
		"role_detail":   parseAdminPage("role_detail.html"),
		"events_list":    parseAdminPage("events_list.html"),
		"sessions_list":  parseAdminPage("sessions_list.html"),
		"groups_list":    parseAdminPage("groups_list.html"),
		"group_create":   parseAdminPage("group_create.html"),
		"group_detail":   parseAdminPage("group_detail.html"),
		"org_import":     parseAdminPage("org_import.html"),
		"oidc":           parseAdminPage("oidc.html"),
	}

	return &AdminConsoleHandler{
		store:    store,
		sessions: sessions,
		audit:    auditLogger,
		logger:   logger,
		issuer:   issuer,
		pages:    pages,
	}
}

// pageData holds common data passed to all admin templates.
type pageData struct {
	Title     string
	ActiveNav string
	User      *middleware.AuthenticatedUser
	CSRFToken string
	Flash     string
	Error     string

	// Page-specific data
	Stats        *model.DashboardStats
	Users        []*model.AdminUserResponse
	UserDetail   *model.AdminUserResponse
	Sessions     []*session.Session
	Orgs         []*model.OrgResponse
	OrgDetail    *model.Organization
	OrgSettings  *model.OrgSettingsResponse
	Clients      []*model.AdminClientResponse
	ClientDetail *model.AdminClientResponse
	ClientSecret string
	Roles        []*model.RoleResponse
	RoleDetail   *model.RoleResponse
	RoleUsers    []*model.UserRoleAssignment
	UserRoles    []*model.Role
	AllRoles     []*model.Role
	Events         []*model.AuditEvent
	EventFilter    string
	GlobalSessions []*session.SessionWithUser
	Groups         []*model.GroupResponse
	GroupDetail    *model.GroupResponse
	GroupMembers   []*model.GroupMember
	GroupRoles     []*model.GroupRoleAssignment
	UserGroups     []*model.Group
	OIDC           *DiscoveryResponse
	Search       string
	Pagination   *paginationData
}

type paginationData struct {
	Page       int
	TotalPages int
	Total      int
	From       int
	To         int
	Pages      []int
	BaseURL    string
	QueryExtra string
}

func (h *AdminConsoleHandler) render(w http.ResponseWriter, r *http.Request, pageName string, data *pageData) {
	data.User = middleware.GetAuthenticatedUser(r.Context())
	data.CSRFToken = middleware.GetCSRFToken(r)

	flash := middleware.GetFlash(w, r)
	if flash != "" {
		data.Flash = flash
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")

	tmpl, ok := h.pages[pageName]
	if !ok {
		h.logger.Error("unknown page template", "page", pageName)
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error("failed to render template", "page", pageName, "error", err)
	}
}

// renderPartial renders a named template block for HTMX partial responses.
func (h *AdminConsoleHandler) renderPartial(w http.ResponseWriter, r *http.Request, pageName, blockName string, data *pageData) {
	data.User = middleware.GetAuthenticatedUser(r.Context())
	data.CSRFToken = middleware.GetCSRFToken(r)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	tmpl, ok := h.pages[pageName]
	if !ok {
		h.logger.Error("unknown page template", "page", pageName)
		return
	}

	if err := tmpl.ExecuteTemplate(w, blockName, data); err != nil {
		h.logger.Error("failed to render partial", "page", pageName, "block", blockName, "error", err)
	}
}

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

// ListUsersPage handles GET /admin/users
func (h *AdminConsoleHandler) ListUsersPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	search := r.URL.Query().Get("search")
	page := queryInt(r, "page", 1)
	limit := 20
	offset := (page - 1) * limit

	users, total, err := h.store.ListUsers(ctx, orgID, search, "", limit, offset)
	if err != nil {
		h.logger.Error("failed to list users", "error", err)
		h.render(w, r, "users_list", &pageData{Title: "Users", ActiveNav: "users", Error: "Failed to load users."})
		return
	}

	adminUsers := make([]*model.AdminUserResponse, len(users))
	for i, u := range users {
		count, _ := h.sessions.CountByUserID(ctx, u.ID)
		adminUsers[i] = u.ToAdminResponse(count)
	}

	pg := buildPagination(page, limit, total, "/admin/users", search)

	if r.Header.Get("HX-Request") == "true" {
		h.renderPartial(w, r, "users_list", "users_table", &pageData{Users: adminUsers, Search: search, Pagination: pg})
		return
	}

	h.render(w, r, "users_list", &pageData{
		Title:      "Users",
		ActiveNav:  "users",
		Users:      adminUsers,
		Search:     search,
		Pagination: pg,
	})
}

// CreateUserPage handles GET /admin/users/new
func (h *AdminConsoleHandler) CreateUserPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "user_create", &pageData{Title: "Create User", ActiveNav: "users"})
}

// CreateUserAction handles POST /admin/users
func (h *AdminConsoleHandler) CreateUserAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, "user_create", &pageData{Title: "Create User", ActiveNav: "users", Error: "Invalid form data."})
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
	enabled := r.FormValue("enabled") == "true"
	emailVerified := r.FormValue("email_verified") == "true"

	// Validate
	var errors []string
	if fe := auth.ValidateEmail(email); fe != nil {
		errors = append(errors, fe.Message)
	}
	if fe := auth.ValidatePassword(password); fe != nil {
		errors = append(errors, fe.Message)
	}
	if fe := auth.ValidateUsername(username); fe != nil {
		errors = append(errors, fe.Message)
	}
	if len(errors) > 0 {
		h.render(w, r, "user_create", &pageData{Title: "Create User", ActiveNav: "users", Error: strings.Join(errors, " ")})
		return
	}

	// Check duplicates
	if existing, _ := h.store.GetUserByEmail(ctx, email, orgID); existing != nil {
		h.render(w, r, "user_create", &pageData{Title: "Create User", ActiveNav: "users", Error: "A user with this email already exists."})
		return
	}
	if existing, _ := h.store.GetUserByUsername(ctx, username, orgID); existing != nil {
		h.render(w, r, "user_create", &pageData{Title: "Create User", ActiveNav: "users", Error: "A user with this username already exists."})
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		h.render(w, r, "user_create", &pageData{Title: "Create User", ActiveNav: "users", Error: "Internal error."})
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
		h.render(w, r, "user_create", &pageData{Title: "Create User", ActiveNav: "users", Error: "Failed to create user."})
		return
	}

	h.auditLog(r, orgID, model.EventUserCreated, "user", created.ID.String(), username)
	middleware.SetFlash(w, "User created successfully.")
	http.Redirect(w, r, "/admin/users", http.StatusFound)
}

// UserDetailPage handles GET /admin/users/{id}
func (h *AdminConsoleHandler) UserDetailPage(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusFound)
		return
	}

	ctx := r.Context()

	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		middleware.SetFlash(w, "User not found.")
		http.Redirect(w, r, "/admin/users", http.StatusFound)
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
		http.Redirect(w, r, "/admin/users", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	req := &model.UpdateUserRequest{
		Username:      strings.TrimSpace(r.FormValue("username")),
		Email:         strings.ToLower(strings.TrimSpace(r.FormValue("email"))),
		GivenName:     strings.TrimSpace(r.FormValue("given_name")),
		FamilyName:    strings.TrimSpace(r.FormValue("family_name")),
		Enabled:       r.FormValue("enabled") == "true",
		EmailVerified: r.FormValue("email_verified") == "true",
	}

	if _, err := h.store.UpdateUser(r.Context(), userID, req); err != nil {
		h.logger.Error("failed to update user", "error", err)
		middleware.SetFlash(w, "Failed to update user.")
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	authUser := middleware.GetAuthenticatedUser(r.Context())
	h.auditLog(r, authUser.OrgID, model.EventUserUpdated, "user", userID.String(), req.Username)
	middleware.SetFlash(w, "User updated successfully.")
	http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
}

// DeleteUserAction handles POST /admin/users/{id}/delete
func (h *AdminConsoleHandler) DeleteUserAction(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusFound)
		return
	}

	authUser := middleware.GetAuthenticatedUser(r.Context())
	if authUser != nil && authUser.UserID == userID {
		middleware.SetFlash(w, "You cannot delete your own account.")
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	ctx := r.Context()

	if err := h.sessions.DeleteByUserID(ctx, userID); err != nil {
		h.logger.Error("failed to delete user sessions", "error", err)
	}
	if err := h.store.DeleteUser(ctx, userID); err != nil {
		h.logger.Error("failed to delete user", "error", err)
		middleware.SetFlash(w, "Failed to delete user.")
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	h.auditLog(r, authUser.OrgID, model.EventUserDeleted, "user", userID.String(), "")
	middleware.SetFlash(w, "User deleted.")
	http.Redirect(w, r, "/admin/users", http.StatusFound)
}

// ResetPasswordAction handles POST /admin/users/{id}/reset-password
func (h *AdminConsoleHandler) ResetPasswordAction(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	password := r.FormValue("password")
	if fe := auth.ValidatePassword(password); fe != nil {
		middleware.SetFlash(w, fe.Message)
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		middleware.SetFlash(w, "Internal error.")
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	if err := h.store.UpdatePassword(r.Context(), userID, []byte(hash)); err != nil {
		h.logger.Error("failed to update password", "error", err)
		middleware.SetFlash(w, "Failed to reset password.")
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	pwAuthUser := middleware.GetAuthenticatedUser(r.Context())
	h.auditLog(r, pwAuthUser.OrgID, model.EventUserPasswordReset, "user", userID.String(), "")
	middleware.SetFlash(w, "Password reset successfully.")
	http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
}

// RevokeSessionsAction handles POST /admin/users/{id}/revoke-sessions
func (h *AdminConsoleHandler) RevokeSessionsAction(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusFound)
		return
	}

	if err := h.sessions.DeleteByUserID(r.Context(), userID); err != nil {
		h.logger.Error("failed to revoke sessions", "error", err)
		middleware.SetFlash(w, "Failed to revoke sessions.")
	} else {
		sessAuthUser := middleware.GetAuthenticatedUser(r.Context())
		h.auditLog(r, sessAuthUser.OrgID, model.EventSessionRevoked, "user", userID.String(), "")
		middleware.SetFlash(w, "All sessions revoked.")
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
}

// ListOrgsPage handles GET /admin/organizations
func (h *AdminConsoleHandler) ListOrgsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	search := r.URL.Query().Get("search")
	page := queryInt(r, "page", 1)
	limit := 20
	offset := (page - 1) * limit

	orgs, total, err := h.store.ListOrganizations(ctx, search, limit, offset)
	if err != nil {
		h.logger.Error("failed to list organizations", "error", err)
		h.render(w, r, "orgs_list", &pageData{Title: "Organizations", ActiveNav: "organizations", Error: "Failed to load organizations."})
		return
	}

	orgResponses := make([]*model.OrgResponse, len(orgs))
	for i, o := range orgs {
		count, _ := h.store.CountUsers(ctx, o.ID)
		orgResponses[i] = o.ToOrgResponse(count)
	}

	pg := buildPagination(page, limit, total, "/admin/organizations", search)

	if r.Header.Get("HX-Request") == "true" {
		h.renderPartial(w, r, "orgs_list", "orgs_table", &pageData{Orgs: orgResponses, Search: search, Pagination: pg})
		return
	}

	h.render(w, r, "orgs_list", &pageData{
		Title:      "Organizations",
		ActiveNav:  "organizations",
		Orgs:       orgResponses,
		Search:     search,
		Pagination: pg,
	})
}

// CreateOrgPage handles GET /admin/organizations/new
func (h *AdminConsoleHandler) CreateOrgPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "org_create", &pageData{Title: "Create Organization", ActiveNav: "organizations"})
}

// CreateOrgAction handles POST /admin/organizations
func (h *AdminConsoleHandler) CreateOrgAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, "org_create", &pageData{Title: "Create Organization", ActiveNav: "organizations", Error: "Invalid form data."})
		return
	}

	req := &model.CreateOrgRequest{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Slug:        strings.ToLower(strings.TrimSpace(r.FormValue("slug"))),
		DisplayName: strings.TrimSpace(r.FormValue("display_name")),
	}

	if req.Name == "" || req.Slug == "" {
		h.render(w, r, "org_create", &pageData{Title: "Create Organization", ActiveNav: "organizations", Error: "Name and slug are required."})
		return
	}

	newOrg, err := h.store.CreateOrganization(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			h.render(w, r, "org_create", &pageData{Title: "Create Organization", ActiveNav: "organizations", Error: "An organization with this slug already exists."})
			return
		}
		h.logger.Error("failed to create organization", "error", err)
		h.render(w, r, "org_create", &pageData{Title: "Create Organization", ActiveNav: "organizations", Error: "Failed to create organization."})
		return
	}

	orgAuthUser := middleware.GetAuthenticatedUser(r.Context())
	h.auditLog(r, orgAuthUser.OrgID, model.EventOrgCreated, "organization", newOrg.ID.String(), req.Name)
	middleware.SetFlash(w, "Organization created successfully.")
	http.Redirect(w, r, "/admin/organizations", http.StatusFound)
}

// OrgDetailPage handles GET /admin/organizations/{id}
func (h *AdminConsoleHandler) OrgDetailPage(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/organizations", http.StatusFound)
		return
	}

	ctx := r.Context()

	org, err := h.store.GetOrganizationByID(ctx, orgID)
	if err != nil || org == nil {
		middleware.SetFlash(w, "Organization not found.")
		http.Redirect(w, r, "/admin/organizations", http.StatusFound)
		return
	}

	var settingsResp *model.OrgSettingsResponse
	if settings, sErr := h.store.GetOrgSettings(ctx, orgID); sErr == nil && settings != nil {
		settingsResp = settings.ToResponse()
	}

	h.render(w, r, "org_detail", &pageData{
		Title:       fmt.Sprintf("Organization: %s", org.Name),
		ActiveNav:   "organizations",
		OrgDetail:   org,
		OrgSettings: settingsResp,
	})
}

// UpdateOrgAction handles POST /admin/organizations/{id}
func (h *AdminConsoleHandler) UpdateOrgAction(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/organizations", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf("/admin/organizations/%s", orgID), http.StatusFound)
		return
	}

	req := &model.UpdateOrgRequest{
		Name:        strings.TrimSpace(r.FormValue("name")),
		DisplayName: strings.TrimSpace(r.FormValue("display_name")),
		Enabled:     r.FormValue("enabled") == "true",
	}

	if _, err := h.store.UpdateOrganization(r.Context(), orgID, req); err != nil {
		h.logger.Error("failed to update organization", "error", err)
		middleware.SetFlash(w, "Failed to update organization.")
		http.Redirect(w, r, fmt.Sprintf("/admin/organizations/%s", orgID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Organization updated successfully.")
	http.Redirect(w, r, fmt.Sprintf("/admin/organizations/%s", orgID), http.StatusFound)
}

// UpdateOrgSettingsAction handles POST /admin/organizations/{id}/settings
func (h *AdminConsoleHandler) UpdateOrgSettingsAction(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/organizations", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf("/admin/organizations/%s", orgID), http.StatusFound)
		return
	}

	accessTTL, _ := strconv.Atoi(r.FormValue("access_token_ttl_seconds"))
	refreshTTL, _ := strconv.Atoi(r.FormValue("refresh_token_ttl_seconds"))
	minLen, _ := strconv.Atoi(r.FormValue("password_min_length"))

	if minLen < 1 {
		minLen = 1
	}
	if accessTTL < 60 {
		accessTTL = 60
	}
	if refreshTTL < 60 {
		refreshTTL = 60
	}

	mfa := r.FormValue("mfa_enforcement")
	if mfa != "off" && mfa != "optional" && mfa != "required" {
		mfa = "off"
	}

	req := &model.UpdateOrgSettingsRequest{
		PasswordMinLength:          minLen,
		PasswordRequireUppercase:   r.FormValue("password_require_uppercase") == "true",
		PasswordRequireLowercase:   r.FormValue("password_require_lowercase") == "true",
		PasswordRequireNumbers:     r.FormValue("password_require_numbers") == "true",
		PasswordRequireSymbols:     r.FormValue("password_require_symbols") == "true",
		MFAEnforcement:             mfa,
		AccessTokenTTLSeconds:      accessTTL,
		RefreshTokenTTLSeconds:     refreshTTL,
		SelfRegistrationEnabled:    r.FormValue("self_registration_enabled") == "true",
		EmailVerificationRequired:  r.FormValue("email_verification_required") == "true",
		ForgotPasswordEnabled:      r.FormValue("forgot_password_enabled") == "true",
		RememberMeEnabled:          r.FormValue("remember_me_enabled") == "true",
		LoginPageTitle:             strings.TrimSpace(r.FormValue("login_page_title")),
		LoginPageMessage:           strings.TrimSpace(r.FormValue("login_page_message")),
	}

	if _, err := h.store.UpdateOrgSettings(r.Context(), orgID, req); err != nil {
		h.logger.Error("failed to update org settings", "error", err)
		middleware.SetFlash(w, "Failed to update settings.")
		http.Redirect(w, r, fmt.Sprintf("/admin/organizations/%s", orgID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Settings updated successfully.")
	http.Redirect(w, r, fmt.Sprintf("/admin/organizations/%s", orgID), http.StatusFound)
}

// DeleteOrgAction handles POST /admin/organizations/{id}/delete
func (h *AdminConsoleHandler) DeleteOrgAction(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/organizations", http.StatusFound)
		return
	}

	if err := h.store.DeleteOrganization(r.Context(), orgID); err != nil {
		if strings.Contains(err.Error(), "default") {
			middleware.SetFlash(w, "Cannot delete the default organization.")
		} else {
			h.logger.Error("failed to delete organization", "error", err)
			middleware.SetFlash(w, "Failed to delete organization.")
		}
		http.Redirect(w, r, fmt.Sprintf("/admin/organizations/%s", orgID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Organization deleted.")
	http.Redirect(w, r, "/admin/organizations", http.StatusFound)
}

// OIDCPage handles GET /admin/oidc
func (h *AdminConsoleHandler) OIDCPage(w http.ResponseWriter, r *http.Request) {
	oidc := &DiscoveryResponse{
		Issuer:                           h.issuer,
		AuthorizationEndpoint:            h.issuer + "/oauth/authorize",
		TokenEndpoint:                    h.issuer + "/oauth/token",
		UserinfoEndpoint:                 h.issuer + "/me",
		JWKSURI:                          h.issuer + "/.well-known/jwks.json",
		ResponseTypesSupported:           []string{"code"},
		GrantTypesSupported:              []string{"authorization_code", "refresh_token"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
		ScopesSupported:                  []string{"openid", "profile", "email"},
		ClaimsSupported: []string{
			"sub", "iss", "iat", "exp",
			"preferred_username", "email", "email_verified",
			"given_name", "family_name", "org_id", "roles",
		},
		CodeChallengeMethodsSupported: []string{"S256"},
	}

	h.render(w, r, "oidc", &pageData{
		Title:     "OIDC Configuration",
		ActiveNav: "oidc",
		OIDC:      oidc,
	})
}

// ListClientsPage handles GET /admin/clients
func (h *AdminConsoleHandler) ListClientsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	search := r.URL.Query().Get("search")
	page := queryInt(r, "page", 1)
	limit := 20
	offset := (page - 1) * limit

	clients, total, err := h.store.ListOAuthClients(ctx, orgID, search, limit, offset)
	if err != nil {
		h.logger.Error("failed to list clients", "error", err)
		h.render(w, r, "clients_list", &pageData{Title: "OAuth Clients", ActiveNav: "clients", Error: "Failed to load clients."})
		return
	}

	adminClients := make([]*model.AdminClientResponse, len(clients))
	for i, c := range clients {
		adminClients[i] = c.ToAdminResponse()
	}

	pg := buildPagination(page, limit, total, "/admin/clients", search)

	if r.Header.Get("HX-Request") == "true" {
		h.renderPartial(w, r, "clients_list", "clients_table", &pageData{Clients: adminClients, Search: search, Pagination: pg})
		return
	}

	h.render(w, r, "clients_list", &pageData{
		Title:      "OAuth Clients",
		ActiveNav:  "clients",
		Clients:    adminClients,
		Search:     search,
		Pagination: pg,
	})
}

// CreateClientPage handles GET /admin/clients/new
func (h *AdminConsoleHandler) CreateClientPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "client_create", &pageData{Title: "Create Client", ActiveNav: "clients"})
}

// CreateClientAction handles POST /admin/clients
func (h *AdminConsoleHandler) CreateClientAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, "client_create", &pageData{Title: "Create Client", ActiveNav: "clients", Error: "Invalid form data."})
		return
	}

	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	clientType := r.FormValue("client_type")
	redirectURIsRaw := strings.TrimSpace(r.FormValue("redirect_uris"))

	if name == "" {
		h.render(w, r, "client_create", &pageData{Title: "Create Client", ActiveNav: "clients", Error: "Client name is required."})
		return
	}

	if clientType != "public" && clientType != "confidential" {
		clientType = "public"
	}

	var uris []string
	for _, line := range strings.Split(redirectURIsRaw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			uris = append(uris, trimmed)
		}
	}

	client := &model.OAuthClient{
		OrgID:        orgID,
		Name:         name,
		Description:  description,
		ClientType:   clientType,
		RedirectURIs: uris,
		Enabled:      true,
	}

	var clientSecret string
	if clientType == "confidential" {
		secret, err := generateRandomSecret()
		if err != nil {
			h.logger.Error("failed to generate client secret", "error", err)
			h.render(w, r, "client_create", &pageData{Title: "Create Client", ActiveNav: "clients", Error: "Internal error."})
			return
		}
		clientSecret = secret
		hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
		if err != nil {
			h.logger.Error("failed to hash client secret", "error", err)
			h.render(w, r, "client_create", &pageData{Title: "Create Client", ActiveNav: "clients", Error: "Internal error."})
			return
		}
		client.ClientSecretHash = hash
	}

	created, err := h.store.CreateOAuthClient(ctx, client)
	if err != nil {
		h.logger.Error("failed to create client", "error", err)
		h.render(w, r, "client_create", &pageData{Title: "Create Client", ActiveNav: "clients", Error: "Failed to create client."})
		return
	}

	h.auditLog(r, orgID, model.EventClientCreated, "client", created.ID, created.Name)

	if clientSecret != "" {
		h.render(w, r, "client_detail", &pageData{
			Title:        fmt.Sprintf("Client: %s", created.Name),
			ActiveNav:    "clients",
			ClientDetail: created.ToAdminResponse(),
			ClientSecret: clientSecret,
			Flash:        "Client created. Copy the secret now — it won't be shown again.",
		})
		return
	}

	middleware.SetFlash(w, "Client created successfully.")
	http.Redirect(w, r, fmt.Sprintf("/admin/clients/%s", created.ID), http.StatusFound)
}

// ClientDetailPage handles GET /admin/clients/{id}
func (h *AdminConsoleHandler) ClientDetailPage(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		http.Redirect(w, r, "/admin/clients", http.StatusFound)
		return
	}

	ctx := r.Context()
	client, err := h.store.GetOAuthClient(ctx, clientID)
	if err != nil || client == nil {
		middleware.SetFlash(w, "Client not found.")
		http.Redirect(w, r, "/admin/clients", http.StatusFound)
		return
	}

	h.render(w, r, "client_detail", &pageData{
		Title:        fmt.Sprintf("Client: %s", client.Name),
		ActiveNav:    "clients",
		ClientDetail: client.ToAdminResponse(),
	})
}

// UpdateClientAction handles POST /admin/clients/{id}
func (h *AdminConsoleHandler) UpdateClientAction(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		http.Redirect(w, r, "/admin/clients", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf("/admin/clients/%s", clientID), http.StatusFound)
		return
	}

	req := &model.UpdateClientRequest{
		Name:         strings.TrimSpace(r.FormValue("name")),
		Description:  strings.TrimSpace(r.FormValue("description")),
		RedirectURIs: strings.TrimSpace(r.FormValue("redirect_uris")),
		Enabled:      r.FormValue("enabled") == "true",
	}

	if _, err := h.store.UpdateOAuthClient(r.Context(), clientID, req); err != nil {
		h.logger.Error("failed to update client", "error", err)
		middleware.SetFlash(w, "Failed to update client.")
		http.Redirect(w, r, fmt.Sprintf("/admin/clients/%s", clientID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Client updated successfully.")
	http.Redirect(w, r, fmt.Sprintf("/admin/clients/%s", clientID), http.StatusFound)
}

// DeleteClientAction handles POST /admin/clients/{id}/delete
func (h *AdminConsoleHandler) DeleteClientAction(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		http.Redirect(w, r, "/admin/clients", http.StatusFound)
		return
	}

	if err := h.store.DeleteOAuthClient(r.Context(), clientID); err != nil {
		h.logger.Error("failed to delete client", "error", err)
		middleware.SetFlash(w, "Failed to delete client.")
		http.Redirect(w, r, fmt.Sprintf("/admin/clients/%s", clientID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Client deleted.")
	http.Redirect(w, r, "/admin/clients", http.StatusFound)
}

// RegenerateSecretAction handles POST /admin/clients/{id}/regenerate-secret
func (h *AdminConsoleHandler) RegenerateSecretAction(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		http.Redirect(w, r, "/admin/clients", http.StatusFound)
		return
	}

	ctx := r.Context()
	client, err := h.store.GetOAuthClient(ctx, clientID)
	if err != nil || client == nil {
		middleware.SetFlash(w, "Client not found.")
		http.Redirect(w, r, "/admin/clients", http.StatusFound)
		return
	}

	if client.ClientType != "confidential" {
		middleware.SetFlash(w, "Only confidential clients have secrets.")
		http.Redirect(w, r, fmt.Sprintf("/admin/clients/%s", clientID), http.StatusFound)
		return
	}

	secret, err := generateRandomSecret()
	if err != nil {
		h.logger.Error("failed to generate secret", "error", err)
		middleware.SetFlash(w, "Failed to regenerate secret.")
		http.Redirect(w, r, fmt.Sprintf("/admin/clients/%s", clientID), http.StatusFound)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("failed to hash secret", "error", err)
		middleware.SetFlash(w, "Failed to regenerate secret.")
		http.Redirect(w, r, fmt.Sprintf("/admin/clients/%s", clientID), http.StatusFound)
		return
	}

	if err := h.store.UpdateClientSecret(ctx, clientID, hash); err != nil {
		h.logger.Error("failed to update client secret", "error", err)
		middleware.SetFlash(w, "Failed to regenerate secret.")
		http.Redirect(w, r, fmt.Sprintf("/admin/clients/%s", clientID), http.StatusFound)
		return
	}

	h.render(w, r, "client_detail", &pageData{
		Title:        fmt.Sprintf("Client: %s", client.Name),
		ActiveNav:    "clients",
		ClientDetail: client.ToAdminResponse(),
		ClientSecret: secret,
		Flash:        "Secret regenerated. Copy it now — it won't be shown again.",
	})
}

func generateRandomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

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
		h.render(w, r, "roles_list", &pageData{Title: "Roles", ActiveNav: "roles", Error: "Failed to load roles."})
		return
	}

	roleResponses := make([]*model.RoleResponse, len(roles))
	for i, role := range roles {
		count, _ := h.store.CountRoleUsers(ctx, role.ID)
		roleResponses[i] = role.ToRoleResponse(count)
	}

	pg := buildPagination(page, limit, total, "/admin/roles", search)

	if r.Header.Get("HX-Request") == "true" {
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
	h.render(w, r, "role_create", &pageData{Title: "Create Role", ActiveNav: "roles"})
}

// CreateRoleAction handles POST /admin/roles
func (h *AdminConsoleHandler) CreateRoleAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, "role_create", &pageData{Title: "Create Role", ActiveNav: "roles", Error: "Invalid form data."})
		return
	}

	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	name := strings.ToLower(strings.TrimSpace(r.FormValue("name")))
	description := strings.TrimSpace(r.FormValue("description"))

	if name == "" {
		h.render(w, r, "role_create", &pageData{Title: "Create Role", ActiveNav: "roles", Error: "Role name is required."})
		return
	}

	role := &model.Role{
		OrgID:       orgID,
		Name:        name,
		Description: description,
	}

	if _, err := h.store.CreateRole(ctx, role); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			h.render(w, r, "role_create", &pageData{Title: "Create Role", ActiveNav: "roles", Error: "A role with this name already exists."})
			return
		}
		h.logger.Error("failed to create role", "error", err)
		h.render(w, r, "role_create", &pageData{Title: "Create Role", ActiveNav: "roles", Error: "Failed to create role."})
		return
	}

	middleware.SetFlash(w, "Role created successfully.")
	http.Redirect(w, r, "/admin/roles", http.StatusFound)
}

// RoleDetailPage handles GET /admin/roles/{id}
func (h *AdminConsoleHandler) RoleDetailPage(w http.ResponseWriter, r *http.Request) {
	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/roles", http.StatusFound)
		return
	}

	ctx := r.Context()
	role, err := h.store.GetRoleByID(ctx, roleID)
	if err != nil || role == nil {
		middleware.SetFlash(w, "Role not found.")
		http.Redirect(w, r, "/admin/roles", http.StatusFound)
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
		http.Redirect(w, r, "/admin/roles", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf("/admin/roles/%s", roleID), http.StatusFound)
		return
	}

	req := &model.UpdateRoleRequest{
		Name:        strings.ToLower(strings.TrimSpace(r.FormValue("name"))),
		Description: strings.TrimSpace(r.FormValue("description")),
	}

	if _, err := h.store.UpdateRole(r.Context(), roleID, req); err != nil {
		h.logger.Error("failed to update role", "error", err)
		middleware.SetFlash(w, "Failed to update role.")
		http.Redirect(w, r, fmt.Sprintf("/admin/roles/%s", roleID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Role updated successfully.")
	http.Redirect(w, r, fmt.Sprintf("/admin/roles/%s", roleID), http.StatusFound)
}

// DeleteRoleAction handles POST /admin/roles/{id}/delete
func (h *AdminConsoleHandler) DeleteRoleAction(w http.ResponseWriter, r *http.Request) {
	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/roles", http.StatusFound)
		return
	}

	if err := h.store.DeleteRole(r.Context(), roleID); err != nil {
		if strings.Contains(err.Error(), "builtin") {
			middleware.SetFlash(w, "Cannot delete built-in roles.")
		} else {
			h.logger.Error("failed to delete role", "error", err)
			middleware.SetFlash(w, "Failed to delete role.")
		}
		http.Redirect(w, r, fmt.Sprintf("/admin/roles/%s", roleID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Role deleted.")
	http.Redirect(w, r, "/admin/roles", http.StatusFound)
}

// AssignRoleAction handles POST /admin/users/{id}/roles
func (h *AdminConsoleHandler) AssignRoleAction(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	roleID, err := uuid.Parse(r.FormValue("role_id"))
	if err != nil {
		middleware.SetFlash(w, "Invalid role.")
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	if err := h.store.AssignRole(r.Context(), userID, roleID); err != nil {
		h.logger.Error("failed to assign role", "error", err)
		middleware.SetFlash(w, "Failed to assign role.")
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Role assigned.")
	http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
}

// UnassignRoleAction handles POST /admin/users/{id}/roles/{roleId}/delete
func (h *AdminConsoleHandler) UnassignRoleAction(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusFound)
		return
	}

	roleID, err := uuid.Parse(chi.URLParam(r, "roleId"))
	if err != nil {
		middleware.SetFlash(w, "Invalid role.")
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	if err := h.store.UnassignRole(r.Context(), userID, roleID); err != nil {
		h.logger.Error("failed to unassign role", "error", err)
		middleware.SetFlash(w, "Failed to remove role.")
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Role removed.")
	http.Redirect(w, r, fmt.Sprintf("/admin/users/%s", userID), http.StatusFound)
}

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
		h.render(w, r, "events_list", &pageData{Title: "Audit Events", ActiveNav: "events", Error: "Failed to load events."})
		return
	}

	pg := buildPagination(page, limit, total, "/admin/events", search)
	if eventFilter != "" {
		pg.QueryExtra += "&event_type=" + eventFilter
	}

	if r.Header.Get("HX-Request") == "true" {
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

// ListSessionsPage handles GET /admin/sessions
func (h *AdminConsoleHandler) ListSessionsPage(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page := queryInt(r, "page", 1)
	limit := 50
	offset := (page - 1) * limit

	sessions, total, err := h.sessions.ListAll(r.Context(), search, limit, offset)
	if err != nil {
		h.logger.Error("failed to list sessions", "error", err)
		h.render(w, r, "sessions_list", &pageData{Title: "Sessions", ActiveNav: "sessions", Error: "Failed to load sessions."})
		return
	}

	pg := buildPagination(page, limit, total, "/admin/sessions", search)

	if r.Header.Get("HX-Request") == "true" {
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
		http.Redirect(w, r, "/admin/sessions", http.StatusFound)
		return
	}

	if err := h.sessions.Delete(r.Context(), sessionID); err != nil {
		h.logger.Error("failed to revoke session", "error", err)
		middleware.SetFlash(w, "Failed to revoke session.")
	} else {
		authUser := middleware.GetAuthenticatedUser(r.Context())
		h.auditLog(r, authUser.OrgID, model.EventSessionRevoked, "session", sessionID.String(), "")
		middleware.SetFlash(w, "Session revoked.")
	}

	http.Redirect(w, r, "/admin/sessions", http.StatusFound)
}

// RevokeAllSessionsAction handles POST /admin/sessions/revoke-all
func (h *AdminConsoleHandler) RevokeAllSessionsAction(w http.ResponseWriter, r *http.Request) {
	if err := h.sessions.DeleteAll(r.Context()); err != nil {
		h.logger.Error("failed to revoke all sessions", "error", err)
		middleware.SetFlash(w, "Failed to revoke sessions.")
	} else {
		authUser := middleware.GetAuthenticatedUser(r.Context())
		h.auditLog(r, authUser.OrgID, model.EventSessionsRevokedAll, "session", "all", "")
		middleware.SetFlash(w, "All sessions revoked.")
	}

	http.Redirect(w, r, "/admin/sessions", http.StatusFound)
}

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
		h.render(w, r, "groups_list", &pageData{Title: "Groups", ActiveNav: "groups", Error: "Failed to load groups."})
		return
	}

	groupResponses := make([]*model.GroupResponse, len(groups))
	for i, g := range groups {
		memberCount, _ := h.store.CountGroupMembers(ctx, g.ID)
		roleCount, _ := h.store.CountGroupRoles(ctx, g.ID)
		groupResponses[i] = g.ToGroupResponse(memberCount, roleCount)
	}

	pg := buildPagination(page, limit, total, "/admin/groups", search)

	if r.Header.Get("HX-Request") == "true" {
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
	h.render(w, r, "group_create", &pageData{Title: "Create Group", ActiveNav: "groups"})
}

// CreateGroupAction handles POST /admin/groups
func (h *AdminConsoleHandler) CreateGroupAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, "group_create", &pageData{Title: "Create Group", ActiveNav: "groups", Error: "Invalid form data."})
		return
	}

	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))

	if name == "" {
		h.render(w, r, "group_create", &pageData{Title: "Create Group", ActiveNav: "groups", Error: "Group name is required."})
		return
	}

	group := &model.Group{
		OrgID:       orgID,
		Name:        name,
		Description: description,
	}

	if _, err := h.store.CreateGroup(ctx, group); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			h.render(w, r, "group_create", &pageData{Title: "Create Group", ActiveNav: "groups", Error: "A group with this name already exists."})
			return
		}
		h.logger.Error("failed to create group", "error", err)
		h.render(w, r, "group_create", &pageData{Title: "Create Group", ActiveNav: "groups", Error: "Failed to create group."})
		return
	}

	middleware.SetFlash(w, "Group created successfully.")
	http.Redirect(w, r, "/admin/groups", http.StatusFound)
}

// GroupDetailPage handles GET /admin/groups/{id}
func (h *AdminConsoleHandler) GroupDetailPage(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/groups", http.StatusFound)
		return
	}

	ctx := r.Context()
	group, err := h.store.GetGroupByID(ctx, groupID)
	if err != nil || group == nil {
		middleware.SetFlash(w, "Group not found.")
		http.Redirect(w, r, "/admin/groups", http.StatusFound)
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
		http.Redirect(w, r, "/admin/groups", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
		return
	}

	req := &model.UpdateGroupRequest{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
	}

	if _, err := h.store.UpdateGroup(r.Context(), groupID, req); err != nil {
		h.logger.Error("failed to update group", "error", err)
		middleware.SetFlash(w, "Failed to update group.")
		http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Group updated successfully.")
	http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
}

// DeleteGroupAction handles POST /admin/groups/{id}/delete
func (h *AdminConsoleHandler) DeleteGroupAction(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/groups", http.StatusFound)
		return
	}

	if err := h.store.DeleteGroup(r.Context(), groupID); err != nil {
		h.logger.Error("failed to delete group", "error", err)
		middleware.SetFlash(w, "Failed to delete group.")
		http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Group deleted.")
	http.Redirect(w, r, "/admin/groups", http.StatusFound)
}

// AddGroupMemberAction handles POST /admin/groups/{id}/members
func (h *AdminConsoleHandler) AddGroupMemberAction(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/groups", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
		return
	}

	userID, err := uuid.Parse(r.FormValue("user_id"))
	if err != nil {
		middleware.SetFlash(w, "Invalid user.")
		http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
		return
	}

	if err := h.store.AddUserToGroup(r.Context(), userID, groupID); err != nil {
		h.logger.Error("failed to add member to group", "error", err)
		middleware.SetFlash(w, "Failed to add member.")
	} else {
		middleware.SetFlash(w, "Member added.")
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
}

// RemoveGroupMemberAction handles POST /admin/groups/{id}/members/{userId}/delete
func (h *AdminConsoleHandler) RemoveGroupMemberAction(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/groups", http.StatusFound)
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
		return
	}

	if err := h.store.RemoveUserFromGroup(r.Context(), userID, groupID); err != nil {
		h.logger.Error("failed to remove member from group", "error", err)
		middleware.SetFlash(w, "Failed to remove member.")
	} else {
		middleware.SetFlash(w, "Member removed.")
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
}

// AssignGroupRoleAction handles POST /admin/groups/{id}/roles
func (h *AdminConsoleHandler) AssignGroupRoleAction(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/groups", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
		return
	}

	roleID, err := uuid.Parse(r.FormValue("role_id"))
	if err != nil {
		middleware.SetFlash(w, "Invalid role.")
		http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
		return
	}

	if err := h.store.AssignRoleToGroup(r.Context(), groupID, roleID); err != nil {
		h.logger.Error("failed to assign role to group", "error", err)
		middleware.SetFlash(w, "Failed to assign role.")
	} else {
		middleware.SetFlash(w, "Role assigned to group.")
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
}

// UnassignGroupRoleAction handles POST /admin/groups/{id}/roles/{roleId}/delete
func (h *AdminConsoleHandler) UnassignGroupRoleAction(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/groups", http.StatusFound)
		return
	}

	roleID, err := uuid.Parse(chi.URLParam(r, "roleId"))
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
		return
	}

	if err := h.store.UnassignRoleFromGroup(r.Context(), groupID, roleID); err != nil {
		h.logger.Error("failed to unassign role from group", "error", err)
		middleware.SetFlash(w, "Failed to unassign role.")
	} else {
		middleware.SetFlash(w, "Role unassigned from group.")
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/groups/%s", groupID), http.StatusFound)
}

// ExportOrgAction handles GET /admin/organizations/{id}/export — downloads org config as JSON.
func (h *AdminConsoleHandler) ExportOrgAction(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, "/admin/organizations", http.StatusFound)
		return
	}

	ctx := r.Context()
	org, err := h.store.GetOrganizationByID(ctx, orgID)
	if err != nil || org == nil {
		middleware.SetFlash(w, "Organization not found.")
		http.Redirect(w, r, "/admin/organizations", http.StatusFound)
		return
	}

	export := model.OrgExport{
		Organization: model.OrgExportData{
			Name:        org.Name,
			Slug:        org.Slug,
			DisplayName: org.DisplayName,
		},
	}

	if settings, err := h.store.GetOrgSettings(ctx, orgID); err == nil && settings != nil {
		export.Settings = &model.OrgSettingsExport{
			PasswordMinLength:          settings.PasswordMinLength,
			PasswordRequireUppercase:   settings.PasswordRequireUppercase,
			PasswordRequireLowercase:   settings.PasswordRequireLowercase,
			PasswordRequireNumbers:     settings.PasswordRequireNumbers,
			PasswordRequireSymbols:     settings.PasswordRequireSymbols,
			MFAEnforcement:             settings.MFAEnforcement,
			AccessTokenTTLSeconds:      int(settings.AccessTokenTTL.Seconds()),
			RefreshTokenTTLSeconds:     int(settings.RefreshTokenTTL.Seconds()),
			SelfRegistrationEnabled:    settings.SelfRegistrationEnabled,
			EmailVerificationRequired:  settings.EmailVerificationRequired,
			ForgotPasswordEnabled:      settings.ForgotPasswordEnabled,
			RememberMeEnabled:          settings.RememberMeEnabled,
			LoginPageTitle:             settings.LoginPageTitle,
			LoginPageMessage:           settings.LoginPageMessage,
		}
	}

	if roles, _, err := h.store.ListRoles(ctx, orgID, "", 1000, 0); err == nil {
		for _, role := range roles {
			export.Roles = append(export.Roles, model.RoleExport{
				Name:        role.Name,
				Description: role.Description,
			})
		}
	}

	if groups, _, err := h.store.ListGroups(ctx, orgID, "", 1000, 0); err == nil {
		for _, g := range groups {
			ge := model.GroupExport{Name: g.Name, Description: g.Description}
			if groupRoles, err := h.store.GetGroupRoles(ctx, g.ID); err == nil {
				for _, gr := range groupRoles {
					ge.Roles = append(ge.Roles, gr.RoleName)
				}
			}
			export.Groups = append(export.Groups, ge)
		}
	}

	if clients, _, err := h.store.ListOAuthClients(ctx, orgID, "", 1000, 0); err == nil {
		for _, c := range clients {
			export.Clients = append(export.Clients, model.ClientExport{
				ClientID:     c.ID,
				Name:         c.Name,
				Description:  c.Description,
				ClientType:   c.ClientType,
				RedirectURIs: c.RedirectURIs,
				Enabled:      c.Enabled,
			})
		}
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		h.logger.Error("failed to marshal export", "error", err)
		middleware.SetFlash(w, "Failed to export organization.")
		http.Redirect(w, r, fmt.Sprintf("/admin/organizations/%s", orgID), http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="org-%s.json"`, org.Slug))
	w.Write(data)
}

// ImportOrgPage handles GET /admin/organizations/import
func (h *AdminConsoleHandler) ImportOrgPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "org_import", &pageData{Title: "Import Organization", ActiveNav: "organizations"})
}

// ImportOrgAction handles POST /admin/organizations/import
func (h *AdminConsoleHandler) ImportOrgAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.render(w, r, "org_import", &pageData{
			Title: "Import Organization", ActiveNav: "organizations",
			Error: "Invalid form data. Max file size is 10MB.",
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		h.render(w, r, "org_import", &pageData{
			Title: "Import Organization", ActiveNav: "organizations",
			Error: "Please select a JSON file to import.",
		})
		return
	}
	defer file.Close()

	var export model.OrgExport
	if err := json.NewDecoder(file).Decode(&export); err != nil {
		h.render(w, r, "org_import", &pageData{
			Title: "Import Organization", ActiveNav: "organizations",
			Error: "Invalid JSON file format.",
		})
		return
	}

	ctx := r.Context()

	// Create organization
	org, err := h.store.CreateOrganization(ctx, &model.CreateOrgRequest{
		Name:        export.Organization.Name,
		Slug:        export.Organization.Slug,
		DisplayName: export.Organization.DisplayName,
	})
	if err != nil {
		h.render(w, r, "org_import", &pageData{
			Title: "Import Organization", ActiveNav: "organizations",
			Error: fmt.Sprintf("Failed to create organization: %s", err.Error()),
		})
		return
	}

	// Import settings
	if export.Settings != nil {
		h.store.UpdateOrgSettings(ctx, org.ID, &model.UpdateOrgSettingsRequest{
			PasswordMinLength:          export.Settings.PasswordMinLength,
			PasswordRequireUppercase:   export.Settings.PasswordRequireUppercase,
			PasswordRequireLowercase:   export.Settings.PasswordRequireLowercase,
			PasswordRequireNumbers:     export.Settings.PasswordRequireNumbers,
			PasswordRequireSymbols:     export.Settings.PasswordRequireSymbols,
			MFAEnforcement:             export.Settings.MFAEnforcement,
			AccessTokenTTLSeconds:      export.Settings.AccessTokenTTLSeconds,
			RefreshTokenTTLSeconds:     export.Settings.RefreshTokenTTLSeconds,
			SelfRegistrationEnabled:    export.Settings.SelfRegistrationEnabled,
			EmailVerificationRequired:  export.Settings.EmailVerificationRequired,
			ForgotPasswordEnabled:      export.Settings.ForgotPasswordEnabled,
			RememberMeEnabled:          export.Settings.RememberMeEnabled,
			LoginPageTitle:             export.Settings.LoginPageTitle,
			LoginPageMessage:           export.Settings.LoginPageMessage,
		})
	}

	// Import roles
	roleMap := make(map[string]uuid.UUID)
	for _, re := range export.Roles {
		role, err := h.store.CreateRole(ctx, &model.Role{
			OrgID:       org.ID,
			Name:        re.Name,
			Description: re.Description,
		})
		if err == nil {
			roleMap[role.Name] = role.ID
		}
	}

	// Import groups with role assignments
	for _, ge := range export.Groups {
		group, err := h.store.CreateGroup(ctx, &model.Group{
			OrgID:       org.ID,
			Name:        ge.Name,
			Description: ge.Description,
		})
		if err != nil {
			continue
		}
		for _, roleName := range ge.Roles {
			if roleID, ok := roleMap[roleName]; ok {
				h.store.AssignRoleToGroup(ctx, group.ID, roleID)
			}
		}
	}

	middleware.SetFlash(w, fmt.Sprintf("Organization '%s' imported successfully.", org.Name))
	http.Redirect(w, r, fmt.Sprintf("/admin/organizations/%s", org.ID), http.StatusFound)
}

// auditLog is a helper that extracts actor info from the request context and logs an audit event.
func (h *AdminConsoleHandler) auditLog(r *http.Request, orgID uuid.UUID, eventType, targetType, targetID, targetName string) {
	if h.audit == nil {
		return
	}
	authUser := middleware.GetAuthenticatedUser(r.Context())
	var actorID *uuid.UUID
	var actorName string
	if authUser != nil {
		actorID = &authUser.UserID
		actorName = authUser.PreferredUsername
	}
	h.audit.LogSimple(r.Context(), r, orgID, eventType, actorID, actorName, targetType, targetID, targetName)
}

func buildPagination(page, limit, total int, baseURL, search string) *paginationData {
	totalPages := (total + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	from := (page-1)*limit + 1
	to := page * limit
	if to > total {
		to = total
	}
	if total == 0 {
		from = 0
	}

	var pages []int
	for i := 1; i <= totalPages; i++ {
		pages = append(pages, i)
	}

	queryExtra := ""
	if search != "" {
		queryExtra = "&search=" + search
	}

	return &paginationData{
		Page:       page,
		TotalPages: totalPages,
		Total:      total,
		From:       from,
		To:         to,
		Pages:      pages,
		BaseURL:    baseURL,
		QueryExtra: queryExtra,
	}
}
