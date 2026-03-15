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

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
)

// mockAdminUserStore implements AdminUserStore for testing.
type mockAdminUserStore struct {
	userByID       *model.User
	userByIDErr    error
	emailUser      *model.User
	emailErr       error
	usernameUser   *model.User
	usernameErr    error
	createdUser    *model.User
	createErr      error
	listUsers      []*model.User
	listTotal      int
	listErr        error
	updatedUser    *model.User
	updateErr      error
	deleteErr      error
	updatePwErr    error
	countUsers     int
	countUsersErr  error
	countRecent    int
	countRecentErr error
	countOrgs      int
	countOrgsErr   error
	orgSettings    *model.OrgSettings
	orgSettingsErr error
	lastOrgID      uuid.UUID
}

func (m *mockAdminUserStore) GetUserByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return m.userByID, m.userByIDErr
}
func (m *mockAdminUserStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return m.emailUser, m.emailErr
}
func (m *mockAdminUserStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return m.usernameUser, m.usernameErr
}
func (m *mockAdminUserStore) CreateUser(_ context.Context, user *model.User) (*model.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createdUser != nil {
		return m.createdUser, nil
	}
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	return user, nil
}
func (m *mockAdminUserStore) ListUsers(_ context.Context, _ uuid.UUID, _, _ string, _, _ int) ([]*model.User, int, error) {
	return m.listUsers, m.listTotal, m.listErr
}
func (m *mockAdminUserStore) UpdateUser(_ context.Context, _, _ uuid.UUID, _ *model.UpdateUserRequest) (*model.User, error) {
	return m.updatedUser, m.updateErr
}
func (m *mockAdminUserStore) DeleteUser(_ context.Context, _, _ uuid.UUID) error {
	return m.deleteErr
}
func (m *mockAdminUserStore) UpdatePassword(_ context.Context, _, _ uuid.UUID, _ []byte) error {
	return m.updatePwErr
}
func (m *mockAdminUserStore) CountUsers(_ context.Context, orgID uuid.UUID) (int, error) {
	m.lastOrgID = orgID
	return m.countUsers, m.countUsersErr
}
func (m *mockAdminUserStore) CountRecentUsers(_ context.Context, _ uuid.UUID, _ int) (int, error) {
	return m.countRecent, m.countRecentErr
}
func (m *mockAdminUserStore) CountOrganizations(_ context.Context) (int, error) {
	return m.countOrgs, m.countOrgsErr
}
func (m *mockAdminUserStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.orgSettings, m.orgSettingsErr
}

// ── stub methods to satisfy store.UserReader ──

func (m *mockAdminUserStore) FindUserByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}

// ── stub methods to satisfy store.UserWriter ──

func (m *mockAdminUserStore) UpdateLastLoginAt(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockAdminUserStore) IncrementFailedLogins(_ context.Context, _ uuid.UUID, _ int, _ time.Duration) error {
	return nil
}
func (m *mockAdminUserStore) ResetFailedLogins(_ context.Context, _ uuid.UUID) error { return nil }

// ── stub methods to satisfy store.OrgLister ──

func (m *mockAdminUserStore) ListOrganizations(_ context.Context, _ string, _, _ int) ([]*model.Organization, int, error) {
	return nil, 0, nil
}

// ── stub methods to satisfy store.OrgSettingsReadWriter ──

func (m *mockAdminUserStore) UpdateOrgSettings(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	return nil, nil
}

// mockAdminSessionStore implements AdminSessionStore for testing.
type mockAdminSessionStore struct {
	sessions       []*session.Session
	listErr        error
	countByUser    int
	countByUserErr error
	countActive    int
	countActiveErr error
	deleteErr      error
}

func (m *mockAdminSessionStore) ListByUserID(_ context.Context, _ uuid.UUID) ([]*session.Session, error) {
	return m.sessions, m.listErr
}
func (m *mockAdminSessionStore) CountByUserID(_ context.Context, _ uuid.UUID) (int, error) {
	return m.countByUser, m.countByUserErr
}
func (m *mockAdminSessionStore) CountActive(_ context.Context, _ uuid.UUID) (int, error) {
	return m.countActive, m.countActiveErr
}
func (m *mockAdminSessionStore) DeleteByUserID(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func newAdminTestUser() *model.User {
	return &model.User{
		ID:         uuid.New(),
		OrgID:      uuid.New(),
		Username:   "testuser",
		Email:      "test@rampart.local",
		Enabled:    true,
		GivenName:  "Test",
		FamilyName: "User",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func newTestAdminHandler(store *mockAdminUserStore, sessions *mockAdminSessionStore) *AdminHandler {
	return NewAdminHandler(store, sessions, noopLogger())
}

func TestAdminStatsSuccess(t *testing.T) {
	orgID := uuid.New()
	store := &mockAdminUserStore{
		countUsers:  42,
		countRecent: 5,
		countOrgs:   3,
	}
	sessions := &mockAdminSessionStore{countActive: 10}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: orgID}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var stats model.DashboardStats
	if err := json.NewDecoder(w.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if stats.TotalUsers != 42 {
		t.Errorf("total_users = %d, want 42", stats.TotalUsers)
	}
	if stats.ActiveSessions != 10 {
		t.Errorf("active_sessions = %d, want 10", stats.ActiveSessions)
	}
	if stats.RecentUsers != 5 {
		t.Errorf("recent_users = %d, want 5", stats.RecentUsers)
	}
	if stats.TotalOrganizations != 3 {
		t.Errorf("total_organizations = %d, want 3", stats.TotalOrganizations)
	}
}

func TestAdminListUsersSuccess(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{
		listUsers: []*model.User{user},
		listTotal: 1,
	}
	sessions := &mockAdminSessionStore{countByUser: 2}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: user.OrgID}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?page=1&limit=20", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListUsers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp model.ListUsersResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Total)
	}
	if len(resp.Users) != 1 {
		t.Fatalf("users length = %d, want 1", len(resp.Users))
	}
	if resp.Users[0].SessionCount != 2 {
		t.Errorf("session_count = %d, want 2", resp.Users[0].SessionCount)
	}
}

func TestAdminCreateUserSuccess(t *testing.T) {
	orgID := uuid.New()
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: orgID}
	body := []byte(`{"username":"newuser","email":"new@test.com","password":"Str0ng!Pass","given_name":"New","family_name":"User","enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestAdminCreateUserDuplicateEmail(t *testing.T) {
	existing := newAdminTestUser()
	store := &mockAdminUserStore{
		emailUser: existing,
	}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: existing.OrgID}
	body := []byte(`{"username":"other","email":"test@rampart.local","password":"Str0ng!Pass","given_name":"A","family_name":"B","enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestAdminCreateUserValidationError(t *testing.T) {
	orgID := uuid.New()
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: orgID}
	body := []byte(`{"username":"x","email":"bad","password":"short","given_name":"","family_name":"","enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAdminGetUserSuccess(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{userByID: user}
	sessions := &mockAdminSessionStore{countByUser: 3}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/users/{id}", h.GetUser)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: user.OrgID}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+user.ID.String(), http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp model.AdminUserResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Username != "testuser" {
		t.Errorf("username = %q, want testuser", resp.Username)
	}
	if resp.SessionCount != 3 {
		t.Errorf("session_count = %d, want 3", resp.SessionCount)
	}
}

func TestAdminGetUserNotFound(t *testing.T) {
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/users/{id}", h.GetUser)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: uuid.New()}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+uuid.New().String(), http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAdminGetUserInvalidUUID(t *testing.T) {
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/users/{id}", h.GetUser)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/not-a-uuid", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAdminUpdateUserSuccess(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{updatedUser: user}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Put("/api/v1/admin/users/{id}", h.UpdateUser)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: user.OrgID}
	body := []byte(`{"username":"updated","email":"updated@test.com","given_name":"Up","family_name":"Dated","enabled":true,"email_verified":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+user.ID.String(), bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminDeleteUserSuccess(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	// Use a different user as the authenticated caller.
	callerID := uuid.New()
	authUser := &middleware.AuthenticatedUser{UserID: callerID, OrgID: user.OrgID}

	r := chi.NewRouter()
	r.Delete("/api/v1/admin/users/{id}", h.DeleteUser)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+user.ID.String(), http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestAdminDeleteUserSelfDeletion(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: user.ID, OrgID: user.OrgID}

	r := chi.NewRouter()
	r.Delete("/api/v1/admin/users/{id}", h.DeleteUser)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+user.ID.String(), http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAdminResetPasswordSuccess(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{userByID: user}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Post("/api/v1/admin/users/{id}/reset-password", h.ResetPassword)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: user.OrgID}
	body := []byte(`{"password":"NewStr0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+user.ID.String()+"/reset-password", bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestAdminResetPasswordWeakPassword(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{userByID: user}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Post("/api/v1/admin/users/{id}/reset-password", h.ResetPassword)

	body := []byte(`{"password":"weak"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+user.ID.String()+"/reset-password", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAdminListSessionsSuccess(t *testing.T) {
	user := newAdminTestUser()
	sess := &session.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	store := &mockAdminUserStore{userByID: user}
	sessions := &mockAdminSessionStore{sessions: []*session.Session{sess}}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/users/{id}/sessions", h.ListSessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: user.OrgID}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+user.ID.String()+"/sessions", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []*model.SessionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("sessions length = %d, want 1", len(resp))
	}
}

func TestAdminRevokeSessionsSuccess(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{userByID: user}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Delete("/api/v1/admin/users/{id}/sessions", h.RevokeSessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: user.OrgID}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+user.ID.String()+"/sessions", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestAdminStatsNoAuth(t *testing.T) {
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	// No auth context set
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	w := httptest.NewRecorder()

	h.Stats(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAdminStatsCountUsersError(t *testing.T) {
	orgID := uuid.New()
	store := &mockAdminUserStore{
		countUsersErr: fmt.Errorf("db error"),
	}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: orgID}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Stats(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAdminStatsCountActiveSessionsError(t *testing.T) {
	orgID := uuid.New()
	store := &mockAdminUserStore{countUsers: 10}
	sessions := &mockAdminSessionStore{countActiveErr: fmt.Errorf("session store error")}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: orgID}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Stats(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAdminStatsCountRecentUsersError(t *testing.T) {
	orgID := uuid.New()
	store := &mockAdminUserStore{countUsers: 10, countRecentErr: fmt.Errorf("db error")}
	sessions := &mockAdminSessionStore{countActive: 5}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: orgID}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Stats(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAdminStatsCountOrgsError(t *testing.T) {
	orgID := uuid.New()
	store := &mockAdminUserStore{countUsers: 10, countRecent: 2, countOrgsErr: fmt.Errorf("db error")}
	sessions := &mockAdminSessionStore{countActive: 5}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: orgID}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Stats(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAdminListUsersNoAuth(t *testing.T) {
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", http.NoBody)
	w := httptest.NewRecorder()

	h.ListUsers(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAdminListUsersStoreError(t *testing.T) {
	store := &mockAdminUserStore{listErr: fmt.Errorf("db error")}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: uuid.New()}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?page=1&limit=20", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListUsers(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAdminCreateUserNoAuth(t *testing.T) {
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	body := []byte(`{"username":"newuser","email":"new@test.com","password":"Str0ng!Pass","given_name":"New","family_name":"User","enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAdminCreateUserInvalidJSON(t *testing.T) {
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAdminCreateUserDuplicateUsername(t *testing.T) {
	existing := newAdminTestUser()
	store := &mockAdminUserStore{
		usernameUser: existing,
	}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: existing.OrgID}
	body := []byte(`{"username":"testuser","email":"other@test.com","password":"Str0ng!Pass","given_name":"A","family_name":"B","enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestAdminUpdateUserInvalidJSON(t *testing.T) {
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Put("/api/v1/admin/users/{id}", h.UpdateUser)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+uuid.New().String(), bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAdminUpdateUserNotFound(t *testing.T) {
	store := &mockAdminUserStore{updatedUser: nil} // not found
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Put("/api/v1/admin/users/{id}", h.UpdateUser)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: uuid.New()}
	body := []byte(`{"username":"updated","email":"updated@test.com","given_name":"Up","family_name":"Dated","enabled":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+uuid.New().String(), bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAdminUpdateUserStoreError(t *testing.T) {
	store := &mockAdminUserStore{updateErr: fmt.Errorf("db error")}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Put("/api/v1/admin/users/{id}", h.UpdateUser)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: uuid.New()}
	body := []byte(`{"username":"updated","email":"updated@test.com","given_name":"Up","family_name":"Dated","enabled":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+uuid.New().String(), bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAdminDeleteUserNoAuth(t *testing.T) {
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Delete("/api/v1/admin/users/{id}", h.DeleteUser)

	// Provide auth context since resolveOrgID requires it
	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: uuid.New()}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+uuid.New().String(), http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestAdminDeleteUserSessionDeleteError(t *testing.T) {
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{deleteErr: fmt.Errorf("session delete failed")}
	h := newTestAdminHandler(store, sessions)

	callerID := uuid.New()
	authUser := &middleware.AuthenticatedUser{UserID: callerID, OrgID: uuid.New()}

	r := chi.NewRouter()
	r.Delete("/api/v1/admin/users/{id}", h.DeleteUser)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+uuid.New().String(), http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAdminDeleteUserStoreError(t *testing.T) {
	store := &mockAdminUserStore{deleteErr: fmt.Errorf("db error")}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	callerID := uuid.New()
	authUser := &middleware.AuthenticatedUser{UserID: callerID, OrgID: uuid.New()}

	r := chi.NewRouter()
	r.Delete("/api/v1/admin/users/{id}", h.DeleteUser)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+uuid.New().String(), http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAdminResetPasswordNotFound(t *testing.T) {
	store := &mockAdminUserStore{userByID: nil}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Post("/api/v1/admin/users/{id}/reset-password", h.ResetPassword)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: uuid.New()}
	body := []byte(`{"password":"NewStr0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+uuid.New().String()+"/reset-password", bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAdminResetPasswordInvalidJSON(t *testing.T) {
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Post("/api/v1/admin/users/{id}/reset-password", h.ResetPassword)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+uuid.New().String()+"/reset-password", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAdminResetPasswordUpdateError(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{userByID: user, updatePwErr: fmt.Errorf("db error")}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Post("/api/v1/admin/users/{id}/reset-password", h.ResetPassword)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: user.OrgID}
	body := []byte(`{"password":"NewStr0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+user.ID.String()+"/reset-password", bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAdminGetUserStoreError(t *testing.T) {
	store := &mockAdminUserStore{userByIDErr: fmt.Errorf("db error")}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/users/{id}", h.GetUser)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: uuid.New()}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+uuid.New().String(), http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAdminListSessionsError(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{userByID: user}
	sessions := &mockAdminSessionStore{listErr: fmt.Errorf("db error")}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/users/{id}/sessions", h.ListSessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: user.OrgID}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+user.ID.String()+"/sessions", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAdminRevokeSessionsError(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{userByID: user}
	sessions := &mockAdminSessionStore{deleteErr: fmt.Errorf("session store down")}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Delete("/api/v1/admin/users/{id}/sessions", h.RevokeSessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: user.OrgID}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+user.ID.String()+"/sessions", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAdminResolveOrgIDIgnoresHeaderWithoutSuperAdmin(t *testing.T) {
	orgID := uuid.New()
	headerOrgID := uuid.New()
	store := &mockAdminUserStore{countUsers: 5, countRecent: 1, countOrgs: 1}
	sessions := &mockAdminSessionStore{countActive: 2}
	h := newTestAdminHandler(store, sessions)

	// User without super_admin role — header should be ignored
	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: orgID, Roles: []string{"admin"}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	req.Header.Set("X-Org-Context", headerOrgID.String())
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify the store was queried with the user's own org, not the header value
	if store.lastOrgID != orgID {
		t.Errorf("org = %s, want %s (user's own org, not header)", store.lastOrgID, orgID)
	}
}

func TestAdminResolveOrgIDAllowsSuperAdmin(t *testing.T) {
	orgID := uuid.New()
	headerOrgID := uuid.New()
	store := &mockAdminUserStore{countUsers: 5, countRecent: 1, countOrgs: 1}
	sessions := &mockAdminSessionStore{countActive: 2}
	h := newTestAdminHandler(store, sessions)

	// User with super_admin role — header should be honored
	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: orgID, Roles: []string{"super_admin"}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	req.Header.Set("X-Org-Context", headerOrgID.String())
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify the store was queried with the header org
	if store.lastOrgID != headerOrgID {
		t.Errorf("org = %s, want %s (header org for super_admin)", store.lastOrgID, headerOrgID)
	}
}

func TestAdminListUsersWithPaginationLimits(t *testing.T) {
	store := &mockAdminUserStore{
		listUsers: []*model.User{},
		listTotal: 0,
	}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: uuid.New()}

	// Test with limit exceeding max
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?page=1&limit=200", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListUsers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp model.ListUsersResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Limit != maxPageLimit {
		t.Errorf("limit = %d, want %d (clamped to max)", resp.Limit, maxPageLimit)
	}
}

func TestAdminCreateUserInvalidGivenName(t *testing.T) {
	orgID := uuid.New()
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: orgID}
	body := []byte(`{"username":"newuser","email":"new@test.com","password":"Str0ng!Pass","given_name":"<script>","family_name":"User","enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestAdminUpdateUserInvalidFamilyName(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{updatedUser: user}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Put("/api/v1/admin/users/{id}", h.UpdateUser)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: user.OrgID}
	body := []byte(`{"username":"updated","email":"updated@test.com","given_name":"Up","family_name":"Da>ted","enabled":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+user.ID.String(), bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestAdminUpdateUserValidNames(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{updatedUser: user}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Put("/api/v1/admin/users/{id}", h.UpdateUser)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: user.OrgID}
	body := []byte(`{"username":"updated","email":"updated@test.com","given_name":"Valid","family_name":"Name","enabled":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+user.ID.String(), bytes.NewReader(body))
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}
