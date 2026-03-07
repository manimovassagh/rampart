package handler

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/audit"
	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
	"github.com/manimovassagh/rampart/internal/social"
)

//go:embed templates/admin/*.html templates/admin/partials/*.html
var adminTemplateFS embed.FS

//go:embed static/admin.css static/admin.js static/htmx.min.js
var staticFS embed.FS

// Admin console string constants (SonarQube S1192 compliance).
const (
	// Flash / error messages
	msgInvalidForm    = "Invalid form data."
	msgInternalErr    = "Internal error."
	msgInvalidRole    = "Invalid role."
	msgRegenFailed    = "Failed to regenerate secret."
	msgDuplicateKey   = "duplicate key"
	msgInvalidJSON    = "Invalid or malformed JSON request body."
	msgAuthRequired   = "Authentication required."
	msgInternalServer = "Internal server error."
	msgUnexpectedErr  = "An unexpected error occurred."
	msgInvalidLogin   = "Invalid username/email or password."

	// OAuth constants
	oauthServerError = "server_error"
	tokenTypeBearer  = "Bearer"

	// ActiveNav values
	navUsers         = "users"
	navRoles         = "roles"
	navGroups        = "groups"
	navOrganizations = "organizations"
	navClients       = "clients"
	navSessions      = "sessions"
	navEvents        = "events"
	navSocial        = "social"

	// Redirect paths
	pathAdminUsers     = "/admin/users"
	pathAdminUserFmt   = "/admin/users/%s"
	pathAdminOrgs      = "/admin/organizations"
	pathAdminOrgFmt    = "/admin/organizations/%s"
	pathAdminClients   = "/admin/clients"
	pathAdminClientFmt = "/admin/clients/%s"
	pathAdminRoles     = "/admin/roles"
	pathAdminRoleFmt   = "/admin/roles/%s"
	pathAdminGroups    = "/admin/groups"
	pathAdminGroupFmt  = "/admin/groups/%s"
	pathAdminSessions  = "/admin/sessions"
	pathAdminEvents    = "/admin/events"

	// Page titles
	titleCreateUser   = "Create User"
	titleCreateClient = "Create Client"
	titleCreateOrg    = "Create Organization"
	titleCreateRole   = "Create Role"
	titleCreateGroup  = "Create Group"
	titleImportOrg    = "Import Organization"

	// Template names
	tmplUserCreate   = "user_create"
	tmplClientCreate = "client_create"
	tmplOrgCreate    = "org_create"
	tmplRoleCreate   = "role_create"
	tmplGroupCreate  = "group_create"
	tmplOrgImport    = "org_import"

	// Form values
	formValueTrue          = "true"
	clientTypeConfidential = "confidential"

	// HTMX header
	headerHXRequest = "HX-Request"

	// Content types
	contentTypeHTML = "text/html; charset=utf-8"
	cacheNoStore    = "no-store"
)

// StaticHandler returns an http.Handler that serves embedded static assets (CSS, JS).
func StaticHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("failed to create static sub-filesystem: " + err.Error())
	}
	return http.FileServer(http.FS(sub))
}

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

	// Social provider config operations
	UpsertSocialProviderConfig(ctx context.Context, cfg *model.SocialProviderConfig) error
	ListSocialProviderConfigs(ctx context.Context, orgID uuid.UUID) ([]*model.SocialProviderConfig, error)
	DeleteSocialProviderConfig(ctx context.Context, orgID uuid.UUID, provider string) error
}

// AdminConsoleSessionStore defines session operations for the admin console.
type AdminConsoleSessionStore interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*session.Session, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	CountActive(ctx context.Context) (int, error)
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	Delete(ctx context.Context, sessionID uuid.UUID) error
	ListAll(ctx context.Context, search string, limit, offset int) ([]*session.WithUser, int, error)
	DeleteAll(ctx context.Context) error
}

// SocialProviderInfo holds display data for a social provider on the admin page.
type SocialProviderInfo struct {
	Name        string
	Label       string
	Enabled     bool
	ClientID    string
	HasSecret   bool
	ExtraConfig map[string]string
	ExtraFields []SocialProviderField
}

// SocialProviderField describes an extra config field for a provider.
type SocialProviderField struct {
	Key   string
	Label string
	Value string
}

// AdminConsoleHandler serves SSR admin pages.
type AdminConsoleHandler struct {
	store          AdminConsoleStore
	sessions       AdminConsoleSessionStore
	audit          *audit.Logger
	logger         *slog.Logger
	issuer         string
	pages          map[string]*template.Template
	socialRegistry *social.Registry
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
func NewAdminConsoleHandler(store AdminConsoleStore, sessions AdminConsoleSessionStore, logger *slog.Logger, issuer string, auditLogger *audit.Logger, socialReg *social.Registry) *AdminConsoleHandler {
	pages := map[string]*template.Template{
		"dashboard":        parseAdminPage("dashboard.html"),
		"users_list":       parseAdminPage("users_list.html"),
		"user_create":      parseAdminPage("user_create.html"),
		"user_detail":      parseAdminPage("user_detail.html"),
		"orgs_list":        parseAdminPage("orgs_list.html"),
		"org_create":       parseAdminPage("org_create.html"),
		"org_detail":       parseAdminPage("org_detail.html"),
		"clients_list":     parseAdminPage("clients_list.html"),
		"client_create":    parseAdminPage("client_create.html"),
		"client_detail":    parseAdminPage("client_detail.html"),
		"roles_list":       parseAdminPage("roles_list.html"),
		"role_create":      parseAdminPage("role_create.html"),
		"role_detail":      parseAdminPage("role_detail.html"),
		"events_list":      parseAdminPage("events_list.html"),
		"sessions_list":    parseAdminPage("sessions_list.html"),
		"groups_list":      parseAdminPage("groups_list.html"),
		"group_create":     parseAdminPage("group_create.html"),
		"group_detail":     parseAdminPage("group_detail.html"),
		"org_import":       parseAdminPage("org_import.html"),
		"oidc":             parseAdminPage("oidc.html"),
		"social_providers": parseAdminPage("social_providers.html"),
	}

	return &AdminConsoleHandler{
		store:          store,
		sessions:       sessions,
		audit:          auditLogger,
		logger:         logger,
		issuer:         issuer,
		pages:          pages,
		socialRegistry: socialReg,
	}
}

// pageData holds common data passed to all admin templates.
type pageData struct {
	Title     string
	ActiveNav string
	User      *middleware.AuthenticatedUser
	OrgName   string
	CSRFToken string
	Flash     string
	Error     string

	// Page-specific data
	Stats           *model.DashboardStats
	Users           []*model.AdminUserResponse
	UserDetail      *model.AdminUserResponse
	Sessions        []*session.Session
	Orgs            []*model.OrgResponse
	OrgDetail       *model.Organization
	OrgSettings     *model.OrgSettingsResponse
	Clients         []*model.AdminClientResponse
	ClientDetail    *model.AdminClientResponse
	ClientSecret    string
	Roles           []*model.RoleResponse
	RoleDetail      *model.RoleResponse
	RoleUsers       []*model.UserRoleAssignment
	UserRoles       []*model.Role
	AllRoles        []*model.Role
	Events          []*model.AuditEvent
	EventFilter     string
	StatusFilter    string
	GlobalSessions  []*session.WithUser
	Groups          []*model.GroupResponse
	GroupDetail     *model.GroupResponse
	GroupMembers    []*model.GroupMember
	GroupRoles      []*model.GroupRoleAssignment
	UserGroups      []*model.Group
	OIDC            *DiscoveryResponse
	SocialProviders []SocialProviderInfo
	Search          string
	Pagination      *paginationData
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

	if data.User != nil && data.OrgName == "" {
		if org, err := h.store.GetOrganizationByID(r.Context(), data.User.OrgID); err == nil && org != nil {
			data.OrgName = org.Name
		}
	}

	flash := middleware.GetFlash(w, r)
	if flash != "" {
		data.Flash = flash
	}

	w.Header().Set("Content-Type", contentTypeHTML)
	w.Header().Set("Cache-Control", cacheNoStore)
	w.Header().Set("X-Frame-Options", "DENY")

	tmpl, ok := h.pages[pageName]
	if !ok {
		h.logger.Error("unknown page template", "page", pageName)
		http.Error(w, msgInternalErr, http.StatusInternalServerError)
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

	w.Header().Set("Content-Type", contentTypeHTML)
	w.Header().Set("Cache-Control", cacheNoStore)

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
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, Error: strings.Join(errors, " ")})
		return
	}

	// Check duplicates
	if existing, _ := h.store.GetUserByEmail(ctx, email, orgID); existing != nil {
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, Error: "A user with this email already exists."})
		return
	}
	if existing, _ := h.store.GetUserByUsername(ctx, username, orgID); existing != nil {
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, Error: "A user with this username already exists."})
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, Error: msgInternalErr})
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
		h.render(w, r, tmplUserCreate, &pageData{Title: titleCreateUser, ActiveNav: navUsers, Error: "Failed to create user."})
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

	if err := h.sessions.DeleteByUserID(ctx, userID); err != nil {
		h.logger.Error("failed to delete user sessions", "error", err)
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

	if err := h.sessions.DeleteByUserID(r.Context(), userID); err != nil {
		h.logger.Error("failed to revoke sessions", "error", err)
		middleware.SetFlash(w, "Failed to revoke sessions.")
	} else {
		sessAuthUser := middleware.GetAuthenticatedUser(r.Context())
		h.auditLog(r, sessAuthUser.OrgID, model.EventSessionRevoked, "user", userID.String(), "")
		middleware.SetFlash(w, "All sessions revoked.")
	}

	http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
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
		h.render(w, r, "orgs_list", &pageData{Title: "Organizations", ActiveNav: navOrganizations, Error: "Failed to load organizations."})
		return
	}

	orgResponses := make([]*model.OrgResponse, len(orgs))
	for i, o := range orgs {
		count, _ := h.store.CountUsers(ctx, o.ID)
		orgResponses[i] = o.ToOrgResponse(count)
	}

	pg := buildPagination(page, limit, total, pathAdminOrgs, search)

	if r.Header.Get(headerHXRequest) == formValueTrue {
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
	h.render(w, r, tmplOrgCreate, &pageData{Title: titleCreateOrg, ActiveNav: navOrganizations})
}

// CreateOrgAction handles POST /admin/organizations
func (h *AdminConsoleHandler) CreateOrgAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, tmplOrgCreate, &pageData{Title: titleCreateOrg, ActiveNav: navOrganizations, Error: msgInvalidForm})
		return
	}

	req := &model.CreateOrgRequest{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Slug:        strings.ToLower(strings.TrimSpace(r.FormValue("slug"))),
		DisplayName: strings.TrimSpace(r.FormValue("display_name")),
	}

	if req.Name == "" || req.Slug == "" {
		h.render(w, r, tmplOrgCreate, &pageData{Title: titleCreateOrg, ActiveNav: navOrganizations, Error: "Name and slug are required."})
		return
	}

	newOrg, err := h.store.CreateOrganization(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), msgDuplicateKey) || strings.Contains(err.Error(), "unique") {
			h.render(w, r, tmplOrgCreate, &pageData{Title: titleCreateOrg, ActiveNav: navOrganizations, Error: "An organization with this slug already exists."})
			return
		}
		h.logger.Error("failed to create organization", "error", err)
		h.render(w, r, tmplOrgCreate, &pageData{Title: titleCreateOrg, ActiveNav: navOrganizations, Error: "Failed to create organization."})
		return
	}

	orgAuthUser := middleware.GetAuthenticatedUser(r.Context())
	h.auditLog(r, orgAuthUser.OrgID, model.EventOrgCreated, "organization", newOrg.ID.String(), req.Name)
	middleware.SetFlash(w, "Organization created successfully.")
	http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
}

// OrgDetailPage handles GET /admin/organizations/{id}
func (h *AdminConsoleHandler) OrgDetailPage(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
		return
	}

	ctx := r.Context()

	org, err := h.store.GetOrganizationByID(ctx, orgID)
	if err != nil || org == nil {
		middleware.SetFlash(w, "Organization not found.")
		http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
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
		http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
		return
	}

	req := &model.UpdateOrgRequest{
		Name:        strings.TrimSpace(r.FormValue("name")),
		DisplayName: strings.TrimSpace(r.FormValue("display_name")),
		Enabled:     r.FormValue("enabled") == formValueTrue,
	}

	if _, err := h.store.UpdateOrganization(r.Context(), orgID, req); err != nil {
		h.logger.Error("failed to update organization", "error", err)
		middleware.SetFlash(w, "Failed to update organization.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Organization updated successfully.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
}

// UpdateOrgSettingsAction handles POST /admin/organizations/{id}/settings
func (h *AdminConsoleHandler) UpdateOrgSettingsAction(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
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
	if mfa != mfaOff && mfa != mfaOptional && mfa != mfaRequired {
		mfa = mfaOff
	}

	req := &model.UpdateOrgSettingsRequest{
		PasswordMinLength:         minLen,
		PasswordRequireUppercase:  r.FormValue("password_require_uppercase") == formValueTrue,
		PasswordRequireLowercase:  r.FormValue("password_require_lowercase") == formValueTrue,
		PasswordRequireNumbers:    r.FormValue("password_require_numbers") == formValueTrue,
		PasswordRequireSymbols:    r.FormValue("password_require_symbols") == formValueTrue,
		MFAEnforcement:            mfa,
		AccessTokenTTLSeconds:     accessTTL,
		RefreshTokenTTLSeconds:    refreshTTL,
		SelfRegistrationEnabled:   r.FormValue("self_registration_enabled") == formValueTrue,
		EmailVerificationRequired: r.FormValue("email_verification_required") == formValueTrue,
		ForgotPasswordEnabled:     r.FormValue("forgot_password_enabled") == formValueTrue,
		RememberMeEnabled:         r.FormValue("remember_me_enabled") == formValueTrue,
		LoginPageTitle:            strings.TrimSpace(r.FormValue("login_page_title")),
		LoginPageMessage:          strings.TrimSpace(r.FormValue("login_page_message")),
		LoginTheme:                strings.TrimSpace(r.FormValue("login_theme")),
	}

	if _, err := h.store.UpdateOrgSettings(r.Context(), orgID, req); err != nil {
		h.logger.Error("failed to update org settings", "error", err)
		middleware.SetFlash(w, "Failed to update settings.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Settings updated successfully.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
}

// DeleteOrgAction handles POST /admin/organizations/{id}/delete
func (h *AdminConsoleHandler) DeleteOrgAction(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
		return
	}

	if err := h.store.DeleteOrganization(r.Context(), orgID); err != nil {
		if strings.Contains(err.Error(), "default") {
			middleware.SetFlash(w, "Cannot delete the default organization.")
		} else {
			h.logger.Error("failed to delete organization", "error", err)
			middleware.SetFlash(w, "Failed to delete organization.")
		}
		http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Organization deleted.")
	http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
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

// socialProviderDef defines the known providers and their extra fields.
type socialProviderDef struct {
	name        string
	label       string
	extraFields []SocialProviderField
}

var knownProviders = []socialProviderDef{
	{"google", "Google OAuth 2.0", nil},
	{"github", "GitHub OAuth", nil},
	{"apple", "Apple Sign In", []SocialProviderField{
		{Key: "team_id", Label: "Team ID"},
		{Key: "key_id", Label: "Key ID"},
	}},
}

// SocialProvidersPage handles GET /admin/social
func (h *AdminConsoleHandler) SocialProvidersPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	dbConfigs, err := h.store.ListSocialProviderConfigs(ctx, orgID)
	if err != nil {
		h.logger.Error("failed to list social provider configs", "error", err)
	}
	configMap := make(map[string]*model.SocialProviderConfig, len(dbConfigs))
	for _, c := range dbConfigs {
		configMap[c.Provider] = c
	}

	providers := make([]SocialProviderInfo, 0, len(knownProviders))
	for _, def := range knownProviders {
		info := SocialProviderInfo{
			Name:        def.name,
			Label:       def.label,
			ExtraFields: def.extraFields,
		}

		if cfg, ok := configMap[def.name]; ok {
			info.Enabled = cfg.Enabled
			info.ClientID = cfg.ClientID
			info.HasSecret = cfg.ClientSecret != ""
			info.ExtraConfig = cfg.ExtraConfig
			for i, f := range info.ExtraFields {
				if v, exists := cfg.ExtraConfig[f.Key]; exists {
					info.ExtraFields[i].Value = v
				}
			}
		} else {
			_, info.Enabled = h.socialRegistry.Get(def.name)
		}

		providers = append(providers, info)
	}

	h.render(w, r, "social_providers", &pageData{
		Title:           "Social Providers",
		ActiveNav:       navSocial,
		SocialProviders: providers,
		Flash:           r.URL.Query().Get("flash"),
		Error:           r.URL.Query().Get("error"),
	})
}

// UpdateSocialProviderAction handles POST /admin/social/{provider}
func (h *AdminConsoleHandler) UpdateSocialProviderAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID
	provider := chi.URLParam(r, "provider")

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/social?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	clientID := strings.TrimSpace(r.FormValue("client_id"))
	clientSecret := strings.TrimSpace(r.FormValue("client_secret"))
	enabled := r.FormValue("enabled") == "on"

	cfg := &model.SocialProviderConfig{
		OrgID:        orgID,
		Provider:     provider,
		Enabled:      enabled,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		ExtraConfig:  make(map[string]string),
	}

	for _, def := range knownProviders {
		if def.name == provider {
			for _, f := range def.extraFields {
				if v := strings.TrimSpace(r.FormValue(f.Key)); v != "" {
					cfg.ExtraConfig[f.Key] = v
				}
			}
			break
		}
	}

	if err := h.store.UpsertSocialProviderConfig(ctx, cfg); err != nil {
		h.logger.Error("failed to save social provider config", "provider", provider, "error", err)
		http.Redirect(w, r, "/admin/social?error=Failed+to+save+configuration", http.StatusSeeOther)
		return
	}

	h.refreshSocialProvider(provider, cfg)
	http.Redirect(w, r, "/admin/social?flash=Provider+configuration+saved", http.StatusSeeOther)
}

// refreshSocialProvider updates the in-memory registry after a config change.
func (h *AdminConsoleHandler) refreshSocialProvider(provider string, cfg *model.SocialProviderConfig) {
	if !cfg.Enabled || cfg.ClientID == "" {
		h.socialRegistry.Unregister(provider)
		return
	}

	switch provider {
	case "google":
		secret := cfg.ClientSecret
		if secret == "" {
			if existing, ok := h.socialRegistry.Get(provider); ok {
				if gp, isGoogle := existing.(*social.GoogleProvider); isGoogle {
					secret = gp.ClientSecret
				}
			}
		}
		h.socialRegistry.Register(&social.GoogleProvider{
			ClientID:     cfg.ClientID,
			ClientSecret: secret,
		})
	case "github":
		secret := cfg.ClientSecret
		if secret == "" {
			if existing, ok := h.socialRegistry.Get(provider); ok {
				if gp, isGitHub := existing.(*social.GitHubProvider); isGitHub {
					secret = gp.ClientSecret
				}
			}
		}
		h.socialRegistry.Register(&social.GitHubProvider{
			ClientID:     cfg.ClientID,
			ClientSecret: secret,
		})
	case "apple":
		h.socialRegistry.Register(&social.AppleProvider{
			ClientID: cfg.ClientID,
			TeamID:   cfg.ExtraConfig["team_id"],
			KeyID:    cfg.ExtraConfig["key_id"],
		})
	}
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
		h.render(w, r, "clients_list", &pageData{Title: "OAuth Clients", ActiveNav: navClients, Error: "Failed to load clients."})
		return
	}

	adminClients := make([]*model.AdminClientResponse, len(clients))
	for i, c := range clients {
		adminClients[i] = c.ToAdminResponse()
	}

	pg := buildPagination(page, limit, total, pathAdminClients, search)

	if r.Header.Get(headerHXRequest) == formValueTrue {
		h.renderPartial(w, r, "clients_list", "clients_table", &pageData{Clients: adminClients, Search: search, Pagination: pg})
		return
	}

	h.render(w, r, "clients_list", &pageData{
		Title:      "OAuth Clients",
		ActiveNav:  navClients,
		Clients:    adminClients,
		Search:     search,
		Pagination: pg,
	})
}

// CreateClientPage handles GET /admin/clients/new
func (h *AdminConsoleHandler) CreateClientPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, tmplClientCreate, &pageData{Title: titleCreateClient, ActiveNav: navClients})
}

// CreateClientAction handles POST /admin/clients
func (h *AdminConsoleHandler) CreateClientAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, tmplClientCreate, &pageData{Title: titleCreateClient, ActiveNav: navClients, Error: msgInvalidForm})
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
		h.render(w, r, tmplClientCreate, &pageData{Title: titleCreateClient, ActiveNav: navClients, Error: "Client name is required."})
		return
	}

	if clientType != "public" && clientType != clientTypeConfidential {
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
	if clientType == clientTypeConfidential {
		secret, err := generateRandomSecret()
		if err != nil {
			h.logger.Error("failed to generate client secret", "error", err)
			h.render(w, r, tmplClientCreate, &pageData{Title: titleCreateClient, ActiveNav: navClients, Error: msgInternalErr})
			return
		}
		clientSecret = secret
		hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
		if err != nil {
			h.logger.Error("failed to hash client secret", "error", err)
			h.render(w, r, tmplClientCreate, &pageData{Title: titleCreateClient, ActiveNav: navClients, Error: msgInternalErr})
			return
		}
		client.ClientSecretHash = hash
	}

	created, err := h.store.CreateOAuthClient(ctx, client)
	if err != nil {
		h.logger.Error("failed to create client", "error", err)
		h.render(w, r, tmplClientCreate, &pageData{Title: titleCreateClient, ActiveNav: navClients, Error: "Failed to create client."})
		return
	}

	h.auditLog(r, orgID, model.EventClientCreated, "client", created.ID, created.Name)

	if clientSecret != "" {
		h.render(w, r, "client_detail", &pageData{
			Title:        fmt.Sprintf("Client: %s", created.Name),
			ActiveNav:    navClients,
			ClientDetail: created.ToAdminResponse(),
			ClientSecret: clientSecret,
			Flash:        "Client created. Copy the secret now — it won't be shown again.",
		})
		return
	}

	middleware.SetFlash(w, "Client created successfully.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, created.ID), http.StatusFound)
}

// ClientDetailPage handles GET /admin/clients/{id}
func (h *AdminConsoleHandler) ClientDetailPage(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		http.Redirect(w, r, pathAdminClients, http.StatusFound)
		return
	}

	ctx := r.Context()
	client, err := h.store.GetOAuthClient(ctx, clientID)
	if err != nil || client == nil {
		middleware.SetFlash(w, "Client not found.")
		http.Redirect(w, r, pathAdminClients, http.StatusFound)
		return
	}

	h.render(w, r, "client_detail", &pageData{
		Title:        fmt.Sprintf("Client: %s", client.Name),
		ActiveNav:    navClients,
		ClientDetail: client.ToAdminResponse(),
	})
}

// UpdateClientAction handles POST /admin/clients/{id}
func (h *AdminConsoleHandler) UpdateClientAction(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		http.Redirect(w, r, pathAdminClients, http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	req := &model.UpdateClientRequest{
		Name:         strings.TrimSpace(r.FormValue("name")),
		Description:  strings.TrimSpace(r.FormValue("description")),
		RedirectURIs: strings.TrimSpace(r.FormValue("redirect_uris")),
		Enabled:      r.FormValue("enabled") == formValueTrue,
	}

	if _, err := h.store.UpdateOAuthClient(r.Context(), clientID, req); err != nil {
		h.logger.Error("failed to update client", "error", err)
		middleware.SetFlash(w, "Failed to update client.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Client updated successfully.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
}

// DeleteClientAction handles POST /admin/clients/{id}/delete
func (h *AdminConsoleHandler) DeleteClientAction(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		http.Redirect(w, r, pathAdminClients, http.StatusFound)
		return
	}

	if err := h.store.DeleteOAuthClient(r.Context(), clientID); err != nil {
		h.logger.Error("failed to delete client", "error", err)
		middleware.SetFlash(w, "Failed to delete client.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Client deleted.")
	http.Redirect(w, r, pathAdminClients, http.StatusFound)
}

// RegenerateSecretAction handles POST /admin/clients/{id}/regenerate-secret
func (h *AdminConsoleHandler) RegenerateSecretAction(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	if clientID == "" {
		http.Redirect(w, r, pathAdminClients, http.StatusFound)
		return
	}

	ctx := r.Context()
	client, err := h.store.GetOAuthClient(ctx, clientID)
	if err != nil || client == nil {
		middleware.SetFlash(w, "Client not found.")
		http.Redirect(w, r, pathAdminClients, http.StatusFound)
		return
	}

	if client.ClientType != clientTypeConfidential {
		middleware.SetFlash(w, "Only confidential clients have secrets.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	secret, err := generateRandomSecret()
	if err != nil {
		h.logger.Error("failed to generate secret", "error", err)
		middleware.SetFlash(w, msgRegenFailed)
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("failed to hash secret", "error", err)
		middleware.SetFlash(w, msgRegenFailed)
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	if err := h.store.UpdateClientSecret(ctx, clientID, hash); err != nil {
		h.logger.Error("failed to update client secret", "error", err)
		middleware.SetFlash(w, msgRegenFailed)
		http.Redirect(w, r, fmt.Sprintf(pathAdminClientFmt, clientID), http.StatusFound)
		return
	}

	h.render(w, r, "client_detail", &pageData{
		Title:        fmt.Sprintf("Client: %s", client.Name),
		ActiveNav:    navClients,
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
		middleware.SetFlash(w, "Invalid form data.")
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

	middleware.SetFlash(w, "Role removed.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminUserFmt, userID), http.StatusFound)
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

// ListSessionsPage handles GET /admin/sessions
func (h *AdminConsoleHandler) ListSessionsPage(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page := queryInt(r, "page", 1)
	limit := 50
	offset := (page - 1) * limit

	sessions, total, err := h.sessions.ListAll(r.Context(), search, limit, offset)
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
		authUser := middleware.GetAuthenticatedUser(r.Context())
		h.auditLog(r, authUser.OrgID, model.EventSessionRevoked, "session", sessionID.String(), "")
		middleware.SetFlash(w, "Session revoked.")
	}

	http.Redirect(w, r, pathAdminSessions, http.StatusFound)
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

	http.Redirect(w, r, pathAdminSessions, http.StatusFound)
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

// ExportOrgAction handles GET /admin/organizations/{id}/export — downloads org config as JSON.
func (h *AdminConsoleHandler) ExportOrgAction(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
		return
	}

	ctx := r.Context()
	org, err := h.store.GetOrganizationByID(ctx, orgID)
	if err != nil || org == nil {
		middleware.SetFlash(w, "Organization not found.")
		http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
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
			PasswordMinLength:         settings.PasswordMinLength,
			PasswordRequireUppercase:  settings.PasswordRequireUppercase,
			PasswordRequireLowercase:  settings.PasswordRequireLowercase,
			PasswordRequireNumbers:    settings.PasswordRequireNumbers,
			PasswordRequireSymbols:    settings.PasswordRequireSymbols,
			MFAEnforcement:            settings.MFAEnforcement,
			AccessTokenTTLSeconds:     int(settings.AccessTokenTTL.Seconds()),
			RefreshTokenTTLSeconds:    int(settings.RefreshTokenTTL.Seconds()),
			SelfRegistrationEnabled:   settings.SelfRegistrationEnabled,
			EmailVerificationRequired: settings.EmailVerificationRequired,
			ForgotPasswordEnabled:     settings.ForgotPasswordEnabled,
			RememberMeEnabled:         settings.RememberMeEnabled,
			LoginPageTitle:            settings.LoginPageTitle,
			LoginPageMessage:          settings.LoginPageMessage,
			LoginTheme:                settings.LoginTheme,
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
		http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="org-%s.json"`, org.Slug))
	if _, err := w.Write(data); err != nil {
		h.logger.Error("failed to write export response", "error", err)
	}
}

// ImportOrgPage handles GET /admin/organizations/import
func (h *AdminConsoleHandler) ImportOrgPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, tmplOrgImport, &pageData{Title: titleImportOrg, ActiveNav: navOrganizations})
}

// ImportOrgAction handles POST /admin/organizations/import
func (h *AdminConsoleHandler) ImportOrgAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.render(w, r, tmplOrgImport, &pageData{
			Title: titleImportOrg, ActiveNav: navOrganizations,
			Error: "Invalid form data. Max file size is 10MB.",
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		h.render(w, r, tmplOrgImport, &pageData{
			Title: titleImportOrg, ActiveNav: navOrganizations,
			Error: "Please select a JSON file to import.",
		})
		return
	}
	defer func() { _ = file.Close() }()

	var export model.OrgExport
	if err := json.NewDecoder(file).Decode(&export); err != nil {
		h.render(w, r, tmplOrgImport, &pageData{
			Title: titleImportOrg, ActiveNav: navOrganizations,
			Error: "Invalid JSON file format.",
		})
		return
	}

	// Validate import size limits to prevent DoS
	const maxImportItems = 100
	if len(export.Roles) > maxImportItems || len(export.Groups) > maxImportItems || len(export.Clients) > maxImportItems {
		h.render(w, r, tmplOrgImport, &pageData{
			Title: titleImportOrg, ActiveNav: navOrganizations,
			Error: fmt.Sprintf("Import exceeds maximum of %d items per category.", maxImportItems),
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
		h.logger.Error("failed to import organization", "error", err)
		msg := "Failed to create organization."
		if strings.Contains(err.Error(), msgDuplicateKey) || strings.Contains(err.Error(), "unique") {
			msg = "An organization with this slug already exists."
		}
		h.render(w, r, tmplOrgImport, &pageData{
			Title: titleImportOrg, ActiveNav: navOrganizations,
			Error: msg,
		})
		return
	}

	// Import settings
	if export.Settings != nil {
		_, _ = h.store.UpdateOrgSettings(ctx, org.ID, &model.UpdateOrgSettingsRequest{
			PasswordMinLength:         export.Settings.PasswordMinLength,
			PasswordRequireUppercase:  export.Settings.PasswordRequireUppercase,
			PasswordRequireLowercase:  export.Settings.PasswordRequireLowercase,
			PasswordRequireNumbers:    export.Settings.PasswordRequireNumbers,
			PasswordRequireSymbols:    export.Settings.PasswordRequireSymbols,
			MFAEnforcement:            export.Settings.MFAEnforcement,
			AccessTokenTTLSeconds:     export.Settings.AccessTokenTTLSeconds,
			RefreshTokenTTLSeconds:    export.Settings.RefreshTokenTTLSeconds,
			SelfRegistrationEnabled:   export.Settings.SelfRegistrationEnabled,
			EmailVerificationRequired: export.Settings.EmailVerificationRequired,
			ForgotPasswordEnabled:     export.Settings.ForgotPasswordEnabled,
			RememberMeEnabled:         export.Settings.RememberMeEnabled,
			LoginPageTitle:            export.Settings.LoginPageTitle,
			LoginPageMessage:          export.Settings.LoginPageMessage,
			LoginTheme:                export.Settings.LoginTheme,
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
				_ = h.store.AssignRoleToGroup(ctx, group.ID, roleID)
			}
		}
	}

	middleware.SetFlash(w, fmt.Sprintf("Organization '%s' imported successfully.", org.Name))
	http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, org.ID), http.StatusFound)
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
		queryExtra = "&search=" + url.QueryEscape(search)
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

// buildPaginationWithExtra is like buildPagination but appends an additional filter param.
func buildPaginationWithExtra(page, limit, total int, baseURL, search, filterValue, filterKey string) *paginationData {
	pg := buildPagination(page, limit, total, baseURL, search)
	if filterValue != "" {
		pg.QueryExtra += "&" + url.QueryEscape(filterKey) + "=" + url.QueryEscape(filterValue)
	}
	return pg
}
