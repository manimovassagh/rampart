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
	"github.com/manimovassagh/rampart/internal/plugin"
	"github.com/manimovassagh/rampart/internal/session"
	"github.com/manimovassagh/rampart/internal/social"
	"github.com/manimovassagh/rampart/internal/store"
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
	navWebhooks      = "webhooks"
	navSAML          = "saml-providers"
	navPlugins       = "plugins"
	navCompliance    = "compliance"

	// Redirect paths
	pathAdminUsers         = "/admin/users"
	pathAdminUserFmt       = "/admin/users/%s"
	pathAdminOrgs          = "/admin/organizations"
	pathAdminOrgFmt        = "/admin/organizations/%s"
	pathAdminClients       = "/admin/clients"
	pathAdminClientFmt     = "/admin/clients/%s"
	pathAdminRoles         = "/admin/roles"
	pathAdminRoleFmt       = "/admin/roles/%s"
	pathAdminGroups        = "/admin/groups"
	pathAdminGroupFmt      = "/admin/groups/%s"
	pathAdminSessions      = "/admin/sessions"
	pathAdminEvents        = "/admin/events"
	pathAdminWebhooks      = "/admin/webhooks"
	pathAdminWebhookFmt    = "/admin/webhooks/%s"
	pathAdminSAMLProviders = "/admin/saml-providers"
	pathAdminSAMLFmt       = "/admin/saml-providers/%s"

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
	store.UserReader
	store.UserWriter
	store.UserLister
	store.OrgReader
	store.OrgWriter
	store.OrgLister
	store.OrgSettingsReadWriter
	store.OAuthClientReader
	store.OAuthClientWriter
	store.OAuthClientLister
	store.RoleReader
	store.RoleWriter
	store.RoleLister
	store.AuditStore
	store.GroupReader
	store.GroupWriter
	store.GroupLister
	store.SocialProviderConfigStore
	store.WebhookReader
	store.WebhookWriter
	store.WebhookLister
	store.WebhookDeliveryStore
	store.SAMLProviderStore
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
	plugins        *plugin.Registry
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
func NewAdminConsoleHandler(s AdminConsoleStore, sessions AdminConsoleSessionStore, logger *slog.Logger, issuer string, auditLogger *audit.Logger, socialReg *social.Registry, pluginReg *plugin.Registry) *AdminConsoleHandler {
	pages := map[string]*template.Template{
		"dashboard":            parseAdminPage("dashboard.html"),
		"users_list":           parseAdminPage("users_list.html"),
		"user_create":          parseAdminPage("user_create.html"),
		"user_detail":          parseAdminPage("user_detail.html"),
		"orgs_list":            parseAdminPage("orgs_list.html"),
		"org_create":           parseAdminPage("org_create.html"),
		"org_detail":           parseAdminPage("org_detail.html"),
		"clients_list":         parseAdminPage("clients_list.html"),
		"client_create":        parseAdminPage("client_create.html"),
		"client_detail":        parseAdminPage("client_detail.html"),
		"roles_list":           parseAdminPage("roles_list.html"),
		"role_create":          parseAdminPage("role_create.html"),
		"role_detail":          parseAdminPage("role_detail.html"),
		"events_list":          parseAdminPage("events_list.html"),
		"sessions_list":        parseAdminPage("sessions_list.html"),
		"groups_list":          parseAdminPage("groups_list.html"),
		"group_create":         parseAdminPage("group_create.html"),
		"group_detail":         parseAdminPage("group_detail.html"),
		"org_import":           parseAdminPage("org_import.html"),
		"oidc":                 parseAdminPage("oidc.html"),
		"social_providers":     parseAdminPage("social_providers.html"),
		"webhooks_list":        parseAdminPage("webhooks_list.html"),
		"webhook_create":       parseAdminPage("webhook_create.html"),
		"webhook_detail":       parseAdminPage("webhook_detail.html"),
		"saml_providers_list":  parseAdminPage("saml_providers_list.html"),
		"saml_provider_create": parseAdminPage("saml_provider_create.html"),
		"saml_provider_detail": parseAdminPage("saml_provider_detail.html"),
		"plugins_list":         parseAdminPage("plugins_list.html"),
		"compliance":           parseAdminPage("compliance.html"),
	}

	return &AdminConsoleHandler{
		store:          s,
		sessions:       sessions,
		audit:          auditLogger,
		logger:         logger,
		issuer:         issuer,
		pages:          pages,
		socialRegistry: socialReg,
		plugins:        pluginReg,
	}
}

// pageData holds common data passed to all admin templates.
type pageData struct {
	Title      string
	ActiveNav  string
	User       *middleware.AuthenticatedUser
	OrgName    string
	CSRFToken  string
	Flash      string
	Error      string
	FormErrors map[string]string // per-field validation errors
	FormValues map[string]string // preserved form input on validation failure

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
	SearchUsers     []*model.User
	Webhooks        []*model.Webhook
	WebhookDetail   *model.Webhook
	WebhookSecret   string
	Deliveries      []*model.WebhookDelivery
	SAMLProviders   []*model.SAMLProvider
	SAMLDetail      *model.SAMLProvider
	Plugins         []plugin.Info
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

	// Ensure maps are initialized so templates can safely use {{index}}
	if data.FormErrors == nil {
		data.FormErrors = map[string]string{}
	}
	if data.FormValues == nil {
		data.FormValues = map[string]string{}
	}

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
