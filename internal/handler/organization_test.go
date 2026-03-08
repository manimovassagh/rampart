package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

type mockOrgStore struct {
	org           *model.Organization
	orgErr        error
	orgs          []*model.Organization
	orgsTotal     int
	orgsErr       error
	createdOrg    *model.Organization
	createErr     error
	updatedOrg    *model.Organization
	updateErr     error
	deleteErr     error
	countUsers    int
	countUsersErr error
}

func (m *mockOrgStore) GetOrganizationByID(_ context.Context, _ uuid.UUID) (*model.Organization, error) {
	return m.org, m.orgErr
}
func (m *mockOrgStore) ListOrganizations(_ context.Context, _ string, _, _ int) ([]*model.Organization, int, error) {
	return m.orgs, m.orgsTotal, m.orgsErr
}
func (m *mockOrgStore) CreateOrganization(_ context.Context, req *model.CreateOrgRequest) (*model.Organization, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createdOrg != nil {
		return m.createdOrg, nil
	}
	return &model.Organization{
		ID:          uuid.New(),
		Name:        req.Name,
		Slug:        req.Slug,
		DisplayName: req.DisplayName,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}
func (m *mockOrgStore) UpdateOrganization(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgRequest) (*model.Organization, error) {
	return m.updatedOrg, m.updateErr
}
func (m *mockOrgStore) DeleteOrganization(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}
func (m *mockOrgStore) CountUsers(_ context.Context, _ uuid.UUID) (int, error) {
	return m.countUsers, m.countUsersErr
}

// ── stub methods to satisfy store.OrgReader ──

func (m *mockOrgStore) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockOrgStore) GetOrganizationIDBySlug(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// ── stub methods to satisfy store.OrgLister ──

func (m *mockOrgStore) CountOrganizations(_ context.Context) (int, error) { return 0, nil }

// ── stub methods to satisfy store.UserLister ──

func (m *mockOrgStore) ListUsers(_ context.Context, _ uuid.UUID, _, _ string, _, _ int) ([]*model.User, int, error) {
	return nil, 0, nil
}
func (m *mockOrgStore) CountRecentUsers(_ context.Context, _ uuid.UUID, _ int) (int, error) {
	return 0, nil
}

type mockOrgSettingsStore struct {
	settings    *model.OrgSettings
	settingsErr error
	updated     *model.OrgSettings
	updateErr   error
}

func (m *mockOrgSettingsStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.settings, m.settingsErr
}
func (m *mockOrgSettingsStore) UpdateOrgSettings(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	return m.updated, m.updateErr
}

func newTestOrg() *model.Organization {
	return &model.Organization{
		ID:          uuid.New(),
		Name:        "test-org",
		Slug:        "test-org",
		DisplayName: "Test Organization",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func newTestOrgSettings(orgID uuid.UUID) *model.OrgSettings {
	return &model.OrgSettings{
		ID:                       uuid.New(),
		OrgID:                    orgID,
		PasswordMinLength:        8,
		PasswordRequireUppercase: true,
		PasswordRequireLowercase: true,
		PasswordRequireNumbers:   true,
		PasswordRequireSymbols:   true,
		MFAEnforcement:           "off",
		AccessTokenTTL:           15 * time.Minute,
		RefreshTokenTTL:          7 * 24 * time.Hour,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}
}

func newTestOrgHandler(store *mockOrgStore, settings *mockOrgSettingsStore) *OrgHandler {
	return NewOrgHandler(store, settings, noopLogger())
}

func TestOrgListSuccess(t *testing.T) {
	org := newTestOrg()
	store := &mockOrgStore{orgs: []*model.Organization{org}, orgsTotal: 1, countUsers: 5}
	settings := &mockOrgSettingsStore{}
	h := newTestOrgHandler(store, settings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations", http.NoBody)
	w := httptest.NewRecorder()

	h.ListOrgs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp model.ListOrgsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Total)
	}
	if len(resp.Organizations) != 1 {
		t.Fatalf("organizations length = %d, want 1", len(resp.Organizations))
	}
	if resp.Organizations[0].UserCount != 5 {
		t.Errorf("user_count = %d, want 5", resp.Organizations[0].UserCount)
	}
}

func TestOrgCreateSuccess(t *testing.T) {
	store := &mockOrgStore{}
	settings := &mockOrgSettingsStore{}
	h := newTestOrgHandler(store, settings)

	body := []byte(`{"name":"acme","slug":"acme","display_name":"ACME Corp"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/organizations", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateOrg(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestOrgCreateMissingFields(t *testing.T) {
	store := &mockOrgStore{}
	settings := &mockOrgSettingsStore{}
	h := newTestOrgHandler(store, settings)

	body := []byte(`{"name":"","slug":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/organizations", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateOrg(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgCreateDuplicateSlug(t *testing.T) {
	store := &mockOrgStore{createErr: fmt.Errorf("duplicate key value violates unique constraint")}
	settings := &mockOrgSettingsStore{}
	h := newTestOrgHandler(store, settings)

	body := []byte(`{"name":"acme","slug":"acme","display_name":"ACME Corp"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/organizations", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateOrg(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestOrgGetSuccess(t *testing.T) {
	org := newTestOrg()
	store := &mockOrgStore{org: org, countUsers: 10}
	settings := &mockOrgSettingsStore{}
	h := newTestOrgHandler(store, settings)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/organizations/{id}", h.GetOrg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations/"+org.ID.String(), http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp model.OrgResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.UserCount != 10 {
		t.Errorf("user_count = %d, want 10", resp.UserCount)
	}
}

func TestOrgGetNotFound(t *testing.T) {
	store := &mockOrgStore{}
	settings := &mockOrgSettingsStore{}
	h := newTestOrgHandler(store, settings)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/organizations/{id}", h.GetOrg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations/"+uuid.New().String(), http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgUpdateSuccess(t *testing.T) {
	org := newTestOrg()
	store := &mockOrgStore{updatedOrg: org, countUsers: 3}
	settings := &mockOrgSettingsStore{}
	h := newTestOrgHandler(store, settings)

	r := chi.NewRouter()
	r.Put("/api/v1/admin/organizations/{id}", h.UpdateOrg)

	body := []byte(`{"name":"updated","display_name":"Updated Org","enabled":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/organizations/"+org.ID.String(), bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOrgDeleteSuccess(t *testing.T) {
	store := &mockOrgStore{}
	settings := &mockOrgSettingsStore{}
	h := newTestOrgHandler(store, settings)

	r := chi.NewRouter()
	r.Delete("/api/v1/admin/organizations/{id}", h.DeleteOrg)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/organizations/"+uuid.New().String(), http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestOrgDeleteDefaultProtected(t *testing.T) {
	store := &mockOrgStore{deleteErr: fmt.Errorf("cannot delete the default organization")}
	settings := &mockOrgSettingsStore{}
	h := newTestOrgHandler(store, settings)

	r := chi.NewRouter()
	r.Delete("/api/v1/admin/organizations/{id}", h.DeleteOrg)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/organizations/"+uuid.New().String(), http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgGetSettingsSuccess(t *testing.T) {
	orgID := uuid.New()
	s := newTestOrgSettings(orgID)
	store := &mockOrgStore{}
	settings := &mockOrgSettingsStore{settings: s}
	h := newTestOrgHandler(store, settings)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/organizations/{id}/settings", h.GetOrgSettings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations/"+orgID.String()+"/settings", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp model.OrgSettingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.PasswordMinLength != 8 {
		t.Errorf("password_min_length = %d, want 8", resp.PasswordMinLength)
	}
	if resp.AccessTokenTTLSeconds != 900 {
		t.Errorf("access_token_ttl_seconds = %d, want 900", resp.AccessTokenTTLSeconds)
	}
}

func TestOrgUpdateSettingsSuccess(t *testing.T) {
	orgID := uuid.New()
	s := newTestOrgSettings(orgID)
	s.PasswordMinLength = 12
	store := &mockOrgStore{}
	settings := &mockOrgSettingsStore{updated: s}
	h := newTestOrgHandler(store, settings)

	r := chi.NewRouter()
	r.Put("/api/v1/admin/organizations/{id}/settings", h.UpdateOrgSettings)

	body := []byte(`{
		"password_min_length":12,
		"password_require_uppercase":true,
		"password_require_lowercase":true,
		"password_require_numbers":true,
		"password_require_symbols":false,
		"mfa_enforcement":"optional",
		"access_token_ttl_seconds":1800,
		"refresh_token_ttl_seconds":86400,
		"logo_url":"",
		"primary_color":"",
		"background_color":""
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/organizations/"+orgID.String()+"/settings", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgUpdateSettingsInvalidMFA(t *testing.T) {
	store := &mockOrgStore{}
	settings := &mockOrgSettingsStore{}
	h := newTestOrgHandler(store, settings)

	r := chi.NewRouter()
	r.Put("/api/v1/admin/organizations/{id}/settings", h.UpdateOrgSettings)

	body := []byte(`{
		"password_min_length":8,
		"mfa_enforcement":"invalid",
		"access_token_ttl_seconds":900,
		"refresh_token_ttl_seconds":86400
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/organizations/"+uuid.New().String()+"/settings", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
