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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
	"github.com/manimovassagh/rampart/internal/store"
)

// mockConsoleStore implements AdminConsoleStore with configurable return values.
type mockConsoleStore struct {
	// Role methods
	roles           []*model.Role
	rolesTotal      int
	rolesErr        error
	roleByID        *model.Role
	roleByIDErr     error
	roleUsers       []*model.UserRoleAssignment
	roleCount       int
	createdRole     *model.Role
	createRoleErr   error
	updateRoleErr   error
	deleteRoleErr   error
	assignRoleErr   error
	unassignRoleErr error

	// User methods
	userByID       *model.User
	userByIDErr    error
	emailUser      *model.User
	usernameUser   *model.User
	createdUser    *model.User
	createUserErr  error
	updateUserErr  error
	deleteUserErr  error
	updatePwErr    error
	listUsers      []*model.User
	listUsersTotal int
	userRoles      []*model.Role
	userGroups     []*model.Group

	// Webhook methods
	webhooks         []*model.Webhook
	webhooksTotal    int
	webhooksErr      error
	webhookByID      *model.Webhook
	webhookByIDErr   error
	createdWebhook   *model.Webhook
	createWebhookErr error
	updateWebhookErr error
	deleteWebhookErr error
	deliveries       []*model.WebhookDelivery
	deliveriesTotal  int

	// Org methods
	org *model.Organization
}

// ── UserReader ──
func (m *mockConsoleStore) GetUserByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return m.userByID, m.userByIDErr
}
func (m *mockConsoleStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return m.emailUser, nil
}
func (m *mockConsoleStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return m.usernameUser, nil
}
func (m *mockConsoleStore) FindUserByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}

// ── UserWriter ──
func (m *mockConsoleStore) CreateUser(_ context.Context, u *model.User) (*model.User, error) {
	if m.createUserErr != nil {
		return nil, m.createUserErr
	}
	if m.createdUser != nil {
		return m.createdUser, nil
	}
	u.ID = uuid.New()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	return u, nil
}
func (m *mockConsoleStore) UpdateUser(_ context.Context, _, _ uuid.UUID, _ *model.UpdateUserRequest) (*model.User, error) {
	if m.updateUserErr != nil {
		return nil, m.updateUserErr
	}
	return m.userByID, nil
}
func (m *mockConsoleStore) DeleteUser(_ context.Context, _, _ uuid.UUID) error {
	return m.deleteUserErr
}
func (m *mockConsoleStore) UpdatePassword(_ context.Context, _, _ uuid.UUID, _ []byte) error {
	return m.updatePwErr
}
func (m *mockConsoleStore) UpdateLastLoginAt(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockConsoleStore) IncrementFailedLogins(_ context.Context, _ uuid.UUID, _ int, _ time.Duration) error {
	return nil
}
func (m *mockConsoleStore) ResetFailedLogins(_ context.Context, _ uuid.UUID) error { return nil }

// ── UserLister ──
func (m *mockConsoleStore) ListUsers(_ context.Context, _ uuid.UUID, _, _ string, _, _ int) ([]*model.User, int, error) {
	return m.listUsers, m.listUsersTotal, nil
}
func (m *mockConsoleStore) CountUsers(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (m *mockConsoleStore) CountRecentUsers(_ context.Context, _ uuid.UUID, _ int) (int, error) {
	return 0, nil
}

// ── OrgReader ──
func (m *mockConsoleStore) GetOrganizationByID(_ context.Context, _ uuid.UUID) (*model.Organization, error) {
	if m.org != nil {
		return m.org, nil
	}
	return &model.Organization{Name: "Test Org"}, nil
}
func (m *mockConsoleStore) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockConsoleStore) GetOrganizationIDBySlug(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.New(), nil
}

// ── OrgWriter ──
func (m *mockConsoleStore) CreateOrganization(_ context.Context, _ *model.CreateOrgRequest) (*model.Organization, error) {
	return nil, nil
}
func (m *mockConsoleStore) UpdateOrganization(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgRequest) (*model.Organization, error) {
	return nil, nil
}
func (m *mockConsoleStore) DeleteOrganization(_ context.Context, _ uuid.UUID) error { return nil }

// ── OrgLister ──
func (m *mockConsoleStore) ListOrganizations(_ context.Context, _ string, _, _ int) ([]*model.Organization, int, error) {
	return nil, 0, nil
}
func (m *mockConsoleStore) CountOrganizations(_ context.Context) (int, error) { return 0, nil }

// ── OrgSettingsReadWriter ──
func (m *mockConsoleStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return nil, nil
}
func (m *mockConsoleStore) UpdateOrgSettings(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	return nil, nil
}

// ── RoleReader ──
func (m *mockConsoleStore) GetRoleByID(_ context.Context, _ uuid.UUID) (*model.Role, error) {
	return m.roleByID, m.roleByIDErr
}
func (m *mockConsoleStore) GetUserRoles(_ context.Context, _ uuid.UUID) ([]*model.Role, error) {
	return m.userRoles, nil
}
func (m *mockConsoleStore) GetUserRoleNames(_ context.Context, _ uuid.UUID) ([]string, error) {
	return nil, nil
}
func (m *mockConsoleStore) GetRoleUsers(_ context.Context, _ uuid.UUID) ([]*model.UserRoleAssignment, error) {
	return m.roleUsers, nil
}
func (m *mockConsoleStore) CountRoleUsers(_ context.Context, _ uuid.UUID) (int, error) {
	return m.roleCount, nil
}
func (m *mockConsoleStore) CountRoles(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (m *mockConsoleStore) UserCountsByRole(_ context.Context, _ uuid.UUID) ([]model.RoleCount, error) {
	return nil, nil
}

// ── RoleWriter ──
func (m *mockConsoleStore) CreateRole(_ context.Context, role *model.Role) (*model.Role, error) {
	if m.createRoleErr != nil {
		return nil, m.createRoleErr
	}
	if m.createdRole != nil {
		return m.createdRole, nil
	}
	role.ID = uuid.New()
	return role, nil
}
func (m *mockConsoleStore) UpdateRole(_ context.Context, _, _ uuid.UUID, _ *model.UpdateRoleRequest) (*model.Role, error) {
	if m.updateRoleErr != nil {
		return nil, m.updateRoleErr
	}
	return m.roleByID, nil
}
func (m *mockConsoleStore) DeleteRole(_ context.Context, _, _ uuid.UUID) error {
	return m.deleteRoleErr
}
func (m *mockConsoleStore) AssignRole(_ context.Context, _, _ uuid.UUID) error {
	return m.assignRoleErr
}
func (m *mockConsoleStore) UnassignRole(_ context.Context, _, _ uuid.UUID) error {
	return m.unassignRoleErr
}

// ── RoleLister ──
func (m *mockConsoleStore) ListRoles(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*model.Role, int, error) {
	return m.roles, m.rolesTotal, m.rolesErr
}

// ── OAuthClient stubs ──
func (m *mockConsoleStore) GetOAuthClient(_ context.Context, _ string) (*model.OAuthClient, error) {
	return nil, nil
}
func (m *mockConsoleStore) CreateOAuthClient(_ context.Context, _ *model.OAuthClient) (*model.OAuthClient, error) {
	return nil, nil
}
func (m *mockConsoleStore) UpdateOAuthClient(_ context.Context, _ string, _ uuid.UUID, _ *model.UpdateClientRequest) (*model.OAuthClient, error) {
	return nil, nil
}
func (m *mockConsoleStore) DeleteOAuthClient(_ context.Context, _ string, _ uuid.UUID) error {
	return nil
}
func (m *mockConsoleStore) UpdateClientSecret(_ context.Context, _ string, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockConsoleStore) ListOAuthClients(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*model.OAuthClient, int, error) {
	return nil, 0, nil
}
func (m *mockConsoleStore) CountOAuthClients(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

// ── AuditStore ──
func (m *mockConsoleStore) CreateAuditEvent(_ context.Context, _ *model.AuditEvent) error {
	return nil
}
func (m *mockConsoleStore) ListAuditEvents(_ context.Context, _ uuid.UUID, _, _ string, _, _ int) ([]*model.AuditEvent, int, error) {
	return nil, 0, nil
}
func (m *mockConsoleStore) CountRecentEvents(_ context.Context, _ uuid.UUID, _ int) (int, error) {
	return 0, nil
}
func (m *mockConsoleStore) LoginCountsByDay(_ context.Context, _ uuid.UUID, _ int) ([]model.DayCount, error) {
	return nil, nil
}

// ── GroupReader stubs ──
func (m *mockConsoleStore) GetGroupByID(_ context.Context, _ uuid.UUID) (*model.Group, error) {
	return nil, nil
}
func (m *mockConsoleStore) GetGroupMembers(_ context.Context, _ uuid.UUID) ([]*model.GroupMember, error) {
	return nil, nil
}
func (m *mockConsoleStore) GetGroupRoles(_ context.Context, _ uuid.UUID) ([]*model.GroupRoleAssignment, error) {
	return nil, nil
}
func (m *mockConsoleStore) GetUserGroups(_ context.Context, _ uuid.UUID) ([]*model.Group, error) {
	return m.userGroups, nil
}
func (m *mockConsoleStore) GetEffectiveUserRoles(_ context.Context, _ uuid.UUID) ([]string, error) {
	return nil, nil
}
func (m *mockConsoleStore) CountGroupMembers(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockConsoleStore) CountGroupRoles(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockConsoleStore) CountGroups(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }

// ── GroupWriter stubs ──
func (m *mockConsoleStore) CreateGroup(_ context.Context, _ *model.Group) (*model.Group, error) {
	return nil, nil
}
func (m *mockConsoleStore) UpdateGroup(_ context.Context, _ uuid.UUID, _ *model.UpdateGroupRequest) (*model.Group, error) {
	return nil, nil
}
func (m *mockConsoleStore) DeleteGroup(_ context.Context, _ uuid.UUID) error       { return nil }
func (m *mockConsoleStore) AddUserToGroup(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockConsoleStore) RemoveUserFromGroup(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (m *mockConsoleStore) AssignRoleToGroup(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockConsoleStore) UnassignRoleFromGroup(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

// ── GroupLister ──
func (m *mockConsoleStore) ListGroups(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*model.Group, int, error) {
	return nil, 0, nil
}

// ── SocialProviderConfigStore ──
func (m *mockConsoleStore) UpsertSocialProviderConfig(_ context.Context, _ *model.SocialProviderConfig) error {
	return nil
}
func (m *mockConsoleStore) ListSocialProviderConfigs(_ context.Context, _ uuid.UUID) ([]*model.SocialProviderConfig, error) {
	return nil, nil
}
func (m *mockConsoleStore) DeleteSocialProviderConfig(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// ── WebhookReader ──
func (m *mockConsoleStore) GetWebhookByID(_ context.Context, _ uuid.UUID) (*model.Webhook, error) {
	return m.webhookByID, m.webhookByIDErr
}
func (m *mockConsoleStore) GetEnabledWebhooksForEvent(_ context.Context, _ uuid.UUID, _ string) ([]*model.Webhook, error) {
	return nil, nil
}

// ── WebhookWriter ──
func (m *mockConsoleStore) CreateWebhook(_ context.Context, w *model.Webhook) (*model.Webhook, error) {
	if m.createWebhookErr != nil {
		return nil, m.createWebhookErr
	}
	if m.createdWebhook != nil {
		return m.createdWebhook, nil
	}
	w.ID = uuid.New()
	return w, nil
}
func (m *mockConsoleStore) UpdateWebhook(_ context.Context, _ uuid.UUID, _ *model.UpdateWebhookRequest) (*model.Webhook, error) {
	if m.updateWebhookErr != nil {
		return nil, m.updateWebhookErr
	}
	return m.webhookByID, nil
}
func (m *mockConsoleStore) DeleteWebhook(_ context.Context, _ uuid.UUID) error {
	return m.deleteWebhookErr
}

// ── WebhookLister ──
func (m *mockConsoleStore) ListWebhooks(_ context.Context, _ uuid.UUID, _, _ int) ([]*model.Webhook, int, error) {
	return m.webhooks, m.webhooksTotal, m.webhooksErr
}

// ── WebhookDeliveryStore ──
func (m *mockConsoleStore) CreateWebhookDelivery(_ context.Context, _ *model.WebhookDelivery) error {
	return nil
}
func (m *mockConsoleStore) UpdateWebhookDelivery(_ context.Context, _ uuid.UUID, _ string, _ int, _ *int, _ string, _, _ *time.Time) error {
	return nil
}
func (m *mockConsoleStore) GetPendingDeliveries(_ context.Context, _ int) ([]*model.WebhookDelivery, error) {
	return nil, nil
}
func (m *mockConsoleStore) ListWebhookDeliveries(_ context.Context, _ uuid.UUID, _, _ int) ([]*model.WebhookDelivery, int, error) {
	return m.deliveries, m.deliveriesTotal, nil
}
func (m *mockConsoleStore) DeleteOldDeliveries(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}

// ── SAMLProviderStore ──
func (m *mockConsoleStore) CreateSAMLProvider(_ context.Context, _ *model.SAMLProvider) (*model.SAMLProvider, error) {
	return nil, nil
}
func (m *mockConsoleStore) GetSAMLProviderByID(_ context.Context, _ uuid.UUID) (*model.SAMLProvider, error) {
	return nil, nil
}
func (m *mockConsoleStore) ListSAMLProviders(_ context.Context, _ uuid.UUID) ([]*model.SAMLProvider, error) {
	return nil, nil
}
func (m *mockConsoleStore) GetEnabledSAMLProviders(_ context.Context, _ uuid.UUID) ([]*model.SAMLProvider, error) {
	return nil, nil
}
func (m *mockConsoleStore) UpdateSAMLProvider(_ context.Context, _ uuid.UUID, _ *model.UpdateSAMLProviderRequest) (*model.SAMLProvider, error) {
	return nil, nil
}
func (m *mockConsoleStore) DeleteSAMLProvider(_ context.Context, _ uuid.UUID) error { return nil }

// mockConsoleSessionStore implements AdminConsoleSessionStore for testing.
type mockConsoleSessionStore struct {
	sessions     []*session.Session
	countByUser  int
	countActive  int
	deleteErr    error
	deleteAllErr error
	allSessions  []*session.WithUser
	allTotal     int
	allErr       error
}

func (m *mockConsoleSessionStore) ListByUserID(_ context.Context, _ uuid.UUID) ([]*session.Session, error) {
	return m.sessions, nil
}
func (m *mockConsoleSessionStore) CountByUserID(_ context.Context, _ uuid.UUID) (int, error) {
	return m.countByUser, nil
}
func (m *mockConsoleSessionStore) CountActive(_ context.Context, _ uuid.UUID) (int, error) {
	return m.countActive, nil
}
func (m *mockConsoleSessionStore) DeleteByUserID(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}
func (m *mockConsoleSessionStore) Delete(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}
func (m *mockConsoleSessionStore) ListAll(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*session.WithUser, int, error) {
	return m.allSessions, m.allTotal, m.allErr
}
func (m *mockConsoleSessionStore) DeleteAll(_ context.Context, _ uuid.UUID) error {
	return m.deleteAllErr
}

// newTestConsoleHandler creates an AdminConsoleHandler without templates.
// Tests that trigger render() will panic, so we only test redirect-path handlers.
func newTestConsoleHandler(s *mockConsoleStore, sess *mockConsoleSessionStore) *AdminConsoleHandler {
	return &AdminConsoleHandler{
		store:    s,
		sessions: sess,
		logger:   slog.Default(),
		issuer:   "http://localhost:8080",
	}
}

func authContext(userID, orgID uuid.UUID) context.Context {
	return middleware.SetAuthenticatedUser(context.Background(), &middleware.AuthenticatedUser{
		UserID:            userID,
		OrgID:             orgID,
		PreferredUsername: "admin",
		Roles:             []string{"admin"},
	})
}

func formRequest(target string, values url.Values, userID, orgID uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(authContext(userID, orgID))
	return req
}

// ── Role Management Tests ──

func TestCreateRoleActionSuccess(t *testing.T) {
	orgID := uuid.New()
	ms := &mockConsoleStore{}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	form := url.Values{"name": {"editor"}, "description": {"Can edit content"}}
	req := formRequest("/admin/roles", form, uuid.New(), orgID)
	w := httptest.NewRecorder()

	h.CreateRoleAction(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminRoles {
		t.Errorf("redirect = %q, want %q", loc, pathAdminRoles)
	}
}

func TestCreateRoleActionDuplicate(t *testing.T) {
	orgID := uuid.New()
	ms := &mockConsoleStore{createRoleErr: store.ErrDuplicateKey}
	// Need templates for render path; use pages map with a no-op template
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})
	h.pages = minimalPages()

	form := url.Values{"name": {"admin"}, "description": {"dup"}}
	req := formRequest("/admin/roles", form, uuid.New(), orgID)
	w := httptest.NewRecorder()

	h.CreateRoleAction(w, req)

	// Should render the create page (200) with error, not redirect
	if w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) for duplicate role")
	}
}

func TestCreateRoleActionEmptyName(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})
	h.pages = minimalPages()

	form := url.Values{"name": {""}, "description": {"missing name"}}
	req := formRequest("/admin/roles", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.CreateRoleAction(w, req)

	// Should render the form with validation error, not redirect
	if w.Code == http.StatusFound {
		t.Fatal("expected render (not redirect) for empty role name")
	}
}

func TestDeleteRoleActionSuccess(t *testing.T) {
	orgID := uuid.New()
	roleID := uuid.New()
	ms := &mockConsoleStore{}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/roles/{id}/delete", h.DeleteRoleAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/roles/"+roleID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminRoles {
		t.Errorf("redirect = %q, want %q", loc, pathAdminRoles)
	}
}

func TestDeleteRoleActionBuiltinRole(t *testing.T) {
	orgID := uuid.New()
	roleID := uuid.New()
	ms := &mockConsoleStore{deleteRoleErr: store.ErrBuiltinRole}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/roles/{id}/delete", h.DeleteRoleAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/roles/"+roleID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	// Should redirect back to the role detail, not the list
	expected := fmt.Sprintf(pathAdminRoleFmt, roleID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestDeleteRoleActionInvalidUUID(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/roles/{id}/delete", h.DeleteRoleAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/roles/not-a-uuid/delete", http.NoBody)
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

// ── Role Assignment Tests ──

func TestAssignRoleActionSuccess(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	ms := &mockConsoleStore{}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}/roles", h.AssignRoleAction)

	form := url.Values{"role_id": {roleID.String()}}
	req := formRequest("/admin/users/"+userID.String()+"/roles", form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminUserFmt, userID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAssignRoleActionInvalidRoleID(t *testing.T) {
	userID := uuid.New()
	ms := &mockConsoleStore{}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}/roles", h.AssignRoleAction)

	form := url.Values{"role_id": {"not-a-uuid"}}
	req := formRequest("/admin/users/"+userID.String()+"/roles", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminUserFmt, userID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestAssignRoleActionStoreError(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	ms := &mockConsoleStore{assignRoleErr: fmt.Errorf("db error")}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}/roles", h.AssignRoleAction)

	form := url.Values{"role_id": {roleID.String()}}
	req := formRequest("/admin/users/"+userID.String()+"/roles", form, uuid.New(), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestUnassignRoleActionSuccess(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	ms := &mockConsoleStore{}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}/roles/{roleId}/delete", h.UnassignRoleAction)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/users/%s/roles/%s/delete", userID, roleID), http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminUserFmt, userID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

// ── User Management Tests ──

func TestDeleteUserActionSelfDeletion(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	ms := &mockConsoleStore{}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}/delete", h.DeleteUserAction)

	// Authenticated user tries to delete themselves
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+userID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(userID, orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminUserFmt, userID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestDeleteUserActionSuccess(t *testing.T) {
	userID := uuid.New()
	callerID := uuid.New()
	orgID := uuid.New()
	ms := &mockConsoleStore{}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}/delete", h.DeleteUserAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+userID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(callerID, orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminUsers {
		t.Errorf("redirect = %q, want %q", loc, pathAdminUsers)
	}
}

func TestDeleteUserActionStoreError(t *testing.T) {
	userID := uuid.New()
	callerID := uuid.New()
	orgID := uuid.New()
	ms := &mockConsoleStore{deleteUserErr: fmt.Errorf("db error")}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/users/{id}/delete", h.DeleteUserAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+userID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(callerID, orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	// On failure, redirects to user detail page
	expected := fmt.Sprintf(pathAdminUserFmt, userID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

// ── Webhook Management Tests ──

func TestDeleteWebhookActionSuccess(t *testing.T) {
	orgID := uuid.New()
	whID := uuid.New()
	ms := &mockConsoleStore{
		webhookByID: &model.Webhook{ID: whID, OrgID: orgID},
	}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/webhooks/{id}/delete", h.DeleteWebhookAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/webhooks/"+whID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminWebhooks {
		t.Errorf("redirect = %q, want %q", loc, pathAdminWebhooks)
	}
}

func TestDeleteWebhookActionCrossTenant(t *testing.T) {
	whID := uuid.New()
	ownerOrg := uuid.New()
	attackerOrg := uuid.New()
	ms := &mockConsoleStore{
		webhookByID: &model.Webhook{ID: whID, OrgID: ownerOrg},
	}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/webhooks/{id}/delete", h.DeleteWebhookAction)

	// Attacker from a different org tries to delete another org's webhook
	req := httptest.NewRequest(http.MethodPost, "/admin/webhooks/"+whID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), attackerOrg))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	// Should redirect to webhooks list (webhook "not found" for wrong org)
	if loc := w.Header().Get("Location"); loc != pathAdminWebhooks {
		t.Errorf("redirect = %q, want %q", loc, pathAdminWebhooks)
	}
}

func TestDeleteWebhookActionStoreError(t *testing.T) {
	orgID := uuid.New()
	whID := uuid.New()
	ms := &mockConsoleStore{
		webhookByID:      &model.Webhook{ID: whID, OrgID: orgID},
		deleteWebhookErr: fmt.Errorf("db error"),
	}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/webhooks/{id}/delete", h.DeleteWebhookAction)

	req := httptest.NewRequest(http.MethodPost, "/admin/webhooks/"+whID.String()+"/delete", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	// On failure, redirects to webhook detail page
	expected := fmt.Sprintf(pathAdminWebhookFmt, whID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

// ── Session Management Tests ──

func TestRevokeSessionActionSuccess(t *testing.T) {
	sessID := uuid.New()
	ms := &mockConsoleStore{}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

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

func TestRevokeAllSessionsActionSuccess(t *testing.T) {
	ms := &mockConsoleStore{}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/revoke-all", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.RevokeAllSessionsAction(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != pathAdminSessions {
		t.Errorf("redirect = %q, want %q", loc, pathAdminSessions)
	}
}

func TestRevokeAllSessionsActionError(t *testing.T) {
	ms := &mockConsoleStore{}
	sess := &mockConsoleSessionStore{deleteAllErr: fmt.Errorf("session store down")}
	h := newTestConsoleHandler(ms, sess)

	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/revoke-all", http.NoBody)
	req = req.WithContext(authContext(uuid.New(), uuid.New()))
	w := httptest.NewRecorder()

	h.RevokeAllSessionsAction(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	// Should still redirect to sessions page even on error
	if loc := w.Header().Get("Location"); loc != pathAdminSessions {
		t.Errorf("redirect = %q, want %q", loc, pathAdminSessions)
	}
}

// ── Update Role Tests ──

func TestUpdateRoleActionSuccess(t *testing.T) {
	orgID := uuid.New()
	roleID := uuid.New()
	ms := &mockConsoleStore{roleByID: &model.Role{ID: roleID, OrgID: orgID, Name: "editor"}}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/roles/{id}", h.UpdateRoleAction)

	form := url.Values{"name": {"updated-editor"}, "description": {"Updated desc"}}
	req := formRequest("/admin/roles/"+roleID.String(), form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminRoleFmt, roleID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

func TestUpdateRoleActionStoreError(t *testing.T) {
	orgID := uuid.New()
	roleID := uuid.New()
	ms := &mockConsoleStore{updateRoleErr: fmt.Errorf("db error")}
	h := newTestConsoleHandler(ms, &mockConsoleSessionStore{})

	r := chi.NewRouter()
	r.Post("/admin/roles/{id}", h.UpdateRoleAction)

	form := url.Values{"name": {"editor"}, "description": {"desc"}}
	req := formRequest("/admin/roles/"+roleID.String(), form, uuid.New(), orgID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	expected := fmt.Sprintf(pathAdminRoleFmt, roleID)
	if loc := w.Header().Get("Location"); loc != expected {
		t.Errorf("redirect = %q, want %q", loc, expected)
	}
}

// minimalPages returns a template map with stub templates that render without error.
// Used for tests that exercise render() paths (e.g., validation errors).
func minimalPages() map[string]*template.Template {
	stub := template.Must(template.New("base").Parse(`{{define "base"}}stub{{end}}`))
	pages := map[string]*template.Template{}
	for _, name := range []string{
		"role_create", "user_create", "webhook_create",
		"roles_list", "users_list", "webhooks_list",
		"role_detail", "user_detail", "webhook_detail",
	} {
		pages[name] = stub
	}
	return pages
}
