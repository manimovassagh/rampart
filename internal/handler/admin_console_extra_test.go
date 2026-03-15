package handler

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
	"github.com/manimovassagh/rampart/internal/store"
)

// extraMockStore extends mockConsoleStore with configurable fields for clients, groups, and sessions.
type extraMockStore struct {
	mockConsoleStore

	// OAuth client overrides
	oauthClient          *model.OAuthClient
	oauthClientErr       error
	oauthClients         []*model.OAuthClient
	oauthClientsTotal    int
	oauthClientsErr      error
	createdOAuthClient   *model.OAuthClient
	createOAuthClientErr error
	updateOAuthClientErr error
	deleteOAuthClientErr error
	updateSecretErr      error

	// Group overrides
	group                *model.Group
	groupErr             error
	groups               []*model.Group
	groupsTotal          int
	groupsErr            error
	createdGroup         *model.Group
	createGroupErr       error
	updateGroupErr       error
	deleteGroupErr       error
	addMemberErr         error
	removeMemberErr      error
	assignGroupRoleErr   error
	unassignGroupRoleErr error
	groupMembers         []*model.GroupMember
	groupRolesAssign     []*model.GroupRoleAssignment
}

func (m *extraMockStore) GetOAuthClient(_ context.Context, _ string) (*model.OAuthClient, error) {
	return m.oauthClient, m.oauthClientErr
}

func (m *extraMockStore) CreateOAuthClient(_ context.Context, c *model.OAuthClient) (*model.OAuthClient, error) {
	if m.createOAuthClientErr != nil {
		return nil, m.createOAuthClientErr
	}
	if m.createdOAuthClient != nil {
		return m.createdOAuthClient, nil
	}
	c.ID = uuid.New().String()
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	return c, nil
}

func (m *extraMockStore) ListOAuthClients(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*model.OAuthClient, int, error) {
	return m.oauthClients, m.oauthClientsTotal, m.oauthClientsErr
}

func (m *extraMockStore) UpdateOAuthClient(_ context.Context, _ string, _ uuid.UUID, _ *model.UpdateClientRequest) (*model.OAuthClient, error) {
	if m.updateOAuthClientErr != nil {
		return nil, m.updateOAuthClientErr
	}
	return m.oauthClient, nil
}

func (m *extraMockStore) DeleteOAuthClient(_ context.Context, _ string, _ uuid.UUID) error {
	return m.deleteOAuthClientErr
}

func (m *extraMockStore) UpdateClientSecret(_ context.Context, _ string, _ uuid.UUID, _ []byte) error {
	return m.updateSecretErr
}

func (m *extraMockStore) GetGroupByID(_ context.Context, _ uuid.UUID) (*model.Group, error) {
	return m.group, m.groupErr
}

func (m *extraMockStore) ListGroups(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*model.Group, int, error) {
	return m.groups, m.groupsTotal, m.groupsErr
}

func (m *extraMockStore) CreateGroup(_ context.Context, g *model.Group) (*model.Group, error) {
	if m.createGroupErr != nil {
		return nil, m.createGroupErr
	}
	if m.createdGroup != nil {
		return m.createdGroup, nil
	}
	g.ID = uuid.New()
	g.CreatedAt = time.Now()
	g.UpdatedAt = time.Now()
	return g, nil
}

func (m *extraMockStore) UpdateGroup(_ context.Context, _ uuid.UUID, _ *model.UpdateGroupRequest) (*model.Group, error) {
	if m.updateGroupErr != nil {
		return nil, m.updateGroupErr
	}
	return m.group, nil
}

func (m *extraMockStore) DeleteGroup(_ context.Context, _ uuid.UUID) error {
	return m.deleteGroupErr
}

func (m *extraMockStore) AddUserToGroup(_ context.Context, _, _ uuid.UUID) error {
	return m.addMemberErr
}

func (m *extraMockStore) RemoveUserFromGroup(_ context.Context, _, _ uuid.UUID) error {
	return m.removeMemberErr
}

func (m *extraMockStore) AssignRoleToGroup(_ context.Context, _, _ uuid.UUID) error {
	return m.assignGroupRoleErr
}

func (m *extraMockStore) UnassignRoleFromGroup(_ context.Context, _, _ uuid.UUID) error {
	return m.unassignGroupRoleErr
}

func (m *extraMockStore) GetGroupMembers(_ context.Context, _ uuid.UUID) ([]*model.GroupMember, error) {
	return m.groupMembers, nil
}

func (m *extraMockStore) GetGroupRoles(_ context.Context, _ uuid.UUID) ([]*model.GroupRoleAssignment, error) {
	return m.groupRolesAssign, nil
}

// newExtraConsoleHandler creates an AdminConsoleHandler with the extended mock and stub templates.
func newExtraConsoleHandler(s *extraMockStore, sess *mockConsoleSessionStore) *AdminConsoleHandler {
	h := &AdminConsoleHandler{
		store:    s,
		sessions: sess,
		logger:   noopLogger(),
		issuer:   "http://localhost:8080",
		pages:    extraMinimalPages(),
	}
	return h
}

// extraMinimalPages extends minimalPages with all the templates needed by client/group/role/session handlers.
func extraMinimalPages() map[string]*template.Template {
	stub := template.Must(template.New("base").Parse(`{{define "base"}}stub{{end}}`))
	stubPartial := template.Must(template.New("").Parse(
		`{{define "base"}}stub{{end}}` +
			`{{define "clients_table"}}table{{end}}` +
			`{{define "groups_table"}}table{{end}}` +
			`{{define "roles_table"}}table{{end}}` +
			`{{define "sessions_table"}}table{{end}}` +
			`{{define "user_search_results"}}results{{end}}`))
	pages := map[string]*template.Template{}
	for _, name := range []string{
		"client_create", "client_detail",
		"group_create", "group_detail",
		"role_create", "role_detail",
		"roles_list", "users_list",
		"user_create", "user_detail",
		"webhook_create", "webhook_detail", "webhooks_list",
	} {
		pages[name] = stub
	}
	// Pages that have partial rendering need both "base" and the partial block
	for _, name := range []string{
		"clients_list", "groups_list", "roles_list", "sessions_list",
	} {
		pages[name] = stubPartial
	}
	return pages
}

func extraFormRequest(target string, values url.Values, userID, orgID uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(authContext(userID, orgID))
	return req
}

// ══════════════════════════════════════════════════════════════════════════════
// Client handler tests
// ══════════════════════════════════════════════════════════════════════════════

func TestAdminConsoleExtraListClientsPageSuccess(t *testing.T) {
	orgID := uuid.New()
	client := &model.OAuthClient{
		ID: "client-1", OrgID: orgID, Name: "Test Client",
		ClientType: "public", Enabled: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	ms := &extraMockStore{
		oauthClients:      []*model.OAuthClient{client},
		oauthClientsTotal: 1,
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/clients", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()

	h.ListClientsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraListClientsPageStoreError(t *testing.T) {
	orgID := uuid.New()
	ms := &extraMockStore{oauthClientsErr: fmt.Errorf("db error")}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/clients", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()

	h.ListClientsPage(w, req)

	// Should still render (with error message), not panic
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (error rendered in page)", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraListClientsPageHTMX(t *testing.T) {
	orgID := uuid.New()
	ms := &extraMockStore{oauthClients: []*model.OAuthClient{}, oauthClientsTotal: 0}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/clients", http.NoBody)
	req.Header.Set("HX-Request", "true")
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()

	h.ListClientsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraCreateClientPageRenders(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/clients/new", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.CreateClientPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraCreateClientActionPublicSuccess(t *testing.T) {
	orgID := uuid.New()
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{
		"name":          {"My Public App"},
		"description":   {"A test app"},
		"client_type":   {"public"},
		"redirect_uris": {"http://localhost:3000/callback"},
	}
	req := extraFormRequest("/admin/clients", form, uuid.New(), orgID)
	w := httptest.NewRecorder()

	h.CreateClientAction(w, req)

	// Public client should redirect after creation
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraCreateClientActionConfidentialSuccess(t *testing.T) {
	orgID := uuid.New()
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{
		"name":          {"My Confidential App"},
		"description":   {"A test app"},
		"client_type":   {"confidential"},
		"redirect_uris": {"http://localhost:3000/callback"},
	}
	req := extraFormRequest("/admin/clients", form, uuid.New(), orgID)
	w := httptest.NewRecorder()

	h.CreateClientAction(w, req)

	// Confidential client should render the detail page with the secret (200)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (render with secret)", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraCreateClientActionEmptyName(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{
		"name":        {""},
		"client_type": {"public"},
	}
	req := extraFormRequest("/admin/clients", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateClientAction(w, req)

	// Should render form with validation error
	if w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) for empty client name")
	}
}

func TestAdminConsoleExtraCreateClientActionStoreError(t *testing.T) {
	ms := &extraMockStore{createOAuthClientErr: fmt.Errorf("db error")}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{
		"name":        {"App"},
		"client_type": {"public"},
	}
	req := extraFormRequest("/admin/clients", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateClientAction(w, req)

	// Should render error page, not redirect
	if w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) on store error")
	}
}

func TestAdminConsoleExtraClientDetailPageSuccess(t *testing.T) {
	orgID := uuid.New()
	clientID := uuid.New().String()
	ms := &extraMockStore{
		oauthClient: &model.OAuthClient{
			ID: clientID, OrgID: orgID, Name: "Test Client",
			ClientType: "public", Enabled: true,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/clients/{id}", h.ClientDetailPage)

	req := httptest.NewRequest(http.MethodGet, "/admin/clients/"+clientID, http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraClientDetailPageNotFound(t *testing.T) {
	orgID := uuid.New()
	ms := &extraMockStore{} // oauthClient is nil
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/clients/{id}", h.ClientDetailPage)

	req := httptest.NewRequest(http.MethodGet, "/admin/clients/some-id", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminClients {
		t.Errorf("redirect = %q, want %q", loc, pathAdminClients)
	}
}

func TestAdminConsoleExtraClientDetailPageEmptyID(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	// Call directly without chi router, so URLParam("id") returns ""
	req := httptest.NewRequest(http.MethodGet, "/admin/clients/", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.ClientDetailPage(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraClientDetailPageCrossTenant(t *testing.T) {
	ownerOrg := uuid.New()
	attackerOrg := uuid.New()
	clientID := uuid.New().String()
	ms := &extraMockStore{
		oauthClient: &model.OAuthClient{
			ID: clientID, OrgID: ownerOrg, Name: "Owner Client",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/clients/{id}", h.ClientDetailPage)

	req := httptest.NewRequest(http.MethodGet, "/admin/clients/"+clientID, http.NoBody)
	req = req.WithContext(authContext(uuid.New(), attackerOrg))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Different org should be treated as not found
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraUpdateClientActionSuccess(t *testing.T) {
	orgID := uuid.New()
	clientID := uuid.New().String()
	ms := &extraMockStore{
		oauthClient: &model.OAuthClient{
			ID: clientID, OrgID: orgID, Name: "Client",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/clients/{id}", h.UpdateClientAction)

	form := url.Values{"name": {"Updated"}, "description": {"Updated desc"}, "enabled": {"true"}}
	req := extraFormRequest("/admin/clients/"+clientID, form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminClientFmt, clientID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsoleExtraUpdateClientActionStoreError(t *testing.T) {
	orgID := uuid.New()
	clientID := uuid.New().String()
	ms := &extraMockStore{updateOAuthClientErr: fmt.Errorf("db error")}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/clients/{id}", h.UpdateClientAction)

	form := url.Values{"name": {"Updated"}}
	req := extraFormRequest("/admin/clients/"+clientID, form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraUpdateClientActionEmptyID(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodPost, "/admin/clients/", strings.NewReader("name=foo"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.UpdateClientAction(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminClients {
		t.Errorf("redirect = %q, want %q", loc, pathAdminClients)
	}
}

func TestAdminConsoleExtraDeleteClientActionSuccess(t *testing.T) {
	orgID := uuid.New()
	clientID := uuid.New().String()
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/clients/{id}/delete", h.DeleteClientAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/clients/"+clientID+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminClients {
		t.Errorf("redirect = %q, want %q", loc, pathAdminClients)
	}
}

func TestAdminConsoleExtraDeleteClientActionStoreError(t *testing.T) {
	orgID := uuid.New()
	clientID := uuid.New().String()
	ms := &extraMockStore{deleteOAuthClientErr: fmt.Errorf("db error")}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/clients/{id}/delete", h.DeleteClientAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/clients/"+clientID+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminClientFmt, clientID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsoleExtraDeleteClientActionEmptyID(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodPost, "/admin/clients//delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.DeleteClientAction(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraRegenerateSecretActionSuccess(t *testing.T) {
	orgID := uuid.New()
	clientID := uuid.New().String()
	ms := &extraMockStore{
		oauthClient: &model.OAuthClient{
			ID: clientID, OrgID: orgID, Name: "Confidential Client",
			ClientType: "confidential", Enabled: true,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/clients/{id}/regenerate-secret", h.RegenerateSecretAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/clients/"+clientID+"/regenerate-secret", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should render the detail page with the new secret
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraRegenerateSecretActionNotConfidential(t *testing.T) {
	orgID := uuid.New()
	clientID := uuid.New().String()
	ms := &extraMockStore{
		oauthClient: &model.OAuthClient{
			ID: clientID, OrgID: orgID, Name: "Public Client",
			ClientType: "public", Enabled: true,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/clients/{id}/regenerate-secret", h.RegenerateSecretAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/clients/"+clientID+"/regenerate-secret", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraRegenerateSecretActionClientNotFound(t *testing.T) {
	orgID := uuid.New()
	clientID := uuid.New().String()
	ms := &extraMockStore{} // oauthClient is nil
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/clients/{id}/regenerate-secret", h.RegenerateSecretAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/clients/"+clientID+"/regenerate-secret", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminClients {
		t.Errorf("redirect = %q, want %q", loc, pathAdminClients)
	}
}

func TestAdminConsoleExtraRegenerateSecretActionUpdateSecretError(t *testing.T) {
	orgID := uuid.New()
	clientID := uuid.New().String()
	ms := &extraMockStore{
		oauthClient: &model.OAuthClient{
			ID: clientID, OrgID: orgID, Name: "Confidential Client",
			ClientType: "confidential", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		updateSecretErr: fmt.Errorf("db error"),
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/clients/{id}/regenerate-secret", h.RegenerateSecretAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/clients/"+clientID+"/regenerate-secret", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Group handler tests
// ══════════════════════════════════════════════════════════════════════════════

func TestAdminConsoleExtraListGroupsPageSuccess(t *testing.T) {
	orgID := uuid.New()
	group := &model.Group{
		ID: uuid.New(), OrgID: orgID, Name: "Engineers",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	ms := &extraMockStore{groups: []*model.Group{group}, groupsTotal: 1}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/groups", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()

	h.ListGroupsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraListGroupsPageStoreError(t *testing.T) {
	ms := &extraMockStore{groupsErr: fmt.Errorf("db error")}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/groups", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.ListGroupsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (error rendered in page)", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraListGroupsPageHTMX(t *testing.T) {
	ms := &extraMockStore{groups: []*model.Group{}, groupsTotal: 0}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/groups", http.NoBody)
	req.Header.Set("HX-Request", "true")
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.ListGroupsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraCreateGroupPageRenders(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/groups/new", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.CreateGroupPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraCreateGroupActionSuccess(t *testing.T) {
	orgID := uuid.New()
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{"name": {"Engineers"}, "description": {"Engineering team"}}
	req := extraFormRequest("/admin/groups", form, uuid.New(), orgID)
	w := httptest.NewRecorder()

	h.CreateGroupAction(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminGroups {
		t.Errorf("redirect = %q, want %q", loc, pathAdminGroups)
	}
}

func TestAdminConsoleExtraCreateGroupActionEmptyName(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{"name": {""}, "description": {"missing name"}}
	req := extraFormRequest("/admin/groups", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateGroupAction(w, req)

	if w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) for empty group name")
	}
}

func TestAdminConsoleExtraCreateGroupActionDuplicate(t *testing.T) {
	ms := &extraMockStore{createGroupErr: store.ErrDuplicateKey}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{"name": {"Engineers"}, "description": {"dup"}}
	req := extraFormRequest("/admin/groups", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateGroupAction(w, req)

	if w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) for duplicate group")
	}
}

func TestAdminConsoleExtraCreateGroupActionStoreError(t *testing.T) {
	ms := &extraMockStore{createGroupErr: fmt.Errorf("db error")}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{"name": {"Engineers"}}
	req := extraFormRequest("/admin/groups", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateGroupAction(w, req)

	if w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) on store error")
	}
}

func TestAdminConsoleExtraGroupDetailPageSuccess(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	ms := &extraMockStore{
		group: &model.Group{
			ID: groupID, OrgID: orgID, Name: "Engineers",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/groups/{id}", h.GroupDetailPage)

	req := httptest.NewRequest(http.MethodGet, "/admin/groups/"+groupID.String(), http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraGroupDetailPageInvalidUUID(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/groups/{id}", h.GroupDetailPage)

	req := httptest.NewRequest(http.MethodGet, "/admin/groups/not-a-uuid", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminGroups {
		t.Errorf("redirect = %q, want %q", loc, pathAdminGroups)
	}
}

func TestAdminConsoleExtraGroupDetailPageNotFound(t *testing.T) {
	ms := &extraMockStore{} // group is nil
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/groups/{id}", h.GroupDetailPage)

	req := httptest.NewRequest(http.MethodGet, "/admin/groups/"+uuid.New().String(), http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraUpdateGroupActionSuccess(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	ms := &extraMockStore{
		group: &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}", h.UpdateGroupAction)

	form := url.Values{"name": {"Updated Engineers"}, "description": {"Updated desc"}}
	req := extraFormRequest("/admin/groups/"+groupID.String(), form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminGroupFmt, groupID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsoleExtraUpdateGroupActionNotFound(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	ms := &extraMockStore{} // group is nil
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}", h.UpdateGroupAction)

	form := url.Values{"name": {"Updated"}}
	req := extraFormRequest("/admin/groups/"+groupID.String(), form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminGroups {
		t.Errorf("redirect = %q, want %q", loc, pathAdminGroups)
	}
}

func TestAdminConsoleExtraUpdateGroupActionStoreError(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	ms := &extraMockStore{
		group:          &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
		updateGroupErr: fmt.Errorf("db error"),
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}", h.UpdateGroupAction)

	form := url.Values{"name": {"Updated"}}
	req := extraFormRequest("/admin/groups/"+groupID.String(), form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraUpdateGroupActionInvalidUUID(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}", h.UpdateGroupAction)

	form := url.Values{"name": {"Updated"}}
	req := extraFormRequest("/admin/groups/not-a-uuid", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraDeleteGroupActionSuccess(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	ms := &extraMockStore{
		group: &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/delete", h.DeleteGroupAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/groups/"+groupID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminGroups {
		t.Errorf("redirect = %q, want %q", loc, pathAdminGroups)
	}
}

func TestAdminConsoleExtraDeleteGroupActionStoreError(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	ms := &extraMockStore{
		group:          &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
		deleteGroupErr: fmt.Errorf("db error"),
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/delete", h.DeleteGroupAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/groups/"+groupID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminGroupFmt, groupID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsoleExtraDeleteGroupActionNotFound(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	ms := &extraMockStore{} // group is nil
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/delete", h.DeleteGroupAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/groups/"+groupID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminGroups {
		t.Errorf("redirect = %q, want %q", loc, pathAdminGroups)
	}
}

func TestAdminConsoleExtraAddGroupMemberActionSuccess(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	userID := uuid.New()
	ms := &extraMockStore{
		group: &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/members", h.AddGroupMemberAction)

	form := url.Values{"user_id": {userID.String()}}
	req := extraFormRequest("/admin/groups/"+groupID.String()+"/members", form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminGroupFmt, groupID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsoleExtraAddGroupMemberActionInvalidUser(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	ms := &extraMockStore{
		group: &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/members", h.AddGroupMemberAction)

	form := url.Values{"user_id": {"not-a-uuid"}}
	req := extraFormRequest("/admin/groups/"+groupID.String()+"/members", form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraAddGroupMemberActionStoreError(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	userID := uuid.New()
	ms := &extraMockStore{
		group:        &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
		addMemberErr: fmt.Errorf("db error"),
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/members", h.AddGroupMemberAction)

	form := url.Values{"user_id": {userID.String()}}
	req := extraFormRequest("/admin/groups/"+groupID.String()+"/members", form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraRemoveGroupMemberActionSuccess(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	userID := uuid.New()
	ms := &extraMockStore{
		group: &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/members/{userId}/delete", h.RemoveGroupMemberAction)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/groups/%s/members/%s/delete", groupID, userID), http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminGroupFmt, groupID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsoleExtraRemoveGroupMemberActionInvalidUserID(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	ms := &extraMockStore{
		group: &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/members/{userId}/delete", h.RemoveGroupMemberAction)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/groups/%s/members/not-a-uuid/delete", groupID), http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraAssignGroupRoleActionSuccess(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	roleID := uuid.New()
	ms := &extraMockStore{
		group: &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/roles", h.AssignGroupRoleAction)

	form := url.Values{"role_id": {roleID.String()}}
	req := extraFormRequest("/admin/groups/"+groupID.String()+"/roles", form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminGroupFmt, groupID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsoleExtraAssignGroupRoleActionInvalidRoleID(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	ms := &extraMockStore{
		group: &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/roles", h.AssignGroupRoleAction)

	form := url.Values{"role_id": {"not-a-uuid"}}
	req := extraFormRequest("/admin/groups/"+groupID.String()+"/roles", form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraAssignGroupRoleActionStoreError(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	roleID := uuid.New()
	ms := &extraMockStore{
		group:              &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
		assignGroupRoleErr: fmt.Errorf("db error"),
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/roles", h.AssignGroupRoleAction)

	form := url.Values{"role_id": {roleID.String()}}
	req := extraFormRequest("/admin/groups/"+groupID.String()+"/roles", form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraUnassignGroupRoleActionSuccess(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	roleID := uuid.New()
	ms := &extraMockStore{
		group: &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/roles/{roleId}/delete", h.UnassignGroupRoleAction)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/groups/%s/roles/%s/delete", groupID, roleID), http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminGroupFmt, groupID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAdminConsoleExtraUnassignGroupRoleActionInvalidRoleID(t *testing.T) {
	orgID := uuid.New()
	groupID := uuid.New()
	ms := &extraMockStore{
		group: &model.Group{ID: groupID, OrgID: orgID, Name: "Engineers"},
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/groups/{id}/roles/{roleId}/delete", h.UnassignGroupRoleAction)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/groups/%s/roles/not-a-uuid/delete", groupID), http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminConsoleExtraSearchUsersForGroupSuccess(t *testing.T) {
	orgID := uuid.New()
	ms := &extraMockStore{}
	ms.listUsers = []*model.User{
		{ID: uuid.New(), OrgID: orgID, Username: "alice", Email: "alice@test.com",
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	ms.listUsersTotal = 1
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/users/search?q=alice", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()

	h.SearchUsersForGroup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraSearchUsersForGroupEmptyQuery(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/users/search?q=", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.SearchUsersForGroup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("content-type = %q, want text/html", ct)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Role handler tests (covering ListRolesPage and RoleDetailPage which render)
// ══════════════════════════════════════════════════════════════════════════════

func TestAdminConsoleExtraListRolesPageSuccess(t *testing.T) {
	orgID := uuid.New()
	role := &model.Role{
		ID: uuid.New(), OrgID: orgID, Name: "editor",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	ms := &extraMockStore{}
	ms.roles = []*model.Role{role}
	ms.rolesTotal = 1
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/roles", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()

	h.ListRolesPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraListRolesPageStoreError(t *testing.T) {
	ms := &extraMockStore{}
	ms.rolesErr = fmt.Errorf("db error")
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/roles", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.ListRolesPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (error rendered in page)", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraListRolesPageHTMX(t *testing.T) {
	ms := &extraMockStore{}
	ms.roles = []*model.Role{}
	ms.rolesTotal = 0
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/roles", http.NoBody)
	req.Header.Set("HX-Request", "true")
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.ListRolesPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraCreateRolePageRenders(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodGet, "/admin/roles/new", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.CreateRolePage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraRoleDetailPageSuccess(t *testing.T) {
	orgID := uuid.New()
	roleID := uuid.New()
	ms := &extraMockStore{}
	ms.roleByID = &model.Role{
		ID: roleID, OrgID: orgID, Name: "editor",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/roles/{id}", h.RoleDetailPage)

	req := httptest.NewRequest(http.MethodGet, "/admin/roles/"+roleID.String(), http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraRoleDetailPageNotFound(t *testing.T) {
	ms := &extraMockStore{} // roleByID is nil
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/roles/{id}", h.RoleDetailPage)

	req := httptest.NewRequest(http.MethodGet, "/admin/roles/"+uuid.New().String(), http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminRoles {
		t.Errorf("redirect = %q, want %q", loc, pathAdminRoles)
	}
}

func TestAdminConsoleExtraRoleDetailPageInvalidUUID(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Get("/admin/roles/{id}", h.RoleDetailPage)

	req := httptest.NewRequest(http.MethodGet, "/admin/roles/not-a-uuid", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Session handler tests (covering ListSessionsPage which renders)
// ══════════════════════════════════════════════════════════════════════════════

func TestAdminConsoleExtraListSessionsPageSuccess(t *testing.T) {
	orgID := uuid.New()
	sess := &mockConsoleSessionStore{
		allSessions: []*session.WithUser{},
		allTotal:    0,
	}
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, sess)

	req := httptest.NewRequest(http.MethodGet, "/admin/sessions", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()

	h.ListSessionsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraListSessionsPageStoreError(t *testing.T) {
	sess := &mockConsoleSessionStore{allErr: fmt.Errorf("session store down")}
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, sess)

	req := httptest.NewRequest(http.MethodGet, "/admin/sessions", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.ListSessionsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (error rendered in page)", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraListSessionsPageHTMX(t *testing.T) {
	sess := &mockConsoleSessionStore{allSessions: []*session.WithUser{}, allTotal: 0}
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, sess)

	req := httptest.NewRequest(http.MethodGet, "/admin/sessions", http.NoBody)
	req.Header.Set("HX-Request", "true")
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.ListSessionsPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminConsoleExtraRevokeSessionActionInvalidUUID(t *testing.T) {
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/sessions/{id}/delete", h.RevokeSessionAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/not-a-uuid/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminSessions {
		t.Errorf("redirect = %q, want %q", loc, pathAdminSessions)
	}
}

func TestAdminConsoleExtraRevokeSessionActionDeleteError(t *testing.T) {
	sessID := uuid.New()
	sess := &mockConsoleSessionStore{deleteErr: fmt.Errorf("session store down")}
	ms := &extraMockStore{}
	h := newExtraConsoleHandler(ms, sess)

	r := chi.NewRouter()
	r.Post("/admin/sessions/{id}/delete", h.RevokeSessionAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/"+sessID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminSessions {
		t.Errorf("redirect = %q, want %q", loc, pathAdminSessions)
	}
}
