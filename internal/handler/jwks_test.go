package handler

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/manimovassagh/rampart/internal/signing"
)

func newTestKeyPair(t *testing.T) *signing.KeyPair {
	t.Helper()
	dir := t.TempDir()
	kp, err := signing.LoadOrGenerate(dir + "/test-key.pem")
	if err != nil {
		t.Fatalf("LoadOrGenerate error: %v", err)
	}
	return kp
}

func TestJWKSReturnsPublicKey(t *testing.T) {
	kp := newTestKeyPair(t)
	h := JWKSHandler(kp, noopLogger())

	req := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", http.NoBody)
	w := httptest.NewRecorder()

	h(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	cc := w.Header().Get("Cache-Control")
	if cc != "public, max-age=3600" {
		t.Errorf("Cache-Control = %q, want public, max-age=3600", cc)
	}

	var jwks struct {
		Keys []map[string]string `json:"keys"`
	}
	if err := json.NewDecoder(w.Body).Decode(&jwks); err != nil {
		t.Fatalf("failed to decode JWKS: %v", err)
	}

	if len(jwks.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(jwks.Keys))
	}

	key := jwks.Keys[0]
	if key["kty"] != "RSA" {
		t.Errorf("kty = %q, want RSA", key["kty"])
	}
	if key["alg"] != "RS256" {
		t.Errorf("alg = %q, want RS256", key["alg"])
	}
	if key["kid"] != kp.KID {
		t.Errorf("kid = %q, want %q", key["kid"], kp.KID)
	}
}

func TestJWKSKeyCanVerifyToken(t *testing.T) {
	kp := newTestKeyPair(t)
	h := JWKSHandler(kp, noopLogger())

	// Get JWKS
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	var jwks struct {
		Keys []map[string]string `json:"keys"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &jwks); err != nil {
		t.Fatalf("failed to decode JWKS: %v", err)
	}

	// Reconstruct public key from JWK
	nBytes, _ := base64.RawURLEncoding.DecodeString(jwks.Keys[0]["n"])
	eBytes, _ := base64.RawURLEncoding.DecodeString(jwks.Keys[0]["e"])
	reconstructed := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}

	// Sign a token with the private key
	claims := jwt.RegisteredClaims{Subject: "test-user"}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kp.KID
	signed, err := tok.SignedString(kp.PrivateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// Verify with reconstructed public key from JWKS
	parsed, err := jwt.Parse(signed, func(_ *jwt.Token) (any, error) {
		return reconstructed, nil
	}, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		t.Fatalf("failed to verify token with JWKS public key: %v", err)
	}
	if !parsed.Valid {
		t.Error("token should be valid")
	}
}

func TestJWKSIsDeterministic(t *testing.T) {
	kp := newTestKeyPair(t)
	h := JWKSHandler(kp, noopLogger())

	w1 := httptest.NewRecorder()
	h(w1, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	w2 := httptest.NewRecorder()
	h(w2, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	if w1.Body.String() != w2.Body.String() {
		t.Error("JWKS response should be deterministic")
	}
}
