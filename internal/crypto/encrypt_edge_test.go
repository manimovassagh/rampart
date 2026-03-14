package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"sync"
	"testing"
	"time"
)

func edgeKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

// 1. Encrypt/decrypt with maximum size plaintext (1 MB).
func TestLargePlaintext1MB(t *testing.T) {
	enc, err := NewEncryptor(edgeKey(t))
	if err != nil {
		t.Fatal(err)
	}

	plain := make([]byte, 1<<20) // 1 MB
	for i := range plain {
		plain[i] = byte(i % 256)
	}
	original := string(plain)

	ct, err := enc.Encrypt(original)
	if err != nil {
		t.Fatalf("encrypt 1MB: %v", err)
	}
	if !IsEncrypted(ct) {
		t.Fatal("expected enc: prefix on large ciphertext")
	}

	pt, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatalf("decrypt 1MB: %v", err)
	}
	if pt != original {
		t.Fatal("round-trip failed for 1MB plaintext")
	}
}

// 2. Encrypt empty string (already tested but verify both directions explicitly).
func TestEncryptEmptyStringEdge(t *testing.T) {
	enc, err := NewEncryptor(edgeKey(t))
	if err != nil {
		t.Fatal(err)
	}

	ct, err := enc.Encrypt("")
	if err != nil {
		t.Fatalf("encrypt empty: %v", err)
	}
	if ct != "" {
		t.Fatalf("expected empty ciphertext, got %q", ct)
	}
	if IsEncrypted(ct) {
		t.Fatal("empty string should not be considered encrypted")
	}

	pt, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatalf("decrypt empty: %v", err)
	}
	if pt != "" {
		t.Fatalf("expected empty plaintext, got %q", pt)
	}
}

// 3. Encrypt with zero key (all-zero 32-byte key).
// AES accepts any 32-byte key including all-zeros — verify it still works.
func TestZeroKey(t *testing.T) {
	zeroKey := make([]byte, 32) // all zeros
	enc, err := NewEncryptor(zeroKey)
	if err != nil {
		t.Fatalf("zero key should be accepted by AES: %v", err)
	}

	ct, err := enc.Encrypt("hello")
	if err != nil {
		t.Fatalf("encrypt with zero key: %v", err)
	}

	pt, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatalf("decrypt with zero key: %v", err)
	}
	if pt != "hello" {
		t.Fatalf("round-trip with zero key: got %q", pt)
	}
}

// 3b. NewEncryptor with wrong key sizes.
func TestInvalidKeySizes(t *testing.T) {
	for _, size := range []int{0, 1, 15, 16, 24, 31, 33, 64, 128} {
		key := make([]byte, size)
		_, err := NewEncryptor(key)
		if err == nil {
			t.Fatalf("expected error for key length %d", size)
		}
	}
}

// 4. Decrypt with wrong key (should fail gracefully).
func TestDecryptWrongKey(t *testing.T) {
	keyA := edgeKey(t)
	keyB := edgeKey(t)

	encA, err := NewEncryptor(keyA)
	if err != nil {
		t.Fatal(err)
	}
	encB, err := NewEncryptor(keyB)
	if err != nil {
		t.Fatal(err)
	}

	ct, err := encA.Encrypt("secret-data")
	if err != nil {
		t.Fatal(err)
	}

	_, err = encB.Decrypt(ct)
	if err == nil {
		t.Fatal("decrypting with wrong key should fail")
	}
	if !strings.Contains(err.Error(), "decrypting") {
		t.Fatalf("expected 'decrypting' in error, got: %v", err)
	}
}

// 5. Decrypt corrupted ciphertext (flip bits in the middle).
func TestDecryptCorruptedCiphertext(t *testing.T) {
	enc, err := NewEncryptor(edgeKey(t))
	if err != nil {
		t.Fatal(err)
	}

	ct, err := enc.Encrypt("important-data")
	if err != nil {
		t.Fatal(err)
	}

	// Decode the base64 payload, corrupt it, re-encode
	b64 := strings.TrimPrefix(ct, prefix)
	raw, err := base64.RawStdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}

	// Flip a byte in the middle of the ciphertext (after the nonce)
	mid := len(raw) / 2
	raw[mid] ^= 0xFF

	corrupted := prefix + base64.RawStdEncoding.EncodeToString(raw)
	_, err = enc.Decrypt(corrupted)
	if err == nil {
		t.Fatal("decrypting corrupted ciphertext should fail")
	}
}

// 6. Decrypt truncated ciphertext.
func TestDecryptTruncatedCiphertext(t *testing.T) {
	enc, err := NewEncryptor(edgeKey(t))
	if err != nil {
		t.Fatal(err)
	}

	ct, err := enc.Encrypt("test-data")
	if err != nil {
		t.Fatal(err)
	}

	b64 := strings.TrimPrefix(ct, prefix)
	raw, err := base64.RawStdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}

	// Truncate to less than nonce size
	if len(raw) > 4 {
		tooShort := prefix + base64.RawStdEncoding.EncodeToString(raw[:4])
		_, err = enc.Decrypt(tooShort)
		if err == nil {
			t.Fatal("decrypting truncated (shorter than nonce) ciphertext should fail")
		}
		if !strings.Contains(err.Error(), "too short") {
			t.Fatalf("expected 'too short' in error, got: %v", err)
		}
	}

	// Truncate to nonce only (no actual ciphertext body)
	nonceOnly := prefix + base64.RawStdEncoding.EncodeToString(raw[:enc.gcm.NonceSize()])
	_, err = enc.Decrypt(nonceOnly)
	if err == nil {
		t.Fatal("decrypting nonce-only ciphertext should fail")
	}

	// Truncate part of the ciphertext body
	half := enc.gcm.NonceSize() + (len(raw)-enc.gcm.NonceSize())/2
	partial := prefix + base64.RawStdEncoding.EncodeToString(raw[:half])
	_, err = enc.Decrypt(partial)
	if err == nil {
		t.Fatal("decrypting partially truncated ciphertext should fail")
	}
}

// 7. Decrypt with modified nonce.
func TestDecryptModifiedNonce(t *testing.T) {
	enc, err := NewEncryptor(edgeKey(t))
	if err != nil {
		t.Fatal(err)
	}

	ct, err := enc.Encrypt("nonce-test")
	if err != nil {
		t.Fatal(err)
	}

	b64 := strings.TrimPrefix(ct, prefix)
	raw, err := base64.RawStdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}

	// Flip the first byte of the nonce
	raw[0] ^= 0xFF

	modified := prefix + base64.RawStdEncoding.EncodeToString(raw)
	_, err = enc.Decrypt(modified)
	if err == nil {
		t.Fatal("decrypting with modified nonce should fail")
	}
}

// 8. Concurrent encrypt/decrypt (goroutine safety).
func TestConcurrentEncryptDecrypt(t *testing.T) {
	enc, err := NewEncryptor(edgeKey(t))
	if err != nil {
		t.Fatal(err)
	}

	const goroutines = 50
	const iterations = 100
	var wg sync.WaitGroup
	errCh := make(chan error, goroutines*iterations)

	for g := range goroutines {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			for range iterations {
				original := "goroutine-data"
				ct, err := enc.Encrypt(original)
				if err != nil {
					errCh <- err
					return
				}
				pt, err := enc.Decrypt(ct)
				if err != nil {
					errCh <- err
					return
				}
				if pt != original {
					errCh <- err
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent error: %v", err)
	}
}

// 9. Key rotation: encrypt with key A, decrypt with key B (should fail).
func TestKeyRotationFails(t *testing.T) {
	keyA := edgeKey(t)
	keyB := edgeKey(t)

	encA, err := NewEncryptor(keyA)
	if err != nil {
		t.Fatal(err)
	}
	encB, err := NewEncryptor(keyB)
	if err != nil {
		t.Fatal(err)
	}

	ct, err := encA.Encrypt("key-rotation-test")
	if err != nil {
		t.Fatal(err)
	}

	_, err = encB.Decrypt(ct)
	if err == nil {
		t.Fatal("decrypting with different key should fail")
	}

	// Verify original key still works
	pt, err := encA.Decrypt(ct)
	if err != nil {
		t.Fatalf("original key should still decrypt: %v", err)
	}
	if pt != "key-rotation-test" {
		t.Fatalf("got %q, want %q", pt, "key-rotation-test")
	}
}

// 10. IsEncrypted with edge cases.
func TestIsEncryptedEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"empty string", "", false},
		{"just prefix", "enc:", true},
		{"prefix with valid data", "enc:AAAA", true},
		{"prefix with spaces", "enc: spaces after", true},
		{"random string", "hello-world", false},
		{"partial prefix enc", "en:", false},
		{"partial prefix en", "enc", false},
		{"prefix uppercase", "ENC:data", false},
		{"prefix with newline", "enc:\ndata", true},
		{"prefix embedded", "data:enc:more", false},
		{"just e", "e", false},
		{"unicode after prefix", "enc:日本語", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEncrypted(tt.in)
			if got != tt.want {
				t.Fatalf("IsEncrypted(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// 11. Base64 encoding edge cases in encrypted output.
func TestBase64EncodingEdgeCases(t *testing.T) {
	enc, err := NewEncryptor(edgeKey(t))
	if err != nil {
		t.Fatal(err)
	}

	// Test various plaintext lengths to exercise different base64 padding scenarios
	for _, size := range []int{1, 2, 3, 4, 5, 10, 15, 16, 17, 31, 32, 33, 100, 255, 256, 1000} {
		plain := make([]byte, size)
		for i := range plain {
			plain[i] = byte(i % 256)
		}
		original := string(plain)

		ct, err := enc.Encrypt(original)
		if err != nil {
			t.Fatalf("encrypt size %d: %v", size, err)
		}

		// Verify it uses RawStdEncoding (no padding characters)
		b64Part := strings.TrimPrefix(ct, prefix)
		if strings.Contains(b64Part, "=") {
			t.Fatalf("size %d: ciphertext contains padding '=' chars (should use raw encoding)", size)
		}

		// Verify the base64 is valid
		_, err = base64.RawStdEncoding.DecodeString(b64Part)
		if err != nil {
			t.Fatalf("size %d: invalid base64: %v", size, err)
		}

		// Round-trip
		pt, err := enc.Decrypt(ct)
		if err != nil {
			t.Fatalf("decrypt size %d: %v", size, err)
		}
		if pt != original {
			t.Fatalf("round-trip failed for size %d", size)
		}
	}
}

// 11b. Decrypt with invalid base64.
func TestDecryptInvalidBase64(t *testing.T) {
	enc, err := NewEncryptor(edgeKey(t))
	if err != nil {
		t.Fatal(err)
	}

	_, err = enc.Decrypt("enc:!!!not-valid-base64!!!")
	if err == nil {
		t.Fatal("decrypting invalid base64 should fail")
	}
	if !strings.Contains(err.Error(), "decoding") {
		t.Fatalf("expected 'decoding' in error, got: %v", err)
	}
}

// 12. Very long plaintext — verify performance doesn't degrade catastrophically.
func TestLargePlaintextPerformance(t *testing.T) {
	enc, err := NewEncryptor(edgeKey(t))
	if err != nil {
		t.Fatal(err)
	}

	// 4 MB plaintext
	plain := make([]byte, 4<<20)
	for i := range plain {
		plain[i] = byte(i % 256)
	}
	original := string(plain)

	start := time.Now()
	ct, err := enc.Encrypt(original)
	encDur := time.Since(start)
	if err != nil {
		t.Fatalf("encrypt 4MB: %v", err)
	}

	start = time.Now()
	pt, err := enc.Decrypt(ct)
	decDur := time.Since(start)
	if err != nil {
		t.Fatalf("decrypt 4MB: %v", err)
	}
	if pt != original {
		t.Fatal("round-trip failed for 4MB plaintext")
	}

	// Sanity: both should complete well under 5 seconds on any reasonable machine
	const maxDuration = 5 * time.Second
	if encDur > maxDuration {
		t.Fatalf("encryption took too long: %v", encDur)
	}
	if decDur > maxDuration {
		t.Fatalf("decryption took too long: %v", decDur)
	}

	t.Logf("4MB encrypt: %v, decrypt: %v", encDur, decDur)
}

// Additional edge case: decrypt with "enc:" prefix but empty payload.
func TestDecryptEmptyPayloadAfterPrefix(t *testing.T) {
	enc, err := NewEncryptor(edgeKey(t))
	if err != nil {
		t.Fatal(err)
	}

	// "enc:" with no base64 data — decodes to zero bytes, shorter than nonce
	_, err = enc.Decrypt("enc:")
	if err == nil {
		t.Fatal("decrypting 'enc:' with no payload should fail")
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Fatalf("expected 'too short' in error, got: %v", err)
	}
}

// Additional edge case: binary data with null bytes.
func TestBinaryPlaintextWithNullBytes(t *testing.T) {
	enc, err := NewEncryptor(edgeKey(t))
	if err != nil {
		t.Fatal(err)
	}

	original := "before\x00middle\x00after"
	ct, err := enc.Encrypt(original)
	if err != nil {
		t.Fatalf("encrypt binary: %v", err)
	}

	pt, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatalf("decrypt binary: %v", err)
	}
	if pt != original {
		t.Fatalf("round-trip with null bytes failed: got %q, want %q", pt, original)
	}
}

// Additional edge case: the same Encryptor produces unique ciphertexts (nonce uniqueness).
func TestNonceUniqueness(t *testing.T) {
	enc, err := NewEncryptor(edgeKey(t))
	if err != nil {
		t.Fatal(err)
	}

	seen := make(map[string]bool)
	const count = 1000
	for i := range count {
		ct, err := enc.Encrypt("same-plaintext")
		if err != nil {
			t.Fatal(err)
		}
		if seen[ct] {
			t.Fatalf("duplicate ciphertext at iteration %d", i)
		}
		seen[ct] = true
	}
}
