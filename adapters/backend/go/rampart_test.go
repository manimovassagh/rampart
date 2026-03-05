package rampart

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// helper: generate an RSA key pair and return the private key + JWKS JSON
func generateTestKeys(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	jwkKey, err := jwk.FromRaw(privKey.PublicKey)
	if err != nil {
		t.Fatalf("create JWK from public key: %v", err)
	}
	_ = jwkKey.Set(jwk.KeyIDKey, "test-key-1")
	_ = jwkKey.Set(jwk.AlgorithmKey, jwa.RS256)
	_ = jwkKey.Set(jwk.KeyUsageKey, "sig")

	set := jwk.NewSet()
	_ = set.AddKey(jwkKey)

	jwksJSON, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshal JWKS: %v", err)
	}

	return privKey, jwksJSON
}

// helper: create a signed JWT token string
func signToken(t *testing.T, privKey *rsa.PrivateKey, issuer string, claims map[string]interface{}) string {
	t.Helper()

	builder := jwt.New()
	_ = builder.Set(jwt.IssuerKey, issuer)
	_ = builder.Set(jwt.SubjectKey, "user-123")
	_ = builder.Set(jwt.IssuedAtKey, time.Now().Add(-1*time.Minute))
	_ = builder.Set(jwt.ExpirationKey, time.Now().Add(10*time.Minute))

	for k, v := range claims {
		_ = builder.Set(k, v)
	}

	jwkKey, err := jwk.FromRaw(privKey)
	if err != nil {
		t.Fatalf("create JWK from private key: %v", err)
	}
	_ = jwkKey.Set(jwk.KeyIDKey, "test-key-1")
	_ = jwkKey.Set(jwk.AlgorithmKey, jwa.RS256)

	signed, err := jwt.Sign(builder, jwt.WithKey(jwa.RS256, jwkKey))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return string(signed)
}

// helper: start a mock JWKS server
func startJWKSServer(t *testing.T, jwksJSON []byte) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksJSON)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// dummyHandler returns 200 with the claims sub in the body
func dummyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"sub": claims.Sub})
	})
}

func TestMissingAuthHeader(t *testing.T) {
	privKey, jwksJSON := generateTestKeys(t)
	_ = privKey
	srv := startJWKSServer(t, jwksJSON)

	middleware := NewMiddleware(Config{Issuer: srv.URL})
	handler := middleware(dummyHandler())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Error != "unauthorized" {
		t.Errorf("expected error 'unauthorized', got %q", errResp.Error)
	}
}

func TestInvalidToken(t *testing.T) {
	privKey, jwksJSON := generateTestKeys(t)
	_ = privKey
	srv := startJWKSServer(t, jwksJSON)

	middleware := NewMiddleware(Config{Issuer: srv.URL})
	handler := middleware(dummyHandler())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestValidToken(t *testing.T) {
	privKey, jwksJSON := generateTestKeys(t)
	srv := startJWKSServer(t, jwksJSON)

	token := signToken(t, privKey, srv.URL, map[string]interface{}{
		"email":              "alice@example.com",
		"email_verified":     true,
		"preferred_username": "alice",
		"org_id":             "org-1",
		"roles":              []string{"admin", "user"},
	})

	middleware := NewMiddleware(Config{Issuer: srv.URL})
	handler := middleware(dummyHandler())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestInvalidHeaderFormat(t *testing.T) {
	privKey, jwksJSON := generateTestKeys(t)
	_ = privKey
	srv := startJWKSServer(t, jwksJSON)

	middleware := NewMiddleware(Config{Issuer: srv.URL})
	handler := middleware(dummyHandler())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestRequireRolesWithMissingRoles(t *testing.T) {
	privKey, jwksJSON := generateTestKeys(t)
	srv := startJWKSServer(t, jwksJSON)

	token := signToken(t, privKey, srv.URL, map[string]interface{}{
		"email":              "bob@example.com",
		"preferred_username": "bob",
		"roles":              []string{"user"},
	})

	authMiddleware := NewMiddleware(Config{Issuer: srv.URL})
	roleMiddleware := RequireRoles("admin", "superadmin")

	handler := authMiddleware(roleMiddleware(dummyHandler()))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Error != "forbidden" {
		t.Errorf("expected error 'forbidden', got %q", errResp.Error)
	}
}

func TestRequireRolesWithSufficientRoles(t *testing.T) {
	privKey, jwksJSON := generateTestKeys(t)
	srv := startJWKSServer(t, jwksJSON)

	token := signToken(t, privKey, srv.URL, map[string]interface{}{
		"email":              "admin@example.com",
		"preferred_username": "admin",
		"roles":              []string{"admin", "user"},
	})

	authMiddleware := NewMiddleware(Config{Issuer: srv.URL})
	roleMiddleware := RequireRoles("admin")

	handler := authMiddleware(roleMiddleware(dummyHandler()))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestRequireRolesWithoutAuth(t *testing.T) {
	roleMiddleware := RequireRoles("admin")
	handler := roleMiddleware(dummyHandler())

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestClaimsFromContextNoValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	claims, ok := ClaimsFromContext(req.Context())
	if ok || claims != nil {
		t.Error("expected no claims from empty context")
	}
}
