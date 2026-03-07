package crypto

import (
	"crypto/rand"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

func TestRoundTrip(t *testing.T) {
	enc, err := NewEncryptor(testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	original := "my-secret-token-12345"
	ciphertext, err := enc.Encrypt(original)
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext == original {
		t.Fatal("ciphertext should differ from plaintext")
	}
	if !IsEncrypted(ciphertext) {
		t.Fatal("ciphertext should have enc: prefix")
	}
	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != original {
		t.Fatalf("got %q, want %q", decrypted, original)
	}
}

func TestEmptyString(t *testing.T) {
	enc, err := NewEncryptor(testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	ct, err := enc.Encrypt("")
	if err != nil {
		t.Fatal(err)
	}
	if ct != "" {
		t.Fatal("empty input should return empty output")
	}
	pt, err := enc.Decrypt("")
	if err != nil {
		t.Fatal(err)
	}
	if pt != "" {
		t.Fatal("empty decrypt should return empty")
	}
}

func TestPlaintextPassthrough(t *testing.T) {
	enc, err := NewEncryptor(testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	// Un-prefixed values are returned as-is (backwards compat)
	pt, err := enc.Decrypt("some-plaintext-token")
	if err != nil {
		t.Fatal(err)
	}
	if pt != "some-plaintext-token" {
		t.Fatalf("plaintext passthrough failed: got %q", pt)
	}
}

func TestBadKeyLength(t *testing.T) {
	_, err := NewEncryptor([]byte("too-short"))
	if err == nil {
		t.Fatal("expected error for bad key length")
	}
}

func TestUniqueCiphertexts(t *testing.T) {
	enc, err := NewEncryptor(testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	ct1, _ := enc.Encrypt("same")
	ct2, _ := enc.Encrypt("same")
	if ct1 == ct2 {
		t.Fatal("two encryptions of the same plaintext should produce different ciphertexts")
	}
}
