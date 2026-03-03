package signing

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrGenerateCreatesNewKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-key.pem")

	kp, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("LoadOrGenerate error: %v", err)
	}

	if kp.PrivateKey == nil {
		t.Fatal("expected non-nil private key")
	}
	if kp.PublicKey == nil {
		t.Fatal("expected non-nil public key")
	}
	if kp.KID == "" {
		t.Fatal("expected non-empty KID")
	}

	// File should exist
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("key file not created: %v", err)
	}
}

func TestLoadOrGenerateLoadsExistingKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-key.pem")

	// Generate first
	kp1, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("first LoadOrGenerate error: %v", err)
	}

	// Load existing
	kp2, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("second LoadOrGenerate error: %v", err)
	}

	// Keys should be identical
	if kp1.KID != kp2.KID {
		t.Errorf("KID mismatch: %q vs %q", kp1.KID, kp2.KID)
	}
	if kp1.PrivateKey.D.Cmp(kp2.PrivateKey.D) != 0 {
		t.Error("private keys differ after reload")
	}
}

func TestLoadOrGenerateInvalidPEM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-key.pem")

	if err := os.WriteFile(path, []byte("not a pem file"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadOrGenerate(path)
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestLoadOrGenerateUnreadableDir(t *testing.T) {
	// Path to a directory that doesn't exist and can't be written to
	path := "/nonexistent-dir-12345/key.pem"

	_, err := LoadOrGenerate(path)
	if err == nil {
		t.Fatal("expected error for unwritable path")
	}
}

func TestJWK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-key.pem")

	kp, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("LoadOrGenerate error: %v", err)
	}

	jwk := kp.JWK()

	if jwk["kty"] != "RSA" {
		t.Errorf("kty = %q, want RSA", jwk["kty"])
	}
	if jwk["use"] != "sig" {
		t.Errorf("use = %q, want sig", jwk["use"])
	}
	if jwk["alg"] != "RS256" {
		t.Errorf("alg = %q, want RS256", jwk["alg"])
	}
	if jwk["kid"] != kp.KID {
		t.Errorf("kid = %q, want %q", jwk["kid"], kp.KID)
	}

	// Verify n and e can be decoded back to the original public key
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk["n"])
	if err != nil {
		t.Fatalf("failed to decode n: %v", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk["e"])
	if err != nil {
		t.Fatalf("failed to decode e: %v", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	reconstructed := &rsa.PublicKey{N: n, E: int(e.Int64())}
	if reconstructed.N.Cmp(kp.PublicKey.N) != 0 {
		t.Error("reconstructed N doesn't match original")
	}
	if reconstructed.E != kp.PublicKey.E {
		t.Errorf("reconstructed E = %d, want %d", reconstructed.E, kp.PublicKey.E)
	}
}

func TestJWKSResponse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-key.pem")

	kp, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("LoadOrGenerate error: %v", err)
	}

	data, err := kp.JWKSResponse()
	if err != nil {
		t.Fatalf("JWKSResponse error: %v", err)
	}

	var jwks struct {
		Keys []map[string]string `json:"keys"`
	}
	if err := json.Unmarshal(data, &jwks); err != nil {
		t.Fatalf("failed to unmarshal JWKS: %v", err)
	}

	if len(jwks.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(jwks.Keys))
	}
	if jwks.Keys[0]["kid"] != kp.KID {
		t.Errorf("kid = %q, want %q", jwks.Keys[0]["kid"], kp.KID)
	}
}

func TestKIDDeterministic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-key.pem")

	kp, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("LoadOrGenerate error: %v", err)
	}

	kid1 := computeKID(kp.PublicKey)
	kid2 := computeKID(kp.PublicKey)

	if kid1 != kid2 {
		t.Errorf("KID not deterministic: %q vs %q", kid1, kid2)
	}
}
