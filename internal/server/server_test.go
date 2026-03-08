package server

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/manimovassagh/rampart/internal/middleware"
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

var testPubKey *rsa.PublicKey

func init() {
	block, _ := pem.Decode([]byte(testRSAKeyPEM))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic("failed to parse test RSA key: " + err.Error())
	}
	testPubKey = &key.PublicKey
}

func TestNewRouterMiddlewareChain(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	r := NewRouter(logger, []string{"http://localhost:3000"}, false)

	RegisterHealthRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"alive"}`))
	}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("healthz status = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Header().Get(middleware.HeaderRequestID) == "" {
		t.Error("expected X-Request-Id header from middleware")
	}
}

func TestNewRouterNotFound(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	r := NewRouter(logger, []string{"http://localhost:3000"}, false)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestNewServer(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	r := NewRouter(logger, []string{"http://localhost:3000"}, false)
	s := New(":0", r, logger)

	if s.httpServer.ReadTimeout != readTimeout {
		t.Errorf("ReadTimeout = %v, want %v", s.httpServer.ReadTimeout, readTimeout)
	}
	if s.httpServer.WriteTimeout != writeTimeout {
		t.Errorf("WriteTimeout = %v, want %v", s.httpServer.WriteTimeout, writeTimeout)
	}
	if s.httpServer.IdleTimeout != idleTimeout {
		t.Errorf("IdleTimeout = %v, want %v", s.httpServer.IdleTimeout, idleTimeout)
	}
}

func TestServerStartAndShutdown(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"http://localhost:3000"}, false)

	RegisterHealthRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"alive"}`))
	}, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	// Use port 0 for OS-assigned free port
	srv := New("127.0.0.1:0", r, logger)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Graceful shutdown
	if err := srv.Shutdown(); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}

	// Start should have returned nil (ErrServerClosed is swallowed)
	if err := <-errCh; err != nil {
		t.Fatalf("start returned unexpected error: %v", err)
	}
}

func TestNewRouterCORSHeaders(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"http://localhost:3000"}, false)

	RegisterHealthRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Preflight OPTIONS request
	req := httptest.NewRequest(http.MethodOptions, "/healthz", http.NoBody)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected Access-Control-Allow-Origin header from CORS middleware")
	}
}

func TestNewRouterReadyzEndpoint(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"http://localhost:3000"}, false)

	RegisterHealthRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("readyz status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestNewServerAddr(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, nil, false)
	s := New(":9090", r, logger)

	if s.httpServer.Addr != ":9090" {
		t.Errorf("Addr = %q, want :9090", s.httpServer.Addr)
	}
	if s.httpServer.Handler == nil {
		t.Error("Handler is nil")
	}
	if s.logger == nil {
		t.Error("logger is nil")
	}
}

func TestNewRouterMultipleOrigins(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"http://localhost:3000", "http://localhost:5173"}, false)

	RegisterHealthRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test with second origin
	req := httptest.NewRequest(http.MethodOptions, "/healthz", http.NoBody)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected CORS header for second allowed origin")
	}
}

func TestRegisterAuthRoutes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"*"}, false)

	called := false
	RegisterAuthRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/register", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Error("register handler was not called")
	}
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestRegisterLoginRoutes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"*"}, false)

	routes := map[string]bool{}
	loginH := func(w http.ResponseWriter, _ *http.Request) {
		routes["login"] = true
		w.WriteHeader(http.StatusOK)
	}
	refreshH := func(w http.ResponseWriter, _ *http.Request) {
		routes["refresh"] = true
		w.WriteHeader(http.StatusOK)
	}
	logoutH := func(w http.ResponseWriter, _ *http.Request) {
		routes["logout"] = true
		w.WriteHeader(http.StatusOK)
	}

	RegisterLoginRoutes(r, loginH, refreshH, logoutH, nil)

	tests := []struct {
		method string
		path   string
		key    string
	}{
		{http.MethodPost, "/login", "login"},
		{http.MethodPost, "/token/refresh", "refresh"},
		{http.MethodPost, "/logout", "logout"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if !routes[tt.key] {
			t.Errorf("%s %s: handler was not called", tt.method, tt.path)
		}
	}
}

func TestRegisterOAuthRoutes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"*"}, false)

	authorizeCall := 0
	tokenCall := 0
	revokeCall := 0

	consentCall := 0

	RegisterOAuthRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		authorizeCall++
		w.WriteHeader(http.StatusOK)
	}, func(w http.ResponseWriter, _ *http.Request) {
		consentCall++
		w.WriteHeader(http.StatusOK)
	}, func(w http.ResponseWriter, _ *http.Request) {
		tokenCall++
		w.WriteHeader(http.StatusOK)
	}, func(w http.ResponseWriter, _ *http.Request) {
		revokeCall++
		w.WriteHeader(http.StatusOK)
	}, nil)

	// GET /oauth/authorize
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if authorizeCall != 1 {
		t.Errorf("GET /oauth/authorize: handler called %d times, want 1", authorizeCall)
	}

	// POST /oauth/authorize
	req = httptest.NewRequest(http.MethodPost, "/oauth/authorize", http.NoBody)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if authorizeCall != 2 {
		t.Errorf("POST /oauth/authorize: handler called %d times, want 2", authorizeCall)
	}

	// POST /oauth/consent
	req = httptest.NewRequest(http.MethodPost, "/oauth/consent", http.NoBody)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if consentCall != 1 {
		t.Errorf("POST /oauth/consent: handler called %d times, want 1", consentCall)
	}

	// POST /oauth/token
	req = httptest.NewRequest(http.MethodPost, "/oauth/token", http.NoBody)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if tokenCall != 1 {
		t.Errorf("POST /oauth/token: handler called %d times, want 1", tokenCall)
	}

	// POST /oauth/revoke
	req = httptest.NewRequest(http.MethodPost, "/oauth/revoke", http.NoBody)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if revokeCall != 1 {
		t.Errorf("POST /oauth/revoke: handler called %d times, want 1", revokeCall)
	}
}

func TestRegisterOIDCRoutes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"*"}, false)

	discoveryCall := 0
	jwksCall := 0

	RegisterOIDCRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		discoveryCall++
		w.WriteHeader(http.StatusOK)
	}, func(w http.ResponseWriter, _ *http.Request) {
		jwksCall++
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if discoveryCall != 1 {
		t.Error("discovery handler was not called")
	}

	req = httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", http.NoBody)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if jwksCall != 1 {
		t.Error("jwks handler was not called")
	}
}

func TestRegisterHealthRoutesWrongMethod(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"*"}, false)

	RegisterHealthRoutes(r, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/healthz", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /healthz status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestServerShutdownWithoutStart(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, nil, false)
	s := New(":0", r, logger)

	// Shutdown without Start should not error — the underlying http.Server.Shutdown handles it
	if err := s.Shutdown(); err != nil {
		t.Fatalf("Shutdown without Start returned error: %v", err)
	}
}

func TestTimeoutConstants(t *testing.T) {
	if readTimeout != 10*time.Second {
		t.Errorf("readTimeout = %v, want 10s", readTimeout)
	}
	if writeTimeout != 30*time.Second {
		t.Errorf("writeTimeout = %v, want 30s", writeTimeout)
	}
	if idleTimeout != 60*time.Second {
		t.Errorf("idleTimeout = %v, want 60s", idleTimeout)
	}
	if shutdownTimeout != 10*time.Second {
		t.Errorf("shutdownTimeout = %v, want 10s", shutdownTimeout)
	}
}

func TestRegisterProtectedRoutes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"*"}, false)

	RegisterProtectedRoutes(r, testPubKey, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"user":"me"}`))
	})

	// Without token, should get 401
	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("GET /me without token: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// stubAdminEndpoints implements AdminEndpoints for route registration tests.
type stubAdminEndpoints struct{}

func (s *stubAdminEndpoints) Stats(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminEndpoints) ListUsers(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminEndpoints) CreateUser(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminEndpoints) GetUser(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminEndpoints) UpdateUser(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminEndpoints) DeleteUser(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminEndpoints) ResetPassword(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminEndpoints) ListSessions(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminEndpoints) RevokeSessions(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestRegisterAdminRoutes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"*"}, false)

	RegisterAdminRoutes(r, testPubKey, &stubAdminEndpoints{})

	// All admin routes should require auth — should get 401 without token
	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/stats"},
		{http.MethodGet, "/api/v1/admin/users"},
		{http.MethodPost, "/api/v1/admin/users"},
		{http.MethodGet, "/api/v1/admin/users/some-id"},
		{http.MethodPut, "/api/v1/admin/users/some-id"},
		{http.MethodDelete, "/api/v1/admin/users/some-id"},
		{http.MethodPost, "/api/v1/admin/users/some-id/reset-password"},
		{http.MethodGet, "/api/v1/admin/users/some-id/sessions"},
		{http.MethodDelete, "/api/v1/admin/users/some-id/sessions"},
	}

	for _, tt := range paths {
		req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, w.Code, http.StatusUnauthorized)
		}
	}
}

// stubOrgEndpoints implements OrgEndpoints for route registration tests.
type stubOrgEndpoints struct{}

func (s *stubOrgEndpoints) ListOrgs(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubOrgEndpoints) CreateOrg(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubOrgEndpoints) GetOrg(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubOrgEndpoints) UpdateOrg(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubOrgEndpoints) DeleteOrg(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubOrgEndpoints) GetOrgSettings(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubOrgEndpoints) UpdateOrgSettings(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestRegisterOrgRoutes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"*"}, false)

	RegisterOrgRoutes(r, testPubKey, &stubOrgEndpoints{})

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/organizations"},
		{http.MethodPost, "/api/v1/admin/organizations"},
		{http.MethodGet, "/api/v1/admin/organizations/org-1"},
		{http.MethodPut, "/api/v1/admin/organizations/org-1"},
		{http.MethodDelete, "/api/v1/admin/organizations/org-1"},
		{http.MethodGet, "/api/v1/admin/organizations/org-1/settings"},
		{http.MethodPut, "/api/v1/admin/organizations/org-1/settings"},
	}

	for _, tt := range paths {
		req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, w.Code, http.StatusUnauthorized)
		}
	}
}

func TestRegisterExportImportRoutes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"*"}, false)

	RegisterExportImportRoutes(r, testPubKey,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
	)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/organizations/org-1/export"},
		{http.MethodPost, "/api/v1/admin/organizations/org-1/import"},
	}

	for _, tt := range paths {
		req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, w.Code, http.StatusUnauthorized)
		}
	}
}

// stubAdminLoginEndpoints implements AdminLoginEndpoints for tests.
type stubAdminLoginEndpoints struct{}

func (s *stubAdminLoginEndpoints) Login(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminLoginEndpoints) Callback(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminLoginEndpoints) Logout(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// stubAdminConsoleEndpoints implements AdminConsoleEndpoints for tests.
type stubAdminConsoleEndpoints struct{}

func (s *stubAdminConsoleEndpoints) Dashboard(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) ListUsersPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) CreateUserPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) CreateUserAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) UserDetailPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) UpdateUserAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) DeleteUserAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) ResetPasswordAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) RevokeSessionsAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) ListOrgsPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) CreateOrgPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) CreateOrgAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) OrgDetailPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) UpdateOrgAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) UpdateOrgSettingsAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) DeleteOrgAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) ListClientsPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) CreateClientPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) CreateClientAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) ClientDetailPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) UpdateClientAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) DeleteClientAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) RegenerateSecretAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) ListRolesPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) CreateRolePage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) CreateRoleAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) RoleDetailPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) UpdateRoleAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) DeleteRoleAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) AssignRoleAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) UnassignRoleAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) ListEventsPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) ListSessionsPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) RevokeSessionAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) RevokeAllSessionsAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) ListGroupsPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) CreateGroupPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) CreateGroupAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) GroupDetailPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) UpdateGroupAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) DeleteGroupAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) AddGroupMemberAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) RemoveGroupMemberAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) AssignGroupRoleAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) UnassignGroupRoleAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) ExportOrgAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) ImportOrgPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) ImportOrgAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) OIDCPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) SocialProvidersPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (s *stubAdminConsoleEndpoints) UpdateSocialProviderAction(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestRegisterAdminConsoleRoutes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r := NewRouter(logger, []string{"*"}, false)

	hmacKey := []byte("test-hmac-key-for-csrf-testing-only")
	staticHandler := http.FileServer(http.Dir(t.TempDir()))

	RegisterAdminConsoleRoutes(r, testPubKey, hmacKey, staticHandler, &stubAdminLoginEndpoints{}, &stubAdminConsoleEndpoints{})

	// GET /admin (no trailing slash) should redirect to /admin/
	req := httptest.NewRequest(http.MethodGet, "/admin", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("GET /admin: status = %d, want %d", w.Code, http.StatusMovedPermanently)
	}
	if loc := w.Header().Get("Location"); loc != "/admin/" {
		t.Errorf("GET /admin: Location = %q, want %q", loc, "/admin/")
	}

	// Public routes should be accessible (200)
	publicPaths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/login"},
		{http.MethodGet, "/admin/callback"},
	}

	for _, tt := range publicPaths {
		req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, w.Code, http.StatusOK)
		}
	}

	// Protected routes should reject without session
	protectedPaths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/"},
		{http.MethodGet, "/admin/users"},
		{http.MethodGet, "/admin/organizations"},
		{http.MethodGet, "/admin/roles"},
		{http.MethodGet, "/admin/sessions"},
		{http.MethodGet, "/admin/events"},
		{http.MethodGet, "/admin/clients"},
		{http.MethodGet, "/admin/groups"},
		{http.MethodGet, "/admin/oidc"},
	}

	for _, tt := range protectedPaths {
		req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Without session cookie, should get redirect (302/303) or unauthorized (401/403)
		if w.Code == http.StatusOK {
			t.Errorf("%s %s: status = %d, expected non-200 for unauthenticated request", tt.method, tt.path, w.Code)
		}
	}
}
