package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	storeErrors "github.com/manimovassagh/rampart/internal/store"
)

type mockComplianceStore struct {
	auditEvents    []*model.AuditEvent
	auditTotal     int
	auditErr       error
	recentCount    int
	recentCountErr error
	loginCounts    []model.DayCount
	loginCountsErr error
	createAuditErr error
	userCount      int
	userCountErr   error
	org            *model.Organization
	orgErr         error
	settings       *model.OrgSettings
	settingsErr    error
}

func (m *mockComplianceStore) CreateAuditEvent(_ context.Context, _ *model.AuditEvent) error {
	return m.createAuditErr
}
func (m *mockComplianceStore) ListAuditEvents(_ context.Context, _ uuid.UUID, _, _ string, _, _ int) ([]*model.AuditEvent, int, error) {
	return m.auditEvents, m.auditTotal, m.auditErr
}
func (m *mockComplianceStore) CountRecentEvents(_ context.Context, _ uuid.UUID, _ int) (int, error) {
	return m.recentCount, m.recentCountErr
}
func (m *mockComplianceStore) LoginCountsByDay(_ context.Context, _ uuid.UUID, _ int) ([]model.DayCount, error) {
	return m.loginCounts, m.loginCountsErr
}
func (m *mockComplianceStore) ListUsers(_ context.Context, _ uuid.UUID, _, _ string, _, _ int) ([]*model.User, int, error) {
	return nil, 0, nil
}
func (m *mockComplianceStore) CountUsers(_ context.Context, _ uuid.UUID) (int, error) {
	return m.userCount, m.userCountErr
}
func (m *mockComplianceStore) CountRecentUsers(_ context.Context, _ uuid.UUID, _ int) (int, error) {
	return 0, nil
}
func (m *mockComplianceStore) GetOrganizationByID(_ context.Context, _ uuid.UUID) (*model.Organization, error) {
	return m.org, m.orgErr
}
func (m *mockComplianceStore) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockComplianceStore) GetOrganizationIDBySlug(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockComplianceStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.settings, m.settingsErr
}
func (m *mockComplianceStore) UpdateOrgSettings(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	return nil, nil
}

type mockSCIMStore struct {
	user         *model.User
	userErr      error
	users        []*model.User
	usersTotal   int
	usersErr     error
	createUser   *model.User
	createErr    error
	updateUser   *model.User
	updateErr    error
	deleteErr    error
	group        *model.Group
	groupErr     error
	groups       []*model.Group
	groupsTotal  int
	groupsErr    error
	createGroup  *model.Group
	createGrpErr error
	updateGroup  *model.Group
	updateGrpErr error
	deleteGrpErr error
	members      []*model.GroupMember
	addErr       error
	removeErr    error
	org          *model.Organization
	orgErr       error
}

func (m *mockSCIMStore) GetUserByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return m.user, m.userErr
}
func (m *mockSCIMStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockSCIMStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockSCIMStore) FindUserByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}
func (m *mockSCIMStore) CreateUser(_ context.Context, u *model.User) (*model.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createUser != nil {
		return m.createUser, nil
	}
	u.ID = uuid.New()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	return u, nil
}
func (m *mockSCIMStore) UpdateUser(_ context.Context, _, _ uuid.UUID, _ *model.UpdateUserRequest) (*model.User, error) {
	return m.updateUser, m.updateErr
}
func (m *mockSCIMStore) DeleteUser(_ context.Context, _, _ uuid.UUID) error { return m.deleteErr }
func (m *mockSCIMStore) UpdatePassword(_ context.Context, _, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockSCIMStore) UpdateLastLoginAt(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockSCIMStore) IncrementFailedLogins(_ context.Context, _ uuid.UUID, _ int, _ time.Duration) error {
	return nil
}
func (m *mockSCIMStore) ResetFailedLogins(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockSCIMStore) ListUsers(_ context.Context, _ uuid.UUID, _, _ string, _, _ int) ([]*model.User, int, error) {
	return m.users, m.usersTotal, m.usersErr
}
func (m *mockSCIMStore) CountUsers(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (m *mockSCIMStore) CountRecentUsers(_ context.Context, _ uuid.UUID, _ int) (int, error) {
	return 0, nil
}
func (m *mockSCIMStore) GetGroupByID(_ context.Context, _ uuid.UUID) (*model.Group, error) {
	return m.group, m.groupErr
}
func (m *mockSCIMStore) GetGroupMembers(_ context.Context, _ uuid.UUID) ([]*model.GroupMember, error) {
	return m.members, nil
}
func (m *mockSCIMStore) GetGroupRoles(_ context.Context, _ uuid.UUID) ([]*model.GroupRoleAssignment, error) {
	return nil, nil
}
func (m *mockSCIMStore) GetUserGroups(_ context.Context, _ uuid.UUID) ([]*model.Group, error) {
	return nil, nil
}
func (m *mockSCIMStore) GetEffectiveUserRoles(_ context.Context, _ uuid.UUID) ([]string, error) {
	return nil, nil
}
func (m *mockSCIMStore) CountGroupMembers(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (m *mockSCIMStore) CountGroupRoles(_ context.Context, _ uuid.UUID) (int, error)   { return 0, nil }
func (m *mockSCIMStore) CountGroups(_ context.Context, _ uuid.UUID) (int, error)       { return 0, nil }
func (m *mockSCIMStore) CreateGroup(_ context.Context, g *model.Group) (*model.Group, error) {
	if m.createGrpErr != nil {
		return nil, m.createGrpErr
	}
	if m.createGroup != nil {
		return m.createGroup, nil
	}
	g.ID = uuid.New()
	g.CreatedAt = time.Now()
	g.UpdatedAt = time.Now()
	return g, nil
}
func (m *mockSCIMStore) UpdateGroup(_ context.Context, _ uuid.UUID, _ *model.UpdateGroupRequest) (*model.Group, error) {
	return m.updateGroup, m.updateGrpErr
}
func (m *mockSCIMStore) DeleteGroup(_ context.Context, _ uuid.UUID) error { return m.deleteGrpErr }
func (m *mockSCIMStore) AddUserToGroup(_ context.Context, _, _ uuid.UUID) error {
	return m.addErr
}
func (m *mockSCIMStore) RemoveUserFromGroup(_ context.Context, _, _ uuid.UUID) error {
	return m.removeErr
}
func (m *mockSCIMStore) AssignRoleToGroup(_ context.Context, _, _ uuid.UUID) error     { return nil }
func (m *mockSCIMStore) UnassignRoleFromGroup(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockSCIMStore) ListGroups(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*model.Group, int, error) {
	return m.groups, m.groupsTotal, m.groupsErr
}
func (m *mockSCIMStore) GetOrganizationByID(_ context.Context, _ uuid.UUID) (*model.Organization, error) {
	return m.org, m.orgErr
}
func (m *mockSCIMStore) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockSCIMStore) GetOrganizationIDBySlug(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func newTestSCIMUser() *model.User {
	return &model.User{ID: uuid.New(), OrgID: uuid.New(), Username: "testuser", Email: "test@example.com", EmailVerified: true, GivenName: "Test", FamilyName: "User", Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func newTestGroup() *model.Group {
	return &model.Group{ID: uuid.New(), OrgID: uuid.New(), Name: "test-group", Description: "Test Group", CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func complianceAuthCtx(orgID uuid.UUID) context.Context {
	return middleware.SetAuthenticatedUser(context.Background(), &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: orgID, Roles: []string{"admin"}})
}

func scimCtx(orgID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), contextKey("scim_org_id"), orgID)
}

// Compliance tests

func TestMiscHandlers_ComplianceSOC2(t *testing.T) {
	orgID := uuid.New()
	s := &mockComplianceStore{recentCount: 42, userCount: 15, settings: &model.OrgSettings{OrgID: orgID, PasswordMinLength: 10, AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 7 * 24 * time.Hour}}
	h := NewComplianceHandler(s, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/soc2", http.NoBody)
	req = req.WithContext(complianceAuthCtx(orgID))
	w := httptest.NewRecorder()
	h.SOC2Report(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var rpt complianceReport
	_ = json.NewDecoder(w.Body).Decode(&rpt)
	if rpt.Framework != "SOC2" {
		t.Errorf("framework = %q, want SOC2", rpt.Framework)
	}
	if len(rpt.Checks) != 5 {
		t.Errorf("checks = %d, want 5", len(rpt.Checks))
	}
}

func TestMiscHandlers_ComplianceSOC2Warnings(t *testing.T) {
	orgID := uuid.New()
	s := &mockComplianceStore{recentCount: 0, settingsErr: fmt.Errorf("no settings"), userCount: 150}
	h := NewComplianceHandler(s, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/soc2", http.NoBody)
	req = req.WithContext(complianceAuthCtx(orgID))
	w := httptest.NewRecorder()
	h.SOC2Report(w, req)
	var rpt complianceReport
	_ = json.NewDecoder(w.Body).Decode(&rpt)
	if rpt.Summary["warning"] == 0 {
		t.Error("expected warnings")
	}
}

func TestMiscHandlers_ComplianceGDPR(t *testing.T) {
	orgID := uuid.New()
	s := &mockComplianceStore{recentCount: 10, userCount: 5, settings: &model.OrgSettings{OrgID: orgID, PasswordMinLength: 8, AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 7 * 24 * time.Hour}}
	h := NewComplianceHandler(s, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/gdpr", http.NoBody)
	req = req.WithContext(complianceAuthCtx(orgID))
	w := httptest.NewRecorder()
	h.GDPRReport(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var rpt complianceReport
	_ = json.NewDecoder(w.Body).Decode(&rpt)
	if rpt.Framework != "GDPR" {
		t.Errorf("framework = %q, want GDPR", rpt.Framework)
	}
	if len(rpt.Checks) != 5 {
		t.Errorf("checks = %d, want 5", len(rpt.Checks))
	}
}

func TestMiscHandlers_ComplianceHIPAA(t *testing.T) {
	orgID := uuid.New()
	s := &mockComplianceStore{recentCount: 10, userCount: 5, settings: &model.OrgSettings{OrgID: orgID, PasswordMinLength: 8, AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 7 * 24 * time.Hour}}
	h := NewComplianceHandler(s, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/hipaa", http.NoBody)
	req = req.WithContext(complianceAuthCtx(orgID))
	w := httptest.NewRecorder()
	h.HIPAAReport(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var rpt complianceReport
	_ = json.NewDecoder(w.Body).Decode(&rpt)
	if rpt.Framework != "HIPAA" {
		t.Errorf("framework = %q, want HIPAA", rpt.Framework)
	}
	if len(rpt.Checks) != 7 {
		t.Errorf("checks = %d, want 7", len(rpt.Checks))
	}
}

func TestMiscHandlers_ComplianceHIPAAWeakSettings(t *testing.T) {
	orgID := uuid.New()
	s := &mockComplianceStore{recentCount: 10, userCount: 5, settings: &model.OrgSettings{OrgID: orgID, PasswordMinLength: 4, AccessTokenTTL: 48 * time.Hour, RefreshTokenTTL: 7 * 24 * time.Hour}}
	h := NewComplianceHandler(s, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/hipaa", http.NoBody)
	req = req.WithContext(complianceAuthCtx(orgID))
	w := httptest.NewRecorder()
	h.HIPAAReport(w, req)
	var rpt complianceReport
	_ = json.NewDecoder(w.Body).Decode(&rpt)
	if rpt.Summary["warning"] < 2 {
		t.Errorf("expected >= 2 warnings, got %d", rpt.Summary["warning"])
	}
}

func TestMiscHandlers_ComplianceExportJSON(t *testing.T) {
	orgID := uuid.New()
	ev := []*model.AuditEvent{{ID: uuid.New(), OrgID: orgID, EventType: "user.login", ActorName: "admin", TargetType: "user", TargetName: "u", IPAddress: "127.0.0.1", CreatedAt: time.Now()}}
	h := NewComplianceHandler(&mockComplianceStore{auditEvents: ev, auditTotal: 1}, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/audit-export?format=json", http.NoBody)
	req = req.WithContext(complianceAuthCtx(orgID))
	w := httptest.NewRecorder()
	h.ExportAuditTrail(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, "audit-export.json") {
		t.Errorf("Content-Disposition = %q", cd)
	}
}

func TestMiscHandlers_ComplianceExportCSV(t *testing.T) {
	orgID := uuid.New()
	ev := []*model.AuditEvent{{ID: uuid.New(), OrgID: orgID, EventType: "user.login", ActorName: "admin", TargetType: "user", TargetName: "u", IPAddress: "127.0.0.1", CreatedAt: time.Now()}}
	h := NewComplianceHandler(&mockComplianceStore{auditEvents: ev, auditTotal: 1}, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/audit-export?format=csv", http.NoBody)
	req = req.WithContext(complianceAuthCtx(orgID))
	w := httptest.NewRecorder()
	h.ExportAuditTrail(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("Content-Type = %q", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "id,event_type,actor_name") {
		t.Error("CSV header missing")
	}
	if !strings.Contains(body, "user.login") {
		t.Error("CSV data missing")
	}
}

func TestMiscHandlers_ComplianceExportDefault(t *testing.T) {
	h := NewComplianceHandler(&mockComplianceStore{auditEvents: []*model.AuditEvent{}}, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/audit-export", http.NoBody)
	req = req.WithContext(complianceAuthCtx(uuid.New()))
	w := httptest.NewRecorder()
	h.ExportAuditTrail(w, req)
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestMiscHandlers_ComplianceExportStoreErr(t *testing.T) {
	h := NewComplianceHandler(&mockComplianceStore{auditErr: fmt.Errorf("db")}, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/audit-export", http.NoBody)
	req = req.WithContext(complianceAuthCtx(uuid.New()))
	w := httptest.NewRecorder()
	h.ExportAuditTrail(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// SCIM tests

func TestMiscHandlers_SCIMSPConfig(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	w := httptest.NewRecorder()
	h.ServiceProviderConfig(w, httptest.NewRequest(http.MethodGet, "/scim/v2/ServiceProviderConfig", http.NoBody))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != scimMediaType {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestMiscHandlers_SCIMResTypes(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	w := httptest.NewRecorder()
	h.ResourceTypes(w, httptest.NewRequest(http.MethodGet, "/scim/v2/ResourceTypes", http.NoBody))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMSchemas(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	w := httptest.NewRecorder()
	h.Schemas(w, httptest.NewRequest(http.MethodGet, "/scim/v2/Schemas", http.NoBody))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMListUsers(t *testing.T) {
	orgID := uuid.New()
	u := newTestSCIMUser()
	u.OrgID = orgID
	h := NewSCIMHandler(&mockSCIMStore{users: []*model.User{u}, usersTotal: 1}, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", http.NoBody)
	req = req.WithContext(scimCtx(orgID))
	w := httptest.NewRecorder()
	h.ListUsers(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMListUsersErr(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{usersErr: fmt.Errorf("db")}, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", http.NoBody)
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	h.ListUsers(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMGetUser(t *testing.T) {
	orgID := uuid.New()
	u := newTestSCIMUser()
	u.OrgID = orgID
	h := NewSCIMHandler(&mockSCIMStore{user: u}, noopLogger())
	r := chi.NewRouter()
	r.Get("/scim/v2/Users/{id}", h.GetUser)
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users/"+u.ID.String(), http.NoBody)
	req = req.WithContext(scimCtx(orgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMGetUserNotFound(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{user: nil}, noopLogger())
	r := chi.NewRouter()
	r.Get("/scim/v2/Users/{id}", h.GetUser)
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users/"+uuid.New().String(), http.NoBody)
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMGetUserBadID(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	r := chi.NewRouter()
	r.Get("/scim/v2/Users/{id}", h.GetUser)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/scim/v2/Users/bad", http.NoBody))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMGetUserWrongOrg(t *testing.T) {
	u := newTestSCIMUser()
	h := NewSCIMHandler(&mockSCIMStore{user: u}, noopLogger())
	r := chi.NewRouter()
	r.Get("/scim/v2/Users/{id}", h.GetUser)
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users/"+u.ID.String(), http.NoBody)
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMCreateUser(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	body := `{"userName":"new","name":{"givenName":"N","familyName":"U"},"emails":[{"value":"n@e.com"}],"active":true}`
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", strings.NewReader(body))
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	h.CreateUser(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMCreateUserDup(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{createErr: storeErrors.ErrDuplicateKey}, noopLogger())
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", strings.NewReader(`{"userName":"d","name":{},"active":true}`))
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	h.CreateUser(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMCreateUserBadJSON(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", strings.NewReader("bad"))
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	h.CreateUser(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMCreateUserStoreErr(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{createErr: fmt.Errorf("db")}, noopLogger())
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", strings.NewReader(`{"userName":"n","name":{},"active":true}`))
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	h.CreateUser(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMUpdateUser(t *testing.T) {
	u := newTestSCIMUser()
	h := NewSCIMHandler(&mockSCIMStore{updateUser: u}, noopLogger())
	r := chi.NewRouter()
	r.Put("/scim/v2/Users/{id}", h.UpdateUser)
	req := httptest.NewRequest(http.MethodPut, "/scim/v2/Users/"+u.ID.String(), strings.NewReader(`{"userName":"upd","name":{},"active":true}`))
	req = req.WithContext(scimCtx(u.OrgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMUpdateUserBadID(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	r := chi.NewRouter()
	r.Put("/scim/v2/Users/{id}", h.UpdateUser)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/scim/v2/Users/bad", strings.NewReader("{}")))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMUpdateUserStoreErr(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{updateErr: fmt.Errorf("db")}, noopLogger())
	r := chi.NewRouter()
	r.Put("/scim/v2/Users/{id}", h.UpdateUser)
	req := httptest.NewRequest(http.MethodPut, "/scim/v2/Users/"+uuid.New().String(), strings.NewReader(`{"userName":"u","name":{},"active":true}`))
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMPatchUser(t *testing.T) {
	u := newTestSCIMUser()
	h := NewSCIMHandler(&mockSCIMStore{updateUser: u}, noopLogger())
	r := chi.NewRouter()
	r.Patch("/scim/v2/Users/{id}", h.PatchUser)
	req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/"+u.ID.String(), strings.NewReader(`{"schemas":[],"Operations":[{"op":"replace","path":"active","value":false}]}`))
	req = req.WithContext(scimCtx(u.OrgID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMPatchUserBadID(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	r := chi.NewRouter()
	r.Patch("/scim/v2/Users/{id}", h.PatchUser)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/bad", strings.NewReader("{}")))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMDeleteUser(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	r := chi.NewRouter()
	r.Delete("/scim/v2/Users/{id}", h.DeleteUser)
	req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/"+uuid.New().String(), http.NoBody)
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMDeleteUserBadID(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	r := chi.NewRouter()
	r.Delete("/scim/v2/Users/{id}", h.DeleteUser)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/bad", http.NoBody))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMDeleteUserStoreErr(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{deleteErr: fmt.Errorf("db")}, noopLogger())
	r := chi.NewRouter()
	r.Delete("/scim/v2/Users/{id}", h.DeleteUser)
	req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/"+uuid.New().String(), http.NoBody)
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMListGroups(t *testing.T) {
	orgID := uuid.New()
	g := newTestGroup()
	g.OrgID = orgID
	h := NewSCIMHandler(&mockSCIMStore{groups: []*model.Group{g}, groupsTotal: 1}, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups", http.NoBody)
	req = req.WithContext(scimCtx(orgID))
	w := httptest.NewRecorder()
	h.ListGroups(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMListGroupsErr(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{groupsErr: fmt.Errorf("db")}, noopLogger())
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups", http.NoBody)
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	h.ListGroups(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMGetGroup(t *testing.T) {
	g := newTestGroup()
	h := NewSCIMHandler(&mockSCIMStore{group: g}, noopLogger())
	r := chi.NewRouter()
	r.Get("/scim/v2/Groups/{id}", h.GetGroup)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Groups/"+g.ID.String(), http.NoBody)
	req = req.WithContext(scimCtx(g.OrgID))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMGetGroupNotFound(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{group: nil}, noopLogger())
	r := chi.NewRouter()
	r.Get("/scim/v2/Groups/{id}", h.GetGroup)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/scim/v2/Groups/"+uuid.New().String(), http.NoBody))
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMGetGroupBadID(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	r := chi.NewRouter()
	r.Get("/scim/v2/Groups/{id}", h.GetGroup)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/scim/v2/Groups/bad", http.NoBody))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMCreateGroup(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", strings.NewReader(`{"displayName":"Eng"}`))
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	h.CreateGroup(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMCreateGroupDup(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{createGrpErr: storeErrors.ErrDuplicateKey}, noopLogger())
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", strings.NewReader(`{"displayName":"x"}`))
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	h.CreateGroup(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMCreateGroupStoreErr(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{createGrpErr: fmt.Errorf("db")}, noopLogger())
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", strings.NewReader(`{"displayName":"x"}`))
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	h.CreateGroup(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMCreateGroupBadJSON(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", strings.NewReader("bad"))
	req = req.WithContext(scimCtx(uuid.New()))
	w := httptest.NewRecorder()
	h.CreateGroup(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMUpdateGroup(t *testing.T) {
	g := newTestGroup()
	h := NewSCIMHandler(&mockSCIMStore{group: g, updateGroup: g}, noopLogger())
	r := chi.NewRouter()
	r.Put("/scim/v2/Groups/{id}", h.UpdateGroup)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/scim/v2/Groups/"+g.ID.String(), strings.NewReader(`{"displayName":"Up"}`))
	req = req.WithContext(scimCtx(g.OrgID))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMUpdateGroupBadID(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	r := chi.NewRouter()
	r.Put("/scim/v2/Groups/{id}", h.UpdateGroup)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/scim/v2/Groups/bad", strings.NewReader("{}")))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMUpdateGroupStoreErr(t *testing.T) {
	g := newTestGroup()
	h := NewSCIMHandler(&mockSCIMStore{group: g, updateGrpErr: fmt.Errorf("db")}, noopLogger())
	r := chi.NewRouter()
	r.Put("/scim/v2/Groups/{id}", h.UpdateGroup)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/scim/v2/Groups/"+g.ID.String(), strings.NewReader(`{"displayName":"x"}`))
	req = req.WithContext(scimCtx(g.OrgID))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMPatchGroupAdd(t *testing.T) {
	g := newTestGroup()
	h := NewSCIMHandler(&mockSCIMStore{group: g}, noopLogger())
	r := chi.NewRouter()
	r.Patch("/scim/v2/Groups/{id}", h.PatchGroup)
	body := fmt.Sprintf(`{"schemas":[],"Operations":[{"op":"add","path":"members","value":[{"value":%q}]}]}`, uuid.New().String())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/"+g.ID.String(), strings.NewReader(body))
	req = req.WithContext(scimCtx(g.OrgID))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMPatchGroupRemove(t *testing.T) {
	g := newTestGroup()
	h := NewSCIMHandler(&mockSCIMStore{group: g}, noopLogger())
	r := chi.NewRouter()
	r.Patch("/scim/v2/Groups/{id}", h.PatchGroup)
	body := fmt.Sprintf(`{"schemas":[],"Operations":[{"op":"remove","path":"members[value eq \"%s\"]"}]}`, uuid.New().String())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/"+g.ID.String(), strings.NewReader(body))
	req = req.WithContext(scimCtx(g.OrgID))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMPatchGroupReplace(t *testing.T) {
	g := newTestGroup()
	h := NewSCIMHandler(&mockSCIMStore{group: g, updateGroup: g}, noopLogger())
	r := chi.NewRouter()
	r.Patch("/scim/v2/Groups/{id}", h.PatchGroup)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/"+g.ID.String(), strings.NewReader(`{"schemas":[],"Operations":[{"op":"replace","path":"displayName","value":"R"}]}`))
	req = req.WithContext(scimCtx(g.OrgID))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMPatchGroupBadID(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	r := chi.NewRouter()
	r.Patch("/scim/v2/Groups/{id}", h.PatchGroup)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/bad", strings.NewReader("{}")))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMPatchGroupNotFound(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{group: nil}, noopLogger())
	r := chi.NewRouter()
	r.Patch("/scim/v2/Groups/{id}", h.PatchGroup)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/"+uuid.New().String(), strings.NewReader(`{"schemas":[],"Operations":[]}`)))
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMDeleteGroup(t *testing.T) {
	g := newTestGroup()
	h := NewSCIMHandler(&mockSCIMStore{group: g}, noopLogger())
	r := chi.NewRouter()
	r.Delete("/scim/v2/Groups/{id}", h.DeleteGroup)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Groups/"+g.ID.String(), http.NoBody)
	req = req.WithContext(scimCtx(g.OrgID))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMDeleteGroupBadID(t *testing.T) {
	h := NewSCIMHandler(&mockSCIMStore{}, noopLogger())
	r := chi.NewRouter()
	r.Delete("/scim/v2/Groups/{id}", h.DeleteGroup)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/scim/v2/Groups/bad", http.NoBody))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_SCIMDeleteGroupStoreErr(t *testing.T) {
	g := newTestGroup()
	h := NewSCIMHandler(&mockSCIMStore{group: g, deleteGrpErr: fmt.Errorf("db")}, noopLogger())
	r := chi.NewRouter()
	r.Delete("/scim/v2/Groups/{id}", h.DeleteGroup)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Groups/"+g.ID.String(), http.NoBody)
	req = req.WithContext(scimCtx(g.OrgID))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

// Organization additional error paths

func TestMiscHandlers_OrgListStoreErr(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{orgsErr: fmt.Errorf("db")}, &mockOrgSettingsStore{})
	w := httptest.NewRecorder()
	h.ListOrgs(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations", http.NoBody))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgListPaginationClamp(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{orgs: []*model.Organization{}, orgsTotal: 0}, &mockOrgSettingsStore{})
	w := httptest.NewRecorder()
	h.ListOrgs(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations?page=0&limit=999", http.NoBody))
	var resp model.ListOrgsResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Limit != maxPageLimit {
		t.Errorf("limit = %d, want %d", resp.Limit, maxPageLimit)
	}
	if resp.Page != 1 {
		t.Errorf("page = %d, want 1", resp.Page)
	}
}

func TestMiscHandlers_OrgCreateBadJSON(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{})
	w := httptest.NewRecorder()
	h.CreateOrg(w, httptest.NewRequest(http.MethodPost, "/api/v1/admin/organizations", strings.NewReader("bad")))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgCreateStoreErr(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{createErr: fmt.Errorf("db")}, &mockOrgSettingsStore{})
	w := httptest.NewRecorder()
	h.CreateOrg(w, httptest.NewRequest(http.MethodPost, "/api/v1/admin/organizations", bytes.NewReader([]byte(`{"name":"a","slug":"a"}`))))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgGetStoreErr(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{orgErr: fmt.Errorf("db")}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Get("/o/{id}", h.GetOrg)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/o/"+uuid.New().String(), http.NoBody))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgGetBadUUID(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Get("/o/{id}", h.GetOrg)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/o/bad", http.NoBody))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgUpdateBadJSON(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Put("/o/{id}", h.UpdateOrg)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/o/"+uuid.New().String(), strings.NewReader("bad")))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgUpdateNotFound(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{updatedOrg: nil}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Put("/o/{id}", h.UpdateOrg)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/o/"+uuid.New().String(), bytes.NewReader([]byte(`{"name":"up"}`))))
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgUpdateStoreErr(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{updateErr: fmt.Errorf("db")}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Put("/o/{id}", h.UpdateOrg)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/o/"+uuid.New().String(), bytes.NewReader([]byte(`{"name":"up"}`))))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgDeleteNotFound(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{deleteErr: storeErrors.ErrNotFound}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Delete("/o/{id}", h.DeleteOrg)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/o/"+uuid.New().String(), http.NoBody))
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgDeleteStoreErr(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{deleteErr: fmt.Errorf("db")}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Delete("/o/{id}", h.DeleteOrg)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/o/"+uuid.New().String(), http.NoBody))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgSettingsGetNotFound(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{settings: nil})
	r := chi.NewRouter()
	r.Get("/o/{id}/s", h.GetOrgSettings)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/o/"+uuid.New().String()+"/s", http.NoBody))
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgSettingsGetStoreErr(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{settingsErr: fmt.Errorf("db")})
	r := chi.NewRouter()
	r.Get("/o/{id}/s", h.GetOrgSettings)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/o/"+uuid.New().String()+"/s", http.NoBody))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgSettingsUpdateBadJSON(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Put("/o/{id}/s", h.UpdateOrgSettings)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/o/"+uuid.New().String()+"/s", strings.NewReader("bad")))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgSettingsUpdatePwdShort(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Put("/o/{id}/s", h.UpdateOrgSettings)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/o/"+uuid.New().String()+"/s", strings.NewReader(`{"password_min_length":0,"mfa_enforcement":"off","access_token_ttl_seconds":900,"refresh_token_ttl_seconds":86400}`)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgSettingsUpdateAccessTTLShort(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Put("/o/{id}/s", h.UpdateOrgSettings)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/o/"+uuid.New().String()+"/s", strings.NewReader(`{"password_min_length":8,"mfa_enforcement":"optional","access_token_ttl_seconds":10,"refresh_token_ttl_seconds":86400}`)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgSettingsUpdateRefreshTTLShort(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Put("/o/{id}/s", h.UpdateOrgSettings)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/o/"+uuid.New().String()+"/s", strings.NewReader(`{"password_min_length":8,"mfa_enforcement":"off","access_token_ttl_seconds":900,"refresh_token_ttl_seconds":10}`)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgSettingsUpdateBadColor(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Put("/o/{id}/s", h.UpdateOrgSettings)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/o/"+uuid.New().String()+"/s", strings.NewReader(`{"password_min_length":8,"mfa_enforcement":"off","access_token_ttl_seconds":900,"refresh_token_ttl_seconds":86400,"primary_color":"bad"}`)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgSettingsUpdateBadBgColor(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{})
	r := chi.NewRouter()
	r.Put("/o/{id}/s", h.UpdateOrgSettings)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/o/"+uuid.New().String()+"/s", strings.NewReader(`{"password_min_length":8,"mfa_enforcement":"off","access_token_ttl_seconds":900,"refresh_token_ttl_seconds":86400,"primary_color":"","background_color":"bad"}`)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgSettingsUpdateStoreErr(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{updateErr: fmt.Errorf("db")})
	r := chi.NewRouter()
	r.Put("/o/{id}/s", h.UpdateOrgSettings)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/o/"+uuid.New().String()+"/s", strings.NewReader(`{"password_min_length":8,"mfa_enforcement":"off","access_token_ttl_seconds":900,"refresh_token_ttl_seconds":86400,"primary_color":"","background_color":""}`)))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", w.Code)
	}
}

func TestMiscHandlers_OrgSettingsUpdateNotFound(t *testing.T) {
	h := newTestOrgHandler(&mockOrgStore{}, &mockOrgSettingsStore{updated: nil})
	r := chi.NewRouter()
	r.Put("/o/{id}/s", h.UpdateOrgSettings)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/o/"+uuid.New().String()+"/s", strings.NewReader(`{"password_min_length":8,"mfa_enforcement":"off","access_token_ttl_seconds":900,"refresh_token_ttl_seconds":86400,"primary_color":"","background_color":""}`)))
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d", w.Code)
	}
}

// Helper function tests

func TestMiscHandlers_ParseFilter(t *testing.T) {
	cases := []struct{ in, want string }{{"", ""}, {`userName eq "john"`, "john"}, {"no quotes", ""}}
	for _, c := range cases {
		if got := parseFilter(c.in); got != c.want {
			t.Errorf("parseFilter(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMiscHandlers_ParsePaginationDefaults(t *testing.T) {
	si, cnt := parsePagination(httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	if si != 1 || cnt != 100 {
		t.Errorf("si=%d cnt=%d", si, cnt)
	}
}

func TestMiscHandlers_ParsePaginationCustom(t *testing.T) {
	si, cnt := parsePagination(httptest.NewRequest(http.MethodGet, "/?startIndex=5&count=25", http.NoBody))
	if si != 5 || cnt != 25 {
		t.Errorf("si=%d cnt=%d", si, cnt)
	}
}

func TestMiscHandlers_ParsePaginationInvalid(t *testing.T) {
	si, cnt := parsePagination(httptest.NewRequest(http.MethodGet, "/?startIndex=-1&count=200", http.NoBody))
	if si != 1 {
		t.Errorf("startIndex = %d", si)
	}
	if cnt != 100 {
		t.Errorf("count = %d", cnt)
	}
}

func TestMiscHandlers_SummarizeChecks(t *testing.T) {
	s := summarizeChecks([]complianceCheck{{Status: "pass"}, {Status: "pass"}, {Status: "warning"}, {Status: "fail"}})
	if s["total"] != 4 || s["pass"] != 2 || s["warning"] != 1 || s["fail"] != 1 {
		t.Errorf("summary = %v", s)
	}
}
