package signing

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"sync"
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

// --- Edge-case tests ---

// 1. Key generation produces valid 4096-bit RSA keys
func TestGeneratedKeyIs4096Bits(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.pem")

	kp, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("LoadOrGenerate error: %v", err)
	}

	bits := kp.PrivateKey.N.BitLen()
	if bits != 4096 {
		t.Errorf("expected 4096-bit key, got %d bits", bits)
	}

	if err := kp.PrivateKey.Validate(); err != nil {
		t.Errorf("generated key fails validation: %v", err)
	}
}

// 2. Generated key can sign and verify
func TestSignAndVerify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.pem")

	kp, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("LoadOrGenerate error: %v", err)
	}

	message := []byte("test message for signing")
	hash := sha256.Sum256(message)

	sig, err := rsa.SignPKCS1v15(rand.Reader, kp.PrivateKey, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatalf("signing failed: %v", err)
	}

	err = rsa.VerifyPKCS1v15(kp.PublicKey, crypto.SHA256, hash[:], sig)
	if err != nil {
		t.Errorf("verification failed: %v", err)
	}

	// Verify with wrong message fails
	wrongHash := sha256.Sum256([]byte("wrong message"))
	err = rsa.VerifyPKCS1v15(kp.PublicKey, crypto.SHA256, wrongHash[:], sig)
	if err == nil {
		t.Error("expected verification to fail with wrong message")
	}
}

// 3. Key file permissions are restricted (0600)
func TestKeyFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.pem")

	_, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("LoadOrGenerate error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("expected file permissions 0600, got %04o", perm)
	}
}

// 4. Loading key from non-existent file generates a new key
func TestLoadFromNonExistentFileGeneratesKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.pem")

	kp, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("LoadOrGenerate should generate key for missing file: %v", err)
	}
	if kp == nil {
		t.Fatal("expected non-nil KeyPair")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected key file to be created")
	}
}

// 5. Loading key from empty file fails gracefully
func TestLoadFromEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.pem")

	if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadOrGenerate(path)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
	if got := err.Error(); got != "no PEM block found in signing key file" {
		t.Errorf("unexpected error message: %q", got)
	}
}

// 6. Loading key from corrupted PEM fails gracefully
func TestLoadFromCorruptedPEM(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		content string
	}{
		{"garbage_data", "not a pem at all!!!"},
		{"truncated_PEM", "-----BEGIN PRIVATE KEY-----\nMIIE"},
		{"valid_header_garbage_body", "-----BEGIN PRIVATE KEY-----\nYWJjZGVmZw==\n-----END PRIVATE KEY-----\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".pem")
			if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}

			_, err := LoadOrGenerate(path)
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

// 7. Loading key from wrong PEM type (EC key in RSA slot) fails
func TestLoadECKeyInRSASlotFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ec-key.pem")

	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating EC key: %v", err)
	}

	derBytes, err := x509.MarshalPKCS8PrivateKey(ecKey)
	if err != nil {
		t.Fatalf("marshaling EC key: %v", err)
	}

	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derBytes,
	})

	if err := os.WriteFile(path, pemData, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = LoadOrGenerate(path)
	if err == nil {
		t.Fatal("expected error when loading EC key as RSA")
	}
	if got := err.Error(); got != "signing key is not RSA" {
		t.Errorf("unexpected error: %q, want %q", got, "signing key is not RSA")
	}
}

// 8. Concurrent key loading is safe
func TestConcurrentKeyLoading(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.pem")

	// First generate the key so all goroutines load the same file
	_, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("initial generation error: %v", err)
	}

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	kids := make(chan string, goroutines)

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			kp, loadErr := LoadOrGenerate(path)
			if loadErr != nil {
				errs <- loadErr
				return
			}
			kids <- kp.KID
		}()
	}

	wg.Wait()
	close(errs)
	close(kids)

	for loadErr := range errs {
		t.Errorf("concurrent load error: %v", loadErr)
	}

	var firstKID string
	for kid := range kids {
		if firstKID == "" {
			firstKID = kid
		} else if kid != firstKID {
			t.Errorf("concurrent loads produced different KIDs: %q vs %q", firstKID, kid)
		}
	}
}

// 9. Key ID (kid) generation is deterministic for same key (thorough)
func TestKIDDeterministicMultipleCalls(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.pem")

	kp, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("LoadOrGenerate error: %v", err)
	}

	for i := range 100 {
		kid := computeKID(kp.PublicKey)
		if kid != kp.KID {
			t.Fatalf("KID changed on iteration %d: %q vs %q", i, kid, kp.KID)
		}
	}
}

// 10. Key ID changes for different keys
func TestKIDDiffersForDifferentKeys(t *testing.T) {
	dir := t.TempDir()

	kp1, err := LoadOrGenerate(filepath.Join(dir, "key1.pem"))
	if err != nil {
		t.Fatalf("first key error: %v", err)
	}

	kp2, err := LoadOrGenerate(filepath.Join(dir, "key2.pem"))
	if err != nil {
		t.Fatalf("second key error: %v", err)
	}

	if kp1.KID == kp2.KID {
		t.Error("different keys should produce different KIDs")
	}
}

// 11. Generated PEM can be parsed back
func TestGeneratedPEMRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.pem")

	kp1, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("LoadOrGenerate error: %v", err)
	}

	data, err := os.ReadFile(path) //nolint:gosec // test file using t.TempDir() path
	if err != nil {
		t.Fatalf("reading PEM file: %v", err)
	}

	block, rest := pem.Decode(data)
	if block == nil {
		t.Fatal("no PEM block found in generated file")
	}
	if block.Type != "PRIVATE KEY" {
		t.Errorf("PEM type = %q, want %q", block.Type, "PRIVATE KEY")
	}
	if len(rest) != 0 {
		t.Errorf("unexpected trailing data after PEM block: %d bytes", len(rest))
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("ParsePKCS8PrivateKey error: %v", err)
	}

	priv, ok := key.(*rsa.PrivateKey)
	if !ok {
		t.Fatal("parsed key is not RSA")
	}

	if priv.N.Cmp(kp1.PrivateKey.N) != 0 {
		t.Error("round-tripped N does not match")
	}
	if priv.D.Cmp(kp1.PrivateKey.D) != 0 {
		t.Error("round-tripped D does not match")
	}
	if priv.E != kp1.PrivateKey.E {
		t.Error("round-tripped E does not match")
	}
}

// 12. Verify the rsaKeyBits constant is 4096 (no short-key bypass)
func TestRSAKeyBitsConstant(t *testing.T) {
	if rsaKeyBits != 4096 {
		t.Errorf("rsaKeyBits = %d, want 4096", rsaKeyBits)
	}
}

// Additional edge cases

func TestLoadFromDirectoryPathFails(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadOrGenerate(dir)
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

func TestJWKSResponseContainsAllRequiredFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.pem")

	kp, err := LoadOrGenerate(path)
	if err != nil {
		t.Fatalf("LoadOrGenerate error: %v", err)
	}

	data, err := kp.JWKSResponse()
	if err != nil {
		t.Fatalf("JWKSResponse error: %v", err)
	}

	if !json.Valid(data) {
		t.Error("JWKSResponse did not produce valid JSON")
	}

	var jwks struct {
		Keys []map[string]string `json:"keys"`
	}
	if err := json.Unmarshal(data, &jwks); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	required := []string{"kty", "use", "alg", "kid", "n", "e"}
	for _, field := range required {
		if _, ok := jwks.Keys[0][field]; !ok {
			t.Errorf("JWKS key missing required field %q", field)
		}
	}
}

func TestBase64URLUintNoPadding(t *testing.T) {
	val := big.NewInt(65537)
	encoded := base64URLUint(val)

	for _, c := range encoded {
		if c == '=' {
			t.Error("base64URLUint should produce unpadded output")
		}
		if c == '+' || c == '/' {
			t.Error("base64URLUint should use URL-safe alphabet")
		}
	}

	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	result := new(big.Int).SetBytes(decoded)
	if result.Cmp(val) != 0 {
		t.Errorf("round-trip failed: got %s, want %s", result, val)
	}
}
