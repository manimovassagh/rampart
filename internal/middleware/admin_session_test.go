package middleware

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/token"
)

var testHMACKey = []byte("test-hmac-key-for-middleware-test")

func TestSignAndVerifyCookie(t *testing.T) {
	value := "some-token-value"
	signed := signCookie(value, testHMACKey)

	got, ok := verifySignedCookie(signed, testHMACKey)
	if !ok {
		t.Fatal("expected verification to succeed")
	}
	if got != value {
		t.Errorf("got %q, want %q", got, value)
	}
}

func TestVerifySignedCookieTamperedSignature(t *testing.T) {
	signed := signCookie("my-value", testHMACKey)
	// Tamper with the signature
	tampered := "deadbeef" + signed[8:]
	_, ok := verifySignedCookie(tampered, testHMACKey)
	if ok {
		t.Error("expected verification to fail for tampered signature")
	}
}

func TestVerifySignedCookieWrongKey(t *testing.T) {
	signed := signCookie("my-value", testHMACKey)
	_, ok := verifySignedCookie(signed, []byte("wrong-key"))
	if ok {
		t.Error("expected verification to fail for wrong key")
	}
}

func TestVerifySignedCookieNoDot(t *testing.T) {
	_, ok := verifySignedCookie("noseparator", testHMACKey)
	if ok {
		t.Error("expected verification to fail for missing separator")
	}
}

func TestVerifySignedCookieEmptyString(t *testing.T) {
	_, ok := verifySignedCookie("", testHMACKey)
	if ok {
		t.Error("expected verification to fail for empty string")
	}
}

func TestVerifySignedCookieDotAtStart(t *testing.T) {
	_, ok := verifySignedCookie(".value", testHMACKey)
	if ok {
		t.Error("expected verification to fail when dot is at index 0")
	}
}

func TestAdminSessionNoCookie(t *testing.T) {
	handler := AdminSession(testPubKey, testHMACKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
	loc := w.Header().Get("Location")
	if loc != AdminLoginPath {
		t.Errorf("redirect location = %q, want %q", loc, AdminLoginPath)
	}
}

func TestAdminSessionEmptyCookieValue(t *testing.T) {
	handler := AdminSession(testPubKey, testHMACKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", http.NoBody)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: ""})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminSessionInvalidSignature(t *testing.T) {
	handler := AdminSession(testPubKey, testHMACKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", http.NoBody)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "badsig.badtoken"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminSessionExpiredToken(t *testing.T) {
	tok, err := token.GenerateAccessToken(
		testPrivKey, testKID, testIssuer, -1*time.Hour,
		uuid.New(), uuid.New(),
		"admin", "admin@test.com", false, "", "",
	)
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}

	signed := signCookie(tok, testHMACKey)

	handler := AdminSession(testPubKey, testHMACKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for expired token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", http.NoBody)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: signed})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminSessionValidToken(t *testing.T) {
	tok := generateTestToken(t)
	signed := signCookie(tok, testHMACKey)

	var gotUser *AuthenticatedUser
	handler := AdminSession(testPubKey, testHMACKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = GetAuthenticatedUser(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", http.NoBody)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: signed})
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
}

func TestAdminSessionSignedButGarbageToken(t *testing.T) {
	// Token that passes HMAC check but fails JWT verification
	signed := signCookie("not-a-jwt", testHMACKey)

	handler := AdminSession(testPubKey, testHMACKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", http.NoBody)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: signed})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestSetAdminSession(t *testing.T) {
	w := httptest.NewRecorder()
	SetAdminSession(w, "test-token", testHMACKey, 3600)

	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected session cookie to be set")
	}
	if !found.HttpOnly {
		t.Error("expected HttpOnly flag on session cookie")
	}
	if found.MaxAge != 3600 {
		t.Errorf("MaxAge = %d, want 3600", found.MaxAge)
	}

	// Verify the cookie value is properly signed
	val, ok := verifySignedCookie(found.Value, testHMACKey)
	if !ok {
		t.Fatal("expected cookie value to be validly signed")
	}
	if val != "test-token" {
		t.Errorf("cookie token = %q, want test-token", val)
	}
}

func TestClearAdminSession(t *testing.T) {
	w := httptest.NewRecorder()
	ClearAdminSession(w)

	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected session cookie to be set (for clearing)")
	}
	if found.MaxAge != -1 {
		t.Errorf("MaxAge = %d, want -1", found.MaxAge)
	}
	if found.Value != "" {
		t.Errorf("cookie value = %q, want empty", found.Value)
	}
}

func TestSetFlash(t *testing.T) {
	w := httptest.NewRecorder()
	SetFlash(w, "Login successful")

	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == flashCookieName {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected flash cookie to be set")
	}

	decoded, err := base64.URLEncoding.DecodeString(found.Value)
	if err != nil {
		t.Fatalf("failed to decode flash cookie: %v", err)
	}
	if string(decoded) != "Login successful" {
		t.Errorf("flash message = %q, want %q", string(decoded), "Login successful")
	}
}

func TestGetFlashReadsAndClears(t *testing.T) {
	encoded := base64.URLEncoding.EncodeToString([]byte("Hello flash"))
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: flashCookieName, Value: encoded})
	w := httptest.NewRecorder()

	msg := GetFlash(w, req)
	if msg != "Hello flash" {
		t.Errorf("GetFlash = %q, want %q", msg, "Hello flash")
	}

	// Check that the cookie is cleared
	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == flashCookieName {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected flash cookie to be cleared")
	}
	if found.MaxAge != -1 {
		t.Errorf("flash cookie MaxAge = %d, want -1", found.MaxAge)
	}
}

func TestGetFlashNoCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	msg := GetFlash(w, req)
	if msg != "" {
		t.Errorf("GetFlash = %q, want empty", msg)
	}
}

func TestGetFlashInvalidBase64(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: flashCookieName, Value: "not-valid-base64!!!"})
	w := httptest.NewRecorder()

	msg := GetFlash(w, req)
	if msg != "" {
		t.Errorf("GetFlash = %q, want empty for invalid base64", msg)
	}
}

func TestGetCSRFToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "my-csrf-token"})

	tok := GetCSRFToken(req)
	if tok != "my-csrf-token" {
		t.Errorf("GetCSRFToken = %q, want my-csrf-token", tok)
	}
}

func TestGetCSRFTokenNoCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	tok := GetCSRFToken(req)
	if tok != "" {
		t.Errorf("GetCSRFToken = %q, want empty", tok)
	}
}

func TestGenerateHMACKey(t *testing.T) {
	key, err := GenerateHMACKey()
	if err != nil {
		t.Fatalf("GenerateHMACKey error: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("key length = %d, want 32", len(key))
	}

	// Generate a second key and check they differ
	key2, err := GenerateHMACKey()
	if err != nil {
		t.Fatalf("GenerateHMACKey error: %v", err)
	}
	if bytes.Equal(key, key2) {
		t.Error("two generated keys should differ")
	}
}

func generateTestTokenWithRolesAdmin(t *testing.T, roles ...string) string {
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

func TestRequireAdminSessionAllowsAdmin(t *testing.T) {
	tok := generateTestTokenWithRolesAdmin(t, "admin")
	signed := signCookie(tok, testHMACKey)

	called := false
	handler := AdminSession(testPubKey, testHMACKey)(
		RequireAdminSession()(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: signed})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !called {
		t.Error("expected handler to be called for admin user")
	}
}

func TestRequireAdminSessionBlocksRegularUser(t *testing.T) {
	tok := generateTestTokenWithRolesAdmin(t, "viewer")
	signed := signCookie(tok, testHMACKey)

	handler := AdminSession(testPubKey, testHMACKey)(
		RequireAdminSession()(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("handler should not be called for non-admin user")
			}),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: signed})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireAdminSessionBlocksUserWithNoRoles(t *testing.T) {
	tok := generateTestTokenWithRolesAdmin(t)
	signed := signCookie(tok, testHMACKey)

	handler := AdminSession(testPubKey, testHMACKey)(
		RequireAdminSession()(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("handler should not be called for user with no roles")
			}),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: signed})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireAdminSessionNoUserRedirects(t *testing.T) {
	handler := RequireAdminSession()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
	loc := w.Header().Get("Location")
	if loc != AdminLoginPath {
		t.Errorf("redirect location = %q, want %q", loc, AdminLoginPath)
	}
}

func TestCSRFProtectAllowsGET(t *testing.T) {
	called := false
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/page", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler to be called for GET request")
	}
}

func TestCSRFProtectAllowsHEAD(t *testing.T) {
	called := false
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodHead, "/admin/page", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler to be called for HEAD request")
	}
}

func TestCSRFProtectAllowsOPTIONS(t *testing.T) {
	called := false
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodOptions, "/admin/page", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler to be called for OPTIONS request")
	}
}

func TestCSRFProtectSetsCSRFCookieOnGET(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/admin/page", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	cookies := w.Result().Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == csrfCookieName {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected CSRF cookie to be set on GET")
	}
	if found.HttpOnly {
		t.Error("CSRF cookie should not be HttpOnly")
	}
}

func TestCSRFProtectDoesNotOverwriteExistingCSRFCookie(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/admin/page", http.NoBody)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "existing-token"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == csrfCookieName {
			t.Error("should not overwrite existing CSRF cookie")
		}
	}
}

func TestCSRFProtectBlocksPOSTWithoutCookie(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodPost, "/admin/action", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFProtectBlocksPOSTWithMismatchedToken(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	form := url.Values{}
	form.Set(csrfFieldName, "wrong-token")
	req := httptest.NewRequest(http.MethodPost, "/admin/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "correct-token"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFProtectAllowsPOSTWithMatchingFormToken(t *testing.T) {
	called := false
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	csrfToken := "matching-csrf-token"
	form := url.Values{}
	form.Set(csrfFieldName, csrfToken)
	req := httptest.NewRequest(http.MethodPost, "/admin/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler to be called for valid CSRF token")
	}
}

func TestCSRFProtectAllowsPOSTWithXCSRFTokenHeader(t *testing.T) {
	called := false
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	csrfToken := "header-csrf-token"
	req := httptest.NewRequest(http.MethodPost, "/admin/action", http.NoBody)
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler to be called for valid X-CSRF-Token header")
	}
}

func TestCSRFProtectBlocksPOSTWithEmptyCookieValue(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodPost, "/admin/action", http.NoBody)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: ""})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFProtectBlocksPOSTWithNoFormTokenAndNoHeader(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodPost, "/admin/action", http.NoBody)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "some-token"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFProtectBlocksDELETE(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodDelete, "/admin/resource", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFProtectBlocksPUT(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodPut, "/admin/resource", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}
