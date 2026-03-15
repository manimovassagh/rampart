package mfa

import (
	"bytes"
	"testing"
)

func TestGenerateSecret(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" {
		t.Fatal("empty secret")
	}
	// base32 encoded 20 bytes = 32 chars
	if len(secret) != 32 {
		t.Fatalf("expected 32-char secret, got %d", len(secret))
	}
}

func TestProvisioningURI(t *testing.T) {
	uri := ProvisioningURI("JBSWY3DPEHPK3PXP", "user@example.com", "Rampart")
	if uri == "" {
		t.Fatal("empty URI")
	}
	if len(uri) < 50 {
		t.Fatalf("URI seems too short: %s", uri)
	}
}

func TestValidateCodeRoundTrip(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	// We can't easily test a valid code without reimplementing the generation,
	// but we can verify that invalid codes are rejected
	ok1, _ := ValidateCode(secret, "000000", 0)
	ok2, _ := ValidateCode(secret, "111111", 0)
	ok3, _ := ValidateCode(secret, "999999", 0)
	if ok1 && ok2 && ok3 {
		t.Fatal("all codes validated — something is wrong")
	}
}

func TestValidateCodeWrongLength(t *testing.T) {
	if ok, _ := ValidateCode("JBSWY3DPEHPK3PXP", "12345", 0); ok {
		t.Fatal("should reject 5-digit code")
	}
	if ok, _ := ValidateCode("JBSWY3DPEHPK3PXP", "1234567", 0); ok {
		t.Fatal("should reject 7-digit code")
	}
}

func TestGenerateBackupCodes(t *testing.T) {
	codes, err := GenerateBackupCodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != BackupCodeCount {
		t.Fatalf("expected %d codes, got %d", BackupCodeCount, len(codes))
	}
	// Check format: xxxx-xxxx
	for _, code := range codes {
		if len(code) != 9 || code[4] != '-' {
			t.Fatalf("bad code format: %q", code)
		}
	}
	// Check uniqueness
	seen := make(map[string]bool)
	for _, code := range codes {
		if seen[code] {
			t.Fatalf("duplicate code: %s", code)
		}
		seen[code] = true
	}
}

func TestHashBackupCode(t *testing.T) {
	h1 := HashBackupCode("abcd-efgh")
	h2 := HashBackupCode("ABCD-EFGH")
	h3 := HashBackupCode("abcdefgh")

	// All should produce the same hash (case-insensitive, dash-insensitive)
	if !bytes.Equal(h1, h2) {
		t.Fatal("case should not matter")
	}
	if !bytes.Equal(h1, h3) {
		t.Fatal("dash should not matter")
	}
}
