package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/token"
)

const testJWTSecret = "this-is-a-test-secret-that-is-at-least-32-bytes-long"

func generateTestToken(t *testing.T) string {
	t.Helper()
	tok, err := token.GenerateAccessToken(
		testJWTSecret, 15*time.Minute,
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
	handler := Auth(testJWTSecret)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
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
	handler := Auth(testJWTSecret)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
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
	handler := Auth(testJWTSecret)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
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
	handler := Auth(testJWTSecret)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
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
	// Generate an expired token by using a negative TTL trick
	tok, err := token.GenerateAccessToken(
		testJWTSecret, -1*time.Hour,
		uuid.New(), uuid.New(),
		"admin", "admin@test.com", false, "", "",
	)
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}

	handler := Auth(testJWTSecret)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
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
