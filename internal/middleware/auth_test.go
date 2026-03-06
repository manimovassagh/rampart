package middleware

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/token"
)

// testRSAKeyPEM is a 2048-bit RSA key used only in tests.
const testRSAKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAvN8Ex780DiM6xO5PgniD7BbnTEGx1IkX1LE0EbrGrZJHcVbX
IiUbxBcnAMl/PqPtpS02pf0IgaGPM1DgO10eNcGxRvUcw/H0hbOEgMFIch71egvD
d/Ag8m18vO0MaoSh7xBlJSIfgRLCpyoWwghurFuMViwMcst6Cg35W8+IOCL7KOkj
OdFWIT7baffJ2w7LGq0i3/TlSmoUNVF+sZzM2vj4QMC6T7bUI9ISx9KP1wvxAz99
c1PoSi1bu2e/Yz3CyXeg00Z1BVWNGEcM98iaajTMGP1QmqsavNMjO72Ub+1XpyQ/
ve36uznlaqHhBZqrtTI+YugLtYIRq3etuI2HmwIDAQABAoIBAB9gzeKBmZxfrfvZ
u8vpScGHbJX2tByjShpD9mqbpTZg/w2NZ+B8WciSMCCpWUKG6YxvnoylJSykMq5L
2XUDW2mC7HjlcAn9wKoV0QWzFt4e1pmYKrlaY57jIb4hg9aOgni9OJCawrEm9L/g
9jb2P6zS6NXIK6lGtNfGyo6+Q9tPa2nMF86xrzscKuT8hLq2B4YN3jdL5tCIsfO/
IcnDwL6eCw+sjjeKfsEXful+9JZSAoKr1sukeW+xSE26hZwhxmwyHDPST/D5QkJo
NvbikjKpRRWwuUTondPbpJ3d2C6vIYVXtJjdzE9RdfKDPeuTpRRbgr/bf9LLO9E6
9k6gEYECgYEA2hHOT3Wm+WnVTILaCh4Z0/qeMrpBVcUgNNfaedXdYS4MND2TY2Wr
cd2IUGYvjFJs2neattXYijR7dC9i3j40wYTE3ak1S8rjVdTjF9eoV73zfWjGLUoT
xKTidmxhaWixJJxOQXoYVxwumsebCoRJLQqs8DNauaA3HzHpRSKIm5MCgYEA3bkR
YndjBzUrWFPZnhY89DUvwTCPrbMyHSUCpRiPlcDkQwgTnqvEgutKUtEKUvWDOu0d
4DVjK/P2LwwL8tuA82WaTOZjFTZ3yGNPC+gJeamKS4PgQviEdldLU53VKQpy7kgg
bxwW+aY+hKpVo+9RerdMJRn2SUhiHaHWdedQuNkCgYEApU9UN5Y3wuEAyiRzx7Gz
4Kce39OkDbIGzShIvY1raezvYXbAUVxUUFggqtob92LQk/iRN0L7CSHp6FS3vUQo
1/6fAm3wMgmWto1QrdVVD1a2y33upYx/WdWoux9D5RVxHBDFngtBgl+h0MG5/Yn0
swlhuiEkCI2025gJfthD+LMCgYAD6u85tC5VxES9zM19k5sEHaR4X2lKgm4SQcMo
M6Tl2oCuBoiCNzrDrXCkwfjSum/VLLdobMkRz7+72RSk9+fxZQwy66c4irvXGJoe
9bylH6/H4c6moEmG5cf49EL99KdPOosIK5DkXGGiangU63efGXoI9cp6RQMmzuNB
NhMhEQKBgQDF+33l7x8rxERo1x0E/nwaPr4kISE2RbQSxXL7UARAg457+ol/bs36
O4bU15NoXuwZ1ShWIKMWxLVcEysw7pDHs3xaiokx1SIrn9nVkaOb9iK8B8qgCQ2o
W/xlOTOB+SaEdBMOGPE8JlFPydshVuTUj5wMzN4GejazNxtEXbbGaQ==
-----END RSA PRIVATE KEY-----`

var (
	testPrivKey *rsa.PrivateKey
	testPubKey  *rsa.PublicKey
)

func init() {
	block, _ := pem.Decode([]byte(testRSAKeyPEM))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic("failed to parse test RSA key: " + err.Error())
	}
	testPrivKey = key
	testPubKey = &key.PublicKey
}

const (
	testKID    = "test-kid-123"
	testIssuer = "http://localhost:8080"
)

func generateTestToken(t *testing.T) string {
	t.Helper()
	tok, err := token.GenerateAccessToken(
		testPrivKey, testKID, testIssuer, 15*time.Minute,
		uuid.New(), uuid.New(),
		"admin", "admin@test.com", true, "Admin", "User",
	)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}
	return tok
}

func TestAuthMiddlewareSuccess(t *testing.T) {
	tok := generateTestToken(t)

	var gotUser *AuthenticatedUser
	handler := Auth(testPubKey)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotUser = GetAuthenticatedUser(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotUser == nil {
		t.Fatal("expected authenticated user in context")
	}
	if gotUser.PreferredUsername != "admin" {
		t.Errorf("username = %q, want admin", gotUser.PreferredUsername)
	}
	if gotUser.Email != "admin@test.com" {
		t.Errorf("email = %q, want admin@test.com", gotUser.Email)
	}
}

func TestAuthMiddlewareMissingHeader(t *testing.T) {
	handler := Auth(testPubKey)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareInvalidFormat(t *testing.T) {
	handler := Auth(testPubKey)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	handler := Auth(testPubKey)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareExpiredToken(t *testing.T) {
	tok, err := token.GenerateAccessToken(
		testPrivKey, testKID, testIssuer, -1*time.Hour,
		uuid.New(), uuid.New(),
		"admin", "admin@test.com", false, "", "",
	)
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}

	handler := Auth(testPubKey)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGetAuthenticatedUserNilContext(t *testing.T) {
	//nolint:staticcheck // testing nil context intentionally
	user := GetAuthenticatedUser(nil)
	if user != nil {
		t.Error("expected nil for nil context")
	}
}

func TestGetAuthenticatedUserEmptyContext(t *testing.T) {
	user := GetAuthenticatedUser(context.TODO())
	if user != nil {
		t.Error("expected nil for context without user")
	}
}

func TestSetAuthenticatedUser(t *testing.T) {
	expected := &AuthenticatedUser{
		UserID:            uuid.New(),
		OrgID:             uuid.New(),
		PreferredUsername: "testuser",
		Email:             "test@example.com",
		EmailVerified:     true,
		GivenName:         "Test",
		FamilyName:        "User",
	}

	ctx := SetAuthenticatedUser(context.Background(), expected)
	got := GetAuthenticatedUser(ctx)

	if got == nil {
		t.Fatal("expected user in context")
	}
	if got.UserID != expected.UserID {
		t.Errorf("UserID = %v, want %v", got.UserID, expected.UserID)
	}
	if got.PreferredUsername != "testuser" {
		t.Errorf("PreferredUsername = %q, want testuser", got.PreferredUsername)
	}
	if got.Email != "test@example.com" {
		t.Errorf("Email = %q, want test@example.com", got.Email)
	}
	if !got.EmailVerified {
		t.Error("expected EmailVerified to be true")
	}
	if got.GivenName != "Test" {
		t.Errorf("GivenName = %q, want Test", got.GivenName)
	}
	if got.FamilyName != "User" {
		t.Errorf("FamilyName = %q, want User", got.FamilyName)
	}
}

func TestAuthMiddlewareBearerCaseInsensitive(t *testing.T) {
	tok := generateTestToken(t)

	handler := Auth(testPubKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetAuthenticatedUser(r.Context())
		if user == nil {
			t.Error("expected authenticated user in context")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req.Header.Set("Authorization", "bearer "+tok)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthMiddlewareOnlyTokenNoBearer(t *testing.T) {
	handler := Auth(testPubKey)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req.Header.Set("Authorization", "just-a-token-no-bearer-prefix")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWriteAuthErrorIncludesRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Set(HeaderRequestID, "test-req-123")
	writeAuthError(w, "test error")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	body := w.Body.String()
	if !strings.Contains(body, "test-req-123") {
		t.Errorf("response body missing request ID, got: %s", body)
	}
	if !strings.Contains(body, "unauthorized") {
		t.Errorf("response body missing error code, got: %s", body)
	}
	if !strings.Contains(body, "test error") {
		t.Errorf("response body missing error description, got: %s", body)
	}
}

func TestHasRoleReturnsTrue(t *testing.T) {
	user := &AuthenticatedUser{
		Roles: []string{"admin", "viewer"},
	}
	if !user.HasRole("admin") {
		t.Error("expected HasRole to return true for admin")
	}
}

func TestHasRoleReturnsFalse(t *testing.T) {
	user := &AuthenticatedUser{
		Roles: []string{"viewer"},
	}
	if user.HasRole("admin") {
		t.Error("expected HasRole to return false for admin")
	}
}

func TestHasRoleEmptyRoles(t *testing.T) {
	user := &AuthenticatedUser{}
	if user.HasRole("admin") {
		t.Error("expected HasRole to return false for empty roles")
	}
}

func generateTestTokenWithRoles(t *testing.T, roles ...string) string {
	t.Helper()
	tok, err := token.GenerateAccessToken(
		testPrivKey, testKID, testIssuer, 15*time.Minute,
		uuid.New(), uuid.New(),
		"testuser", "test@test.com", true, "Test", "User",
		roles...,
	)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}
	return tok
}

func TestRequireRoleAdminAllowed(t *testing.T) {
	tok := generateTestTokenWithRoles(t, "admin", "viewer")

	called := false
	handler := Auth(testPubKey)(
		RequireRole("admin")(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !called {
		t.Error("expected handler to be called for admin user")
	}
}

func TestRequireRoleRegularUserForbidden(t *testing.T) {
	tok := generateTestTokenWithRoles(t, "viewer")

	handler := Auth(testPubKey)(
		RequireRole("admin")(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("handler should not be called for non-admin user")
			}),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	body := w.Body.String()
	if !strings.Contains(body, "forbidden") {
		t.Errorf("response body missing 'forbidden' error code, got: %s", body)
	}
	if !strings.Contains(body, "admin") {
		t.Errorf("response body should mention required role, got: %s", body)
	}
}

func TestRequireRoleNoRolesInToken(t *testing.T) {
	tok := generateTestTokenWithRoles(t)

	handler := Auth(testPubKey)(
		RequireRole("admin")(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("handler should not be called for user with no roles")
			}),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireRoleUnauthenticatedReturns401(t *testing.T) {
	handler := Auth(testPubKey)(
		RequireRole("admin")(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("handler should not be called")
			}),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireRoleNoUserInContext(t *testing.T) {
	handler := RequireRole("admin")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireRoleForbiddenResponseIsJSON(t *testing.T) {
	tok := generateTestTokenWithRoles(t, "viewer")

	handler := Auth(testPubKey)(
		RequireRole("admin")(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("handler should not be called")
			}),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestAuthMiddlewarePopulatesRoles(t *testing.T) {
	tok := generateTestTokenWithRoles(t, "admin", "editor")

	var gotUser *AuthenticatedUser
	handler := Auth(testPubKey)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotUser = GetAuthenticatedUser(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if gotUser == nil {
		t.Fatal("expected authenticated user")
	}
	if len(gotUser.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(gotUser.Roles))
	}
	if !gotUser.HasRole("admin") {
		t.Error("expected user to have admin role")
	}
	if !gotUser.HasRole("editor") {
		t.Error("expected user to have editor role")
	}
}

func TestWriteAuthErrorContentType(t *testing.T) {
	w := httptest.NewRecorder()
	writeAuthError(w, "some error")

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestAuthMiddlewareSuccessPopulatesAllFields(t *testing.T) {
	tok := generateTestToken(t)

	var gotUser *AuthenticatedUser
	handler := Auth(testPubKey)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotUser = GetAuthenticatedUser(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if gotUser == nil {
		t.Fatal("expected authenticated user")
	}
	if gotUser.UserID == uuid.Nil {
		t.Error("expected non-nil UserID")
	}
	if gotUser.OrgID == uuid.Nil {
		t.Error("expected non-nil OrgID")
	}
	if !gotUser.EmailVerified {
		t.Error("expected EmailVerified to be true")
	}
	if gotUser.GivenName != "Admin" {
		t.Errorf("GivenName = %q, want Admin", gotUser.GivenName)
	}
	if gotUser.FamilyName != "User" {
		t.Errorf("FamilyName = %q, want User", gotUser.FamilyName)
	}
}
