package handler

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/audit"
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

