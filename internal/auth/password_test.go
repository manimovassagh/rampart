package auth

import (
	"strings"
	"testing"
)

func TestHashPasswordProducesPHCFormat(t *testing.T) {
	hash, err := HashPassword("correcthorsebatterystaple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Errorf("hash does not start with $argon2id$v=19$, got %q", hash[:30])
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Errorf("hash has %d parts, want 6", len(parts))
	}
}

func TestHashPasswordUniqueSalts(t *testing.T) {
	h1, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("first hash error = %v", err)
	}

	h2, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("second hash error = %v", err)
	}

	if h1 == h2 {
		t.Error("two hashes of the same password should differ (unique salts)")
	}
}

func TestVerifyPasswordCorrect(t *testing.T) {
	password := "S3cure!Pass#2024"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !ok {
		t.Error("VerifyPassword() = false, want true for correct password")
	}
}

func TestVerifyPasswordWrong(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if ok {
		t.Error("VerifyPassword() = true, want false for wrong password")
	}
}

func TestVerifyPasswordInvalidFormat(t *testing.T) {
	_, err := VerifyPassword("password", "not-a-valid-hash")
	if err == nil {
		t.Error("VerifyPassword() should return error for invalid hash format")
	}
}

func TestVerifyPasswordInvalidBase64(t *testing.T) {
	_, err := VerifyPassword("password", "$argon2id$v=19$m=65536,t=3,p=4$!!!invalid!!!$!!!invalid!!!")
	if err == nil {
		t.Error("VerifyPassword() should return error for invalid base64")
	}
}
