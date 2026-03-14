package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/token"
)

// ---------------------------------------------------------------------------
// Auth middleware edge cases
// ---------------------------------------------------------------------------

func TestAuthMiddlewareBearerDoubleSpace(t *testing.T) {
	handler := Auth(testPubKey)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called for double-space Bearer header")
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer  double-space-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareBearerNoTokenValue(t *testing.T) {
	// "Bearer " with trailing space but no actual token
	handler := Auth(testPubKey)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called for empty token value")
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareTokenSignedWithWrongKey(t *testing.T) {
	// Generate a completely separate RSA key pair
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate wrong RSA key: %v", err)
	}

	tok, err := token.GenerateAccessToken(
		wrongKey, testKID, testIssuer, testIssuer, 15*time.Minute,
		uuid.New(), uuid.New(),
		"hacker", "hacker@evil.com", false, "Evil", "Hacker",
	)
	if err != nil {
		t.Fatalf("failed to generate token with wrong key: %v", err)
	}

	handler := Auth(testPubKey)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called for token signed with wrong key")
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	// Verify the response is valid JSON with the expected error
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp["error"] != "unauthorized" {
		t.Errorf("error = %q, want unauthorized", resp["error"])
	}
}

func TestAuthMiddlewareEmptyBearerPrefix(t *testing.T) {
	// Just the word "Bearer" with no space or token
	handler := Auth(testPubKey)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareResponseIsJSON(t *testing.T) {
	handler := Auth(testPubKey)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != contentTypeJSON {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Rate limiter edge cases
// ---------------------------------------------------------------------------

func TestRateLimiterBurstAllAtOnce(t *testing.T) {
	// Verify that sending exactly `limit` requests succeeds,
	// and the very next one is blocked (burst boundary).
	const limit = 10
	rl := NewRateLimiter(limit, time.Minute)
	defer rl.Close()

	ip := "10.99.99.99"
	for i := range limit {
		if !rl.Allow(ip) {
			t.Fatalf("request %d of %d should be allowed", i+1, limit)
		}
	}

	// The (limit+1)th request must be blocked
	if rl.Allow(ip) {
		t.Errorf("request %d should be blocked (limit is %d)", limit+1, limit)
	}

	// Second blocked request should also fail
	if rl.Allow(ip) {
		t.Error("subsequent request after limit should still be blocked")
	}
}

func TestRateLimiterMultipleIPsIndependent(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	defer rl.Close()

	ips := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"}

	// Each IP sends 3 requests (at the limit)
	for _, ip := range ips {
		for i := range 3 {
			if !rl.Allow(ip) {
				t.Fatalf("IP %s request %d should be allowed", ip, i+1)
			}
		}
	}

	// All IPs are now at their limit
	for _, ip := range ips {
		if rl.Allow(ip) {
			t.Errorf("IP %s should be blocked after exhausting limit", ip)
		}
	}
}

func TestRateLimiterMiddlewareDifferentIPsGetSeparateLimits(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	defer rl.Close()

	handler := rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First IP uses its quota
	req1 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req1.RemoteAddr = "192.168.1.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("first IP first request: status = %d, want 200", rec1.Code)
	}

	// First IP is now blocked
	rec1b := httptest.NewRecorder()
	handler.ServeHTTP(rec1b, req1)
	if rec1b.Code != http.StatusTooManyRequests {
		t.Errorf("first IP second request: status = %d, want 429", rec1b.Code)
	}

	// Second IP should still be allowed
	req2 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req2.RemoteAddr = "192.168.1.2:5678"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("second IP first request: status = %d, want 200", rec2.Code)
	}
}

func TestRateLimiterCloseIsIdempotent(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	// Calling Close multiple times should not panic
	rl.Close()
	rl.Close()
	rl.Close()
}

// ---------------------------------------------------------------------------
// Admin session edge cases
// ---------------------------------------------------------------------------

func TestAdminSessionTamperedCookieValue(t *testing.T) {
	// Create a valid signed cookie, then tamper with the JWT payload
	tok := generateTestToken(t)
	signed := signCookie(tok, testHMACKey)

	// Tamper by replacing a character in the middle of the signature
	tampered := "X" + signed[1:]

	handler := AdminSession(testPubKey, testHMACKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for tampered cookie")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", http.NoBody)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: tampered})
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

func TestAdminSessionCookieSignedWithWrongHMACKey(t *testing.T) {
	tok := generateTestToken(t)
	wrongKey := []byte("completely-different-hmac-key-xxx")
	signed := signCookie(tok, wrongKey)

	handler := AdminSession(testPubKey, testHMACKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for cookie signed with wrong HMAC key")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", http.NoBody)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: signed})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestAdminSessionExpiredTokenClearsSessionCookie(t *testing.T) {
	tok, err := token.GenerateAccessToken(
		testPrivKey, testKID, testIssuer, testIssuer, -1*time.Hour,
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

	// Verify the session cookie is cleared (MaxAge = -1)
	cookies := w.Result().Cookies()
	var cleared bool
	for _, c := range cookies {
		if c.Name == sessionCookieName && c.MaxAge == -1 {
			cleared = true
			break
		}
	}
	if !cleared {
		t.Error("expected session cookie to be cleared after expired token")
	}
}

// ---------------------------------------------------------------------------
// Request ID edge cases
// ---------------------------------------------------------------------------

func TestRequestIDGeneratesUniqueIDs(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	ids := make(map[string]bool)
	const iterations = 100

	for range iterations {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		id := w.Header().Get(HeaderRequestID)
		if id == "" {
			t.Fatal("expected non-empty request ID")
		}
		if ids[id] {
			t.Fatalf("duplicate request ID generated: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != iterations {
		t.Errorf("expected %d unique IDs, got %d", iterations, len(ids))
	}
}

func TestRequestIDContextAndHeaderMatch(t *testing.T) {
	var contextID string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID = GetRequestID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	headerID := w.Header().Get(HeaderRequestID)
	if headerID == "" {
		t.Fatal("expected X-Request-Id header to be set")
	}
	if contextID != headerID {
		t.Errorf("context ID %q does not match header ID %q", contextID, headerID)
	}
}

// ---------------------------------------------------------------------------
// Recovery middleware edge cases
// ---------------------------------------------------------------------------

func TestRecoveryPanicWithNilValue(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic((*string)(nil)) //nolint:nilpanic // intentional: testing recovery from nil pointer panic
	}))

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	// This should not crash the server
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRecoveryPanicWithStruct(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	type customError struct {
		Code    int
		Message string
	}

	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(customError{Code: 500, Message: "database connection lost"})
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/data", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	body := w.Body.String()
	if !strings.Contains(body, "internal_error") {
		t.Errorf("response body missing error code, got: %s", body)
	}
}

func TestRecoveryDoesNotLeakPanicDetails(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	secretMessage := "database password is s3cret!"
	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(secretMessage)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if strings.Contains(body, secretMessage) {
		t.Error("response body should not leak panic details to client")
	}
	if strings.Contains(body, "s3cret") {
		t.Error("response body should not leak sensitive information")
	}

	// But it should be in the log
	logOutput := buf.String()
	if !strings.Contains(logOutput, secretMessage) {
		t.Error("panic details should be logged server-side")
	}
}

// ---------------------------------------------------------------------------
// MetricsAuth edge cases
// ---------------------------------------------------------------------------

func TestMetricsAuthMissingHeader(t *testing.T) {
	handler := MetricsAuth("secret-token")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMetricsAuthWrongToken(t *testing.T) {
	handler := MetricsAuth("correct-token")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMetricsAuthMalformedHeader(t *testing.T) {
	handler := MetricsAuth("secret")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody)
	req.Header.Set("Authorization", "Token secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMetricsAuthValidToken(t *testing.T) {
	called := false
	handler := MetricsAuth("my-metrics-token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody)
	req.Header.Set("Authorization", "Bearer my-metrics-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler to be called for valid metrics token")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
