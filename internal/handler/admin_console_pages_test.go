package handler

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/plugin"
	"github.com/manimovassagh/rampart/internal/social"
	"github.com/manimovassagh/rampart/internal/store"
)

// allPages returns a template map with stubs for every page used by the handlers under test.
func allPages() map[string]*template.Template {
	stub := template.Must(template.New("base").Parse(`{{define "base"}}stub{{end}}`))
	// Also add a block-level template for HTMX partials
	stubPartial := template.Must(template.New("base").Parse(
		`{{define "base"}}stub{{end}}` +
			`{{define "events_table"}}events{{end}}` +
			`{{define "users_table"}}users{{end}}` +
			`{{define "orgs_table"}}orgs{{end}}` +
			`{{define "webhooks_table"}}webhooks{{end}}`,
	))
	names := []string{
		"events_list", "compliance", "plugins_list",
		"oidc", "social_providers",
		"orgs_list", "org_create", "org_detail", "org_import",
		"webhooks_list", "webhook_create", "webhook_detail",
		"saml_providers_list", "saml_provider_create", "saml_provider_detail",
		"users_list", "user_create", "user_detail",
		"role_create", "roles_list", "role_detail",
	}
	pages := make(map[string]*template.Template, len(names))
	for _, n := range names {
		switch n {
		case "events_list", "users_list", "orgs_list", "webhooks_list":
			pages[n] = stubPartial
		default:
			pages[n] = stub
		}
	}
	return pages
}

// newFullConsoleHandler builds an AdminConsoleHandler with all dependencies mocked.
func newFullConsoleHandler(ms *mockConsoleStore, sess *mockConsoleSessionStore) *AdminConsoleHandler {
	h := &AdminConsoleHandler{
		store:          ms,
		sessions:       sess,
		logger:         slog.Default(),
		issuer:         "http://localhost:8080",
		pages:          allPages(),
		socialRegistry: social.NewRegistry(),
		plugins:        plugin.NewRegistry(slog.Default()),
	}
	return h
}

func getRequest(target string, userID, orgID uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, http.NoBody)
	req = req.WithContext(authContext(userID, orgID))
	return req
}

// ── Events Page ──

func TestAdminConsolePagesListEventsPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/events", uuid.New(), uuid.New())

	h.ListEventsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListEventsPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesListEventsPageWithSearch(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/events?search=login&event_type=user.login", uuid.New(), uuid.New())

	h.ListEventsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListEventsPage with search status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesListEventsPageHTMX(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/events", uuid.New(), uuid.New())
	req.Header.Set(headerHXRequest, formValueTrue)

	h.ListEventsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListEventsPage HTMX status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ── Compliance Page ──

func TestAdminConsolePagesCompliancePage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/compliance", uuid.New(), uuid.New())

	h.CompliancePage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("CompliancePage status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ── Plugins Page ──

func TestAdminConsolePagesPluginsPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/plugins", uuid.New(), uuid.New())

	h.PluginsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PluginsPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ── OIDC Page ──

func TestAdminConsolePagesOIDCPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/oidc", uuid.New(), uuid.New())

	h.OIDCPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("OIDCPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ── Social Providers Page ──

func TestAdminConsolePagesSocialProvidersPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/social", uuid.New(), uuid.New())

	h.SocialProvidersPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("SocialProvidersPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ── Update Social Provider Action ──

func TestAdminConsolePagesUpdateSocialProviderSuccess(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/social/{provider}", h.UpdateSocialProviderAction)

	form := url.Values{
		"client_id":     {"test-client-id"},
		"client_secret": {"test-secret"},
		"enabled":       {"on"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/social/google", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("UpdateSocialProviderAction status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "/admin/social") {
		t.Errorf("redirect = %q, want /admin/social", loc)
	}
}

// ── Organizations Pages ──

func TestAdminConsolePagesListOrgsPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/organizations", uuid.New(), uuid.New())

	h.ListOrgsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListOrgsPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesListOrgsPageHTMX(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/organizations", uuid.New(), uuid.New())
	req.Header.Set(headerHXRequest, formValueTrue)

	h.ListOrgsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListOrgsPage HTMX status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesCreateOrgPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/organizations/new", uuid.New(), uuid.New())

	h.CreateOrgPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("CreateOrgPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesCreateOrgActionSuccess(t *testing.T) {
	org := &model.Organization{ID: uuid.New(), Name: "Test Org", Slug: "test-org"}
	ms := &mockConsoleStore{}
	// Override CreateOrganization to return our org
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	// We need the mock to return a created org, so let's use a custom store
	customStore := &mockOrgCreateStore{mockConsoleStore: ms, createdOrg: org}
	h.store = customStore

	form := url.Values{"name": {"Test Org"}, "slug": {"test-org"}, "display_name": {"Test"}}
	req := formRequest("/admin/organizations", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateOrgAction(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("CreateOrgAction status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminOrgs {
		t.Errorf("redirect = %q, want %q", loc, pathAdminOrgs)
	}
}

func TestAdminConsolePagesCreateOrgActionEmptyName(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{"name": {""}, "slug": {""}, "display_name": {""}}
	req := formRequest("/admin/organizations", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateOrgAction(w, req)

	// Should render form with validation errors (200, not redirect)
	if w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) for empty org name")
	}
}

func TestAdminConsolePagesCreateOrgActionDuplicate(t *testing.T) {
	ms := &mockConsoleStore{}
	customStore := &mockOrgCreateStore{mockConsoleStore: ms, createErr: store.ErrDuplicateKey}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	h.store = customStore

	form := url.Values{"name": {"Test Org"}, "slug": {"test-org"}, "display_name": {"Test"}}
	req := formRequest("/admin/organizations", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateOrgAction(w, req)

	// Should render form with duplicate error (200, not redirect)
	if w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) for duplicate org slug")
	}
}

func TestAdminConsolePagesOrgDetailPage(t *testing.T) {
	orgID := uuid.New()
	ms := &mockConsoleStore{
		org: &model.Organization{ID: orgID, Name: "Test Org"},
	}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/organizations/{id}", h.OrgDetailPage)

	req := getRequest("/admin/organizations/"+orgID.String(), uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("OrgDetailPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesOrgDetailPageInvalidID(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/organizations/{id}", h.OrgDetailPage)

	req := getRequest("/admin/organizations/not-a-uuid", uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("OrgDetailPage invalid ID status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsolePagesUpdateOrgActionSuccess(t *testing.T) {
	orgID := uuid.New()
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/organizations/{id}", h.UpdateOrgAction)

	form := url.Values{"name": {"Updated Org"}, "display_name": {"Updated"}, "enabled": {"true"}}
	req := formRequest("/admin/organizations/"+orgID.String(), form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("UpdateOrgAction status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminOrgFmt, orgID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsolePagesUpdateOrgSettingsAction(t *testing.T) {
	orgID := uuid.New()
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/organizations/{id}/settings", h.UpdateOrgSettingsAction)

	form := url.Values{
		"password_min_length":         {"8"},
		"access_token_ttl_seconds":    {"3600"},
		"refresh_token_ttl_seconds":   {"86400"},
		"mfa_enforcement":             {"off"},
		"primary_color":               {"#FF5500"},
		"background_color":            {"#FFFFFF"},
		"self_registration_enabled":   {"true"},
		"email_verification_required": {"true"},
	}
	req := formRequest("/admin/organizations/"+orgID.String()+"/settings", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("UpdateOrgSettingsAction status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsolePagesDeleteOrgActionSuccess(t *testing.T) {
	orgID := uuid.New()
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/organizations/{id}/delete", h.DeleteOrgAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/organizations/"+orgID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("DeleteOrgAction status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminOrgs {
		t.Errorf("redirect = %q, want %q", loc, pathAdminOrgs)
	}
}

func TestAdminConsolePagesDeleteOrgActionDefaultOrg(t *testing.T) {
	orgID := uuid.New()
	// Override DeleteOrganization to return ErrDefaultOrg
	ms := &mockConsoleStore{}
	customStore := &mockOrgDeleteStore{mockConsoleStore: ms, deleteErr: store.ErrDefaultOrg}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	h.store = customStore

	r := chi.NewRouter()
	r.Post("/admin/organizations/{id}/delete", h.DeleteOrgAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/organizations/"+orgID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("DeleteOrgAction default org status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminOrgFmt, orgID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

// ── Export/Import ──

func TestAdminConsolePagesExportOrgAction(t *testing.T) {
	orgID := uuid.New()
	ms := &mockConsoleStore{
		org: &model.Organization{ID: orgID, Name: "Test Org", Slug: "test-org"},
	}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/organizations/{id}/export", h.ExportOrgAction)

	req := getRequest("/admin/organizations/"+orgID.String()+"/export", uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ExportOrgAction status = %d, want %d", w.Code, http.StatusOK)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "json") {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition = %q, want attachment", cd)
	}
}

func TestAdminConsolePagesExportOrgActionInvalidID(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/organizations/{id}/export", h.ExportOrgAction)

	req := getRequest("/admin/organizations/not-a-uuid/export", uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("ExportOrgAction invalid ID status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsolePagesImportOrgPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/organizations/import", uuid.New(), uuid.New())

	h.ImportOrgPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ImportOrgPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ── Webhooks Pages ──

func TestAdminConsolePagesListWebhooksPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/webhooks", uuid.New(), uuid.New())

	h.ListWebhooksPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListWebhooksPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesListWebhooksPageHTMX(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/webhooks", uuid.New(), uuid.New())
	req.Header.Set(headerHXRequest, formValueTrue)

	h.ListWebhooksPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListWebhooksPage HTMX status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesCreateWebhookPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/webhooks/new", uuid.New(), uuid.New())

	h.CreateWebhookPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("CreateWebhookPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesCreateWebhookActionSuccess(t *testing.T) {
	whID := uuid.New()
	ms := &mockConsoleStore{
		createdWebhook: &model.Webhook{ID: whID, URL: "https://example.com/hook"},
	}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{
		"url":         {"https://example.com/hook"},
		"description": {"Test webhook"},
		"event_types": {"user.created,user.updated"},
	}
	req := formRequest("/admin/webhooks", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateWebhookAction(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("CreateWebhookAction status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "/admin/webhooks/") {
		t.Errorf("redirect = %q, want webhook detail page", loc)
	}
}

func TestAdminConsolePagesCreateWebhookActionEmptyURL(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{"url": {""}, "description": {"test"}, "event_types": {"user.created"}}
	req := formRequest("/admin/webhooks", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateWebhookAction(w, req)

	// Should render form with validation error (200, not redirect)
	if w.Code == http.StatusSeeOther || w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) for empty URL")
	}
}

func TestAdminConsolePagesCreateWebhookActionEmptyEvents(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{"url": {"https://example.com/hook"}, "description": {"test"}, "event_types": {""}}
	req := formRequest("/admin/webhooks", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateWebhookAction(w, req)

	if w.Code == http.StatusSeeOther || w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) for empty event types")
	}
}

func TestAdminConsolePagesWebhookDetailPage(t *testing.T) {
	orgID := uuid.New()
	whID := uuid.New()
	ms := &mockConsoleStore{
		webhookByID: &model.Webhook{ID: whID, OrgID: orgID, Description: "Test WH"},
	}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/webhooks/{id}", h.WebhookDetailPage)

	req := getRequest("/admin/webhooks/"+whID.String(), uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("WebhookDetailPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesWebhookDetailPageNotFound(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/webhooks/{id}", h.WebhookDetailPage)

	req := getRequest("/admin/webhooks/"+uuid.New().String(), uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("WebhookDetailPage not found status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

func TestAdminConsolePagesUpdateWebhookActionSuccess(t *testing.T) {
	orgID := uuid.New()
	whID := uuid.New()
	ms := &mockConsoleStore{
		webhookByID: &model.Webhook{ID: whID, OrgID: orgID},
	}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/webhooks/{id}", h.UpdateWebhookAction)

	form := url.Values{
		"url":         {"https://example.com/updated"},
		"description": {"Updated"},
		"event_types": {"user.created"},
		"enabled":     {"true"},
	}
	req := formRequest("/admin/webhooks/"+whID.String(), form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("UpdateWebhookAction status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	expected := fmt.Sprintf(pathAdminWebhookFmt, whID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsolePagesUpdateWebhookActionInvalidID(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/webhooks/{id}", h.UpdateWebhookAction)

	form := url.Values{"url": {"https://example.com/hook"}}
	req := formRequest("/admin/webhooks/not-a-uuid", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("UpdateWebhookAction invalid ID status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

// ── SAML Providers Pages ──

func TestAdminConsolePagesListSAMLProvidersPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/saml-providers", uuid.New(), uuid.New())

	h.ListSAMLProvidersPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListSAMLProvidersPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesCreateSAMLProviderPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/saml-providers/new", uuid.New(), uuid.New())

	h.CreateSAMLProviderPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("CreateSAMLProviderPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesCreateSAMLProviderActionSuccess(t *testing.T) {
	samlID := uuid.New()
	ms := &mockConsoleStore{}
	customStore := &mockSAMLCreateStore{
		mockConsoleStore: ms,
		created:          &model.SAMLProvider{ID: samlID, Name: "Okta"},
	}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	h.store = customStore

	form := url.Values{
		"name":        {"Okta"},
		"entity_id":   {"https://okta.example.com"},
		"sso_url":     {"https://okta.example.com/sso"},
		"certificate": {"-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----"},
	}
	req := formRequest("/admin/saml-providers", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateSAMLProviderAction(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("CreateSAMLProviderAction status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	loc := w.Header().Get("Location")
	expected := fmt.Sprintf(pathAdminSAMLFmt, samlID)
	if loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsolePagesCreateSAMLProviderActionValidation(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	// Missing all required fields
	form := url.Values{"name": {""}, "entity_id": {""}, "sso_url": {""}, "certificate": {""}}
	req := formRequest("/admin/saml-providers", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateSAMLProviderAction(w, req)

	// Should render form with errors (200, not redirect)
	if w.Code == http.StatusSeeOther || w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) for empty SAML fields")
	}
}

func TestAdminConsolePagesSAMLProviderDetailPageNotFound(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/saml-providers/{id}", h.SAMLProviderDetailPage)

	req := getRequest("/admin/saml-providers/"+uuid.New().String(), uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Provider not found -> redirect
	if w.Code != http.StatusSeeOther {
		t.Fatalf("SAMLProviderDetailPage not found status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

func TestAdminConsolePagesSAMLProviderDetailPageInvalidID(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/saml-providers/{id}", h.SAMLProviderDetailPage)

	req := getRequest("/admin/saml-providers/not-a-uuid", uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("SAMLProviderDetailPage invalid ID status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

func TestAdminConsolePagesUpdateSAMLProviderActionSuccess(t *testing.T) {
	samlID := uuid.New()
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/saml-providers/{id}", h.UpdateSAMLProviderAction)

	form := url.Values{
		"name":        {"Updated Okta"},
		"entity_id":   {"https://okta.example.com"},
		"sso_url":     {"https://okta.example.com/sso"},
		"certificate": {"cert-data"},
		"enabled":     {"true"},
	}
	req := formRequest("/admin/saml-providers/"+samlID.String(), form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("UpdateSAMLProviderAction status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	expected := fmt.Sprintf(pathAdminSAMLFmt, samlID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsolePagesUpdateSAMLProviderActionInvalidID(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/saml-providers/{id}", h.UpdateSAMLProviderAction)

	form := url.Values{"name": {"test"}}
	req := formRequest("/admin/saml-providers/not-a-uuid", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("UpdateSAMLProviderAction invalid ID status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

func TestAdminConsolePagesDeleteSAMLProviderActionSuccess(t *testing.T) {
	samlID := uuid.New()
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/saml-providers/{id}/delete", h.DeleteSAMLProviderAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/saml-providers/"+samlID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("DeleteSAMLProviderAction status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminSAMLProviders {
		t.Errorf("redirect = %q, want %q", loc, pathAdminSAMLProviders)
	}
}

func TestAdminConsolePagesDeleteSAMLProviderActionInvalidID(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/saml-providers/{id}/delete", h.DeleteSAMLProviderAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/saml-providers/not-a-uuid/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("DeleteSAMLProviderAction invalid ID status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

// ── Users Pages ──

func TestAdminConsolePagesListUsersPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/users", uuid.New(), uuid.New())

	h.ListUsersPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListUsersPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesListUsersPageHTMX(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/users?search=john&status=active", uuid.New(), uuid.New())
	req.Header.Set(headerHXRequest, formValueTrue)

	h.ListUsersPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListUsersPage HTMX status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesCreateUserPage(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})
	w := httptest.NewRecorder()
	req := getRequest("/admin/users/new", uuid.New(), uuid.New())

	h.CreateUserPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("CreateUserPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesCreateUserActionValidationErrors(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	// Missing all required fields
	form := url.Values{"username": {""}, "email": {""}, "password": {""}}
	req := formRequest("/admin/users", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateUserAction(w, req)

	// Should render form with errors (200, not redirect)
	if w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) for invalid user data")
	}
}

func TestAdminConsolePagesUserDetailPage(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	ms := &mockConsoleStore{
		userByID: &model.User{ID: userID, OrgID: orgID, Username: "testuser", Email: "test@example.com"},
	}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/users/{id}", h.UserDetailPage)

	req := getRequest("/admin/users/"+userID.String(), uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("UserDetailPage status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsolePagesUserDetailPageInvalidID(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/users/{id}", h.UserDetailPage)

	req := getRequest("/admin/users/not-a-uuid", uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("UserDetailPage invalid ID status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsolePagesUserDetailPageNotFound(t *testing.T) {
	ms := &mockConsoleStore{userByIDErr: fmt.Errorf("not found")}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/users/{id}", h.UserDetailPage)

	req := getRequest("/admin/users/"+uuid.New().String(), uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("UserDetailPage not found status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsolePagesUpdateUserActionSuccess(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	ms := &mockConsoleStore{
		userByID: &model.User{ID: userID, OrgID: orgID, Username: "updated"},
	}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}", h.UpdateUserAction)

	form := url.Values{
		"username":    {"updated"},
		"email":       {"updated@example.com"},
		"given_name":  {"Updated"},
		"family_name": {"User"},
		"enabled":     {"true"},
	}
	req := formRequest("/admin/users/"+userID.String(), form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("UpdateUserAction status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminUserFmt, userID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsolePagesUpdateUserActionInvalidID(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}", h.UpdateUserAction)

	form := url.Values{"username": {"test"}}
	req := formRequest("/admin/users/not-a-uuid", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("UpdateUserAction invalid ID status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsolePagesResetPasswordActionSuccess(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}/reset-password", h.ResetPasswordAction)

	form := url.Values{"password": {"StrongP@ssw0rd123"}}
	req := formRequest("/admin/users/"+userID.String()+"/reset-password", form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("ResetPasswordAction status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminUserFmt, userID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsolePagesResetPasswordActionWeakPassword(t *testing.T) {
	userID := uuid.New()
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}/reset-password", h.ResetPasswordAction)

	form := url.Values{"password": {"short"}}
	req := formRequest("/admin/users/"+userID.String()+"/reset-password", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("ResetPasswordAction weak password status = %d, want %d", w.Code, http.StatusFound)
	}
	// Should redirect back to user page
	expected := fmt.Sprintf(pathAdminUserFmt, userID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsolePagesResetPasswordActionInvalidID(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}/reset-password", h.ResetPasswordAction)

	form := url.Values{"password": {"StrongP@ssw0rd123"}}
	req := formRequest("/admin/users/not-a-uuid/reset-password", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("ResetPasswordAction invalid ID status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsolePagesRevokeSessionsActionSuccess(t *testing.T) {
	userID := uuid.New()
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}/revoke-sessions", h.RevokeSessionsAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+userID.String()+"/revoke-sessions", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("RevokeSessionsAction status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminUserFmt, userID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsolePagesRevokeSessionsActionInvalidID(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newFullConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}/revoke-sessions", h.RevokeSessionsAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/users/not-a-uuid/revoke-sessions", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("RevokeSessionsAction invalid ID status = %d, want %d", w.Code, http.StatusFound)
	}
}

// ── Helper mock store types for overriding specific methods ──

// mockOrgCreateStore overrides CreateOrganization.
type mockOrgCreateStore struct {
	*mockConsoleStore
	createdOrg *model.Organization
	createErr  error
}

func (m *mockOrgCreateStore) CreateOrganization(_ context.Context, _ *model.CreateOrgRequest) (*model.Organization, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createdOrg, nil
}

// mockOrgDeleteStore overrides DeleteOrganization.
type mockOrgDeleteStore struct {
	*mockConsoleStore
	deleteErr error
}

func (m *mockOrgDeleteStore) DeleteOrganization(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

// mockSAMLCreateStore overrides CreateSAMLProvider.
type mockSAMLCreateStore struct {
	*mockConsoleStore
	created   *model.SAMLProvider
	createErr error
}

func (m *mockSAMLCreateStore) CreateSAMLProvider(_ context.Context, _ *model.SAMLProvider) (*model.SAMLProvider, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.created, nil
}
