package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/handler"
	"github.com/manimovassagh/rampart/internal/model"
)

const validRegBody = `{"username":"johndoe","email":"john@example.com","password":"Str0ng!Pass","given_name":"John","family_name":"Doe"}`

// mockDB implements both handler.Pinger and handler.UserStore for integration tests.
type mockDB struct {
	pingErr       error
	defaultOrgID  uuid.UUID
	defaultOrgErr error
	users         map[string]*model.User
	usersByName   map[string]*model.User
}

func newMockDB() *mockDB {
	return &mockDB{
		defaultOrgID: uuid.New(),
		users:        make(map[string]*model.User),
		usersByName:  make(map[string]*model.User),
	}
}

func (m *mockDB) Ping(_ context.Context) error {
	return m.pingErr
}

func (m *mockDB) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return m.defaultOrgID, m.defaultOrgErr
}

func (m *mockDB) GetOrganizationIDBySlug(_ context.Context, _ string) (uuid.UUID, error) {
	return m.defaultOrgID, m.defaultOrgErr
}

func (m *mockDB) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return nil, nil
}

func (m *mockDB) GetUserByEmail(_ context.Context, email string, _ uuid.UUID) (*model.User, error) {
	return m.users[email], nil
}

func (m *mockDB) GetUserByUsername(_ context.Context, username string, _ uuid.UUID) (*model.User, error) {
	return m.usersByName[username], nil
}

func (m *mockDB) CreateUser(_ context.Context, user *model.User) (*model.User, error) {
	now := time.Now()
	created := &model.User{
		ID:         uuid.New(),
		OrgID:      user.OrgID,
		Username:   user.Username,
		Email:      user.Email,
		GivenName:  user.GivenName,
		FamilyName: user.FamilyName,
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	m.users[user.Email] = created
	m.usersByName[user.Username] = created
	return created, nil
}

func apiTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func setupTestServer(db *mockDB) *httptest.Server {
	logger := apiTestLogger()
	router := NewRouter(logger, []string{"*"})

	healthH := handler.NewHealthHandler(db)
	RegisterHealthRoutes(router, healthH.Liveness, healthH.Readiness)

	registerH := handler.NewRegisterHandler(db, logger)
	RegisterAuthRoutes(router, registerH.Register)

	return httptest.NewServer(router)
}

func closeBody(t *testing.T, resp *http.Response) {
	t.Helper()
	if err := resp.Body.Close(); err != nil {
		t.Errorf("failed to close response body: %v", err)
	}
}

func TestAPIHealthzEndpoint(t *testing.T) {
	db := newMockDB()
	srv := setupTestServer(db)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz error: %v", err)
	}
	defer closeBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /healthz status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["status"] != "alive" {
		t.Errorf("status = %q, want alive", body["status"])
	}
}

func TestAPIReadyzEndpointHealthy(t *testing.T) {
	db := newMockDB()
	srv := setupTestServer(db)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz error: %v", err)
	}
	defer closeBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /readyz status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["status"] != "ready" {
		t.Errorf("status = %q, want ready", body["status"])
	}
}

func TestAPIReadyzEndpointUnhealthy(t *testing.T) {
	db := newMockDB()
	db.pingErr = http.ErrServerClosed
	srv := setupTestServer(db)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz error: %v", err)
	}
	defer closeBody(t, resp)

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("GET /readyz status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestAPIRegisterSuccess(t *testing.T) {
	db := newMockDB()
	srv := setupTestServer(db)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/register", "application/json", bytes.NewReader([]byte(validRegBody)))
	if err != nil {
		t.Fatalf("POST /register error: %v", err)
	}
	defer closeBody(t, resp)

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /register status = %d, want %d, body: %s", resp.StatusCode, http.StatusCreated, respBody)
	}

	var user model.UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if user.Username != "johndoe" {
		t.Errorf("username = %q, want johndoe", user.Username)
	}
	if user.Email != "john@example.com" {
		t.Errorf("email = %q, want john@example.com", user.Email)
	}
}

func TestAPIRegisterResponseHasRequestID(t *testing.T) {
	db := newMockDB()
	srv := setupTestServer(db)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/register", "application/json", bytes.NewReader([]byte(validRegBody)))
	if err != nil {
		t.Fatalf("POST /register error: %v", err)
	}
	defer closeBody(t, resp)

	reqID := resp.Header.Get("X-Request-Id")
	if reqID == "" {
		t.Error("response missing X-Request-Id header")
	}
}

func TestAPIRegisterValidationErrors(t *testing.T) {
	db := newMockDB()
	srv := setupTestServer(db)
	defer srv.Close()

	body := `{"username": "", "email": "bad", "password": "weak"}`

	resp, err := http.Post(srv.URL+"/register", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("POST /register error: %v", err)
	}
	defer closeBody(t, resp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("POST /register status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var ve apierror.ValidationError
	if err := json.NewDecoder(resp.Body).Decode(&ve); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if ve.Code != "validation_error" {
		t.Errorf("error = %q, want validation_error", ve.Code)
	}
	if len(ve.Fields) == 0 {
		t.Error("expected field errors, got none")
	}
}

func TestAPIRegisterDuplicateEmail(t *testing.T) {
	db := newMockDB()
	db.users["john@example.com"] = &model.User{Email: "john@example.com"}
	srv := setupTestServer(db)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/register", "application/json", bytes.NewReader([]byte(validRegBody)))
	if err != nil {
		t.Fatalf("POST /register error: %v", err)
	}
	defer closeBody(t, resp)

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("POST /register status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestAPIRegisterDuplicateUsername(t *testing.T) {
	db := newMockDB()
	db.usersByName["johndoe"] = &model.User{Username: "johndoe"}
	srv := setupTestServer(db)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/register", "application/json", bytes.NewReader([]byte(validRegBody)))
	if err != nil {
		t.Fatalf("POST /register error: %v", err)
	}
	defer closeBody(t, resp)

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("POST /register status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestAPIRegisterInvalidJSON(t *testing.T) {
	db := newMockDB()
	srv := setupTestServer(db)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/register", "application/json", bytes.NewReader([]byte("not json")))
	if err != nil {
		t.Fatalf("POST /register error: %v", err)
	}
	defer closeBody(t, resp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("POST /register status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestAPIRegisterPasswordNeverInResponse(t *testing.T) {
	db := newMockDB()
	srv := setupTestServer(db)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/register", "application/json", bytes.NewReader([]byte(validRegBody)))
	if err != nil {
		t.Fatalf("POST /register error: %v", err)
	}
	defer closeBody(t, resp)

	respBody, _ := io.ReadAll(resp.Body)
	if bytes.Contains(respBody, []byte("password_hash")) {
		t.Error("response contains password_hash — this must never be exposed")
	}
	if bytes.Contains(respBody, []byte("argon2")) {
		t.Error("response contains argon2 hash data")
	}
}

func TestAPINotFoundEndpoint(t *testing.T) {
	db := newMockDB()
	srv := setupTestServer(db)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/nonexistent")
	if err != nil {
		t.Fatalf("GET /nonexistent error: %v", err)
	}
	defer closeBody(t, resp)

	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET /nonexistent status = %d, want 404 or 405", resp.StatusCode)
	}
}

func TestAPICORSPreflight(t *testing.T) {
	db := newMockDB()
	srv := setupTestServer(db)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/register", http.NoBody)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	resp, err := http.DefaultClient.Do(req) //nolint:gosec // test code with test server URL
	if err != nil {
		t.Fatalf("OPTIONS /register error: %v", err)
	}
	defer closeBody(t, resp)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("OPTIONS /register status = %d, want 200 or 204", resp.StatusCode)
	}

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin == "" {
		t.Error("missing Access-Control-Allow-Origin header on CORS preflight")
	}
}
