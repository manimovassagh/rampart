package middleware

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
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
