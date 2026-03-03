package handler

import (
	"bytes"
	"context"
	"encoding/json"
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
	defaultOrgID     uuid.UUID
	defaultOrgErr    error
	userByID         *model.User
	userByIDErr      error
	emailUser        *model.User
	emailErr         error
	usernameUser     *model.User
	usernameErr      error
	createdUser      *model.User
	createErr        error
	listUsers        []*model.User
	listTotal        int
	listErr          error
	updatedUser      *model.User
	updateErr        error
	deleteErr        error
	updatePwErr      error
	countUsers       int
	countUsersErr    error
	countRecent      int
	countRecentErr   error
}

func (m *mockAdminUserStore) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return m.defaultOrgID, m.defaultOrgErr
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
func (m *mockAdminUserStore) UpdateUser(_ context.Context, _ uuid.UUID, _ *model.UpdateUserRequest) (*model.User, error) {
	return m.updatedUser, m.updateErr
}
func (m *mockAdminUserStore) DeleteUser(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}
func (m *mockAdminUserStore) UpdatePassword(_ context.Context, _ uuid.UUID, _ []byte) error {
	return m.updatePwErr
}
func (m *mockAdminUserStore) CountUsers(_ context.Context, _ uuid.UUID) (int, error) {
	return m.countUsers, m.countUsersErr
}
func (m *mockAdminUserStore) CountRecentUsers(_ context.Context, _ uuid.UUID, _ int) (int, error) {
	return m.countRecent, m.countRecentErr
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
func (m *mockAdminSessionStore) CountActive(_ context.Context) (int, error) {
	return m.countActive, m.countActiveErr
}
func (m *mockAdminSessionStore) DeleteByUserID(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func newAdminTestUser() *model.User {
	return &model.User{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Username:  "testuser",
		Email:     "test@rampart.local",
		Enabled:   true,
		GivenName: "Test",
		FamilyName: "User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newTestAdminHandler(store *mockAdminUserStore, sessions *mockAdminSessionStore) *AdminHandler {
	return NewAdminHandler(store, sessions, noopLogger())
}

func TestAdminStatsSuccess(t *testing.T) {
	store := &mockAdminUserStore{
		defaultOrgID: uuid.New(),
		countUsers:   42,
		countRecent:  5,
	}
	sessions := &mockAdminSessionStore{countActive: 10}
	h := newTestAdminHandler(store, sessions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
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
}

func TestAdminListUsersSuccess(t *testing.T) {
	user := newAdminTestUser()
	store := &mockAdminUserStore{
		defaultOrgID: user.OrgID,
		listUsers:    []*model.User{user},
		listTotal:    1,
	}
	sessions := &mockAdminSessionStore{countByUser: 2}
	h := newTestAdminHandler(store, sessions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?page=1&limit=20", http.NoBody)
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
	store := &mockAdminUserStore{defaultOrgID: uuid.New()}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	body := []byte(`{"username":"newuser","email":"new@test.com","password":"Str0ng!Pass","given_name":"New","family_name":"User","enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestAdminCreateUserDuplicateEmail(t *testing.T) {
	existing := newAdminTestUser()
	store := &mockAdminUserStore{
		defaultOrgID: existing.OrgID,
		emailUser:    existing,
	}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	body := []byte(`{"username":"other","email":"test@rampart.local","password":"Str0ng!Pass","given_name":"A","family_name":"B","enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestAdminCreateUserValidationError(t *testing.T) {
	store := &mockAdminUserStore{defaultOrgID: uuid.New()}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	body := []byte(`{"username":"x","email":"bad","password":"short","given_name":"","family_name":"","enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+user.ID.String(), http.NoBody)
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+uuid.New().String(), http.NoBody)
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

	body := []byte(`{"username":"updated","email":"updated@test.com","given_name":"Up","family_name":"Dated","enabled":true,"email_verified":true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+user.ID.String(), bytes.NewReader(body))
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

	body := []byte(`{"password":"NewStr0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+user.ID.String()+"/reset-password", bytes.NewReader(body))
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
	userID := uuid.New()
	sess := &session.Session{
		ID:        uuid.New(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{sessions: []*session.Session{sess}}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/users/{id}/sessions", h.ListSessions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+userID.String()+"/sessions", http.NoBody)
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
	userID := uuid.New()
	store := &mockAdminUserStore{}
	sessions := &mockAdminSessionStore{}
	h := newTestAdminHandler(store, sessions)

	r := chi.NewRouter()
	r.Delete("/api/v1/admin/users/{id}/sessions", h.RevokeSessions)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+userID.String()+"/sessions", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}
