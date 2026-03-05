package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// mockExportImportStore implements ExportImportStore for testing.
type mockExportImportStore struct {
	exported  *model.OrgExport
	exportErr error
	importErr error
}

func (m *mockExportImportStore) ExportOrganization(_ context.Context, _ uuid.UUID) (*model.OrgExport, error) {
	return m.exported, m.exportErr
}

func (m *mockExportImportStore) ImportOrganization(_ context.Context, _ *model.OrgExport) error {
	return m.importErr
}

func newTestOrgExport() *model.OrgExport {
	return &model.OrgExport{
		Organization: model.OrgExportData{
			Name:        "test-org",
			Slug:        "test-org",
			DisplayName: "Test Organization",
		},
		Roles: []model.RoleExport{
			{Name: "admin", Description: "Administrator"},
		},
		Groups: []model.GroupExport{
			{Name: "devs", Description: "Developers", Roles: []string{"admin"}},
		},
		Clients: []model.ClientExport{
			{ClientID: "my-app", Name: "My App", ClientType: "public", Enabled: true},
		},
	}
}

func TestExportHandlerSuccess(t *testing.T) {
	exported := newTestOrgExport()
	store := &mockExportImportStore{exported: exported}
	h := ExportHandler(store, noopLogger())

	orgID := uuid.New()
	r := chi.NewRouter()
	r.Get("/api/v1/admin/organizations/{id}/export", h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations/"+orgID.String()+"/export", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result model.OrgExport
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Organization.Name != "test-org" {
		t.Errorf("org name = %q, want test-org", result.Organization.Name)
	}
	if len(result.Roles) != 1 {
		t.Errorf("roles length = %d, want 1", len(result.Roles))
	}
	if len(result.Groups) != 1 {
		t.Errorf("groups length = %d, want 1", len(result.Groups))
	}
	if len(result.Clients) != 1 {
		t.Errorf("clients length = %d, want 1", len(result.Clients))
	}
}

func TestExportHandlerInvalidUUID(t *testing.T) {
	store := &mockExportImportStore{}
	h := ExportHandler(store, noopLogger())

	r := chi.NewRouter()
	r.Get("/api/v1/admin/organizations/{id}/export", h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations/not-a-uuid/export", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestExportHandlerStoreError(t *testing.T) {
	store := &mockExportImportStore{exportErr: fmt.Errorf("db connection failed")}
	h := ExportHandler(store, noopLogger())

	orgID := uuid.New()
	r := chi.NewRouter()
	r.Get("/api/v1/admin/organizations/{id}/export", h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations/"+orgID.String()+"/export", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestImportHandlerSuccess(t *testing.T) {
	store := &mockExportImportStore{}
	h := ImportHandler(store, noopLogger())

	orgID := uuid.New()
	r := chi.NewRouter()
	r.Post("/api/v1/admin/organizations/{id}/import", h)

	payload := newTestOrgExport()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/organizations/"+orgID.String()+"/import", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestImportHandlerInvalidUUID(t *testing.T) {
	store := &mockExportImportStore{}
	h := ImportHandler(store, noopLogger())

	r := chi.NewRouter()
	r.Post("/api/v1/admin/organizations/{id}/import", h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/organizations/bad-uuid/import", bytes.NewReader([]byte("{}")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestImportHandlerInvalidJSON(t *testing.T) {
	store := &mockExportImportStore{}
	h := ImportHandler(store, noopLogger())

	orgID := uuid.New()
	r := chi.NewRouter()
	r.Post("/api/v1/admin/organizations/{id}/import", h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/organizations/"+orgID.String()+"/import", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestImportHandlerMissingName(t *testing.T) {
	store := &mockExportImportStore{}
	h := ImportHandler(store, noopLogger())

	orgID := uuid.New()
	r := chi.NewRouter()
	r.Post("/api/v1/admin/organizations/{id}/import", h)

	payload := &model.OrgExport{
		Organization: model.OrgExportData{Name: "", Slug: ""},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/organizations/"+orgID.String()+"/import", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestImportHandlerMissingSlug(t *testing.T) {
	store := &mockExportImportStore{}
	h := ImportHandler(store, noopLogger())

	orgID := uuid.New()
	r := chi.NewRouter()
	r.Post("/api/v1/admin/organizations/{id}/import", h)

	payload := &model.OrgExport{
		Organization: model.OrgExportData{Name: "test", Slug: ""},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/organizations/"+orgID.String()+"/import", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestImportHandlerStoreError(t *testing.T) {
	store := &mockExportImportStore{importErr: fmt.Errorf("import failed")}
	h := ImportHandler(store, noopLogger())

	orgID := uuid.New()
	r := chi.NewRouter()
	r.Post("/api/v1/admin/organizations/{id}/import", h)

	payload := newTestOrgExport()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/organizations/"+orgID.String()+"/import", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
