package mfa

import (
	"strings"
	"testing"
)

func TestGenerateRecoveryCodes(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		expectErr bool
	}{
		{"generates 8 codes", 8, false},
		{"generates 1 code", 1, false},
		{"zero count errors", 0, true},
		{"negative count errors", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes, err := GenerateRecoveryCodes(tt.count)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GenerateRecoveryCodes(%d) error = %v", tt.count, err)
			}
			if len(codes) != tt.count {
				t.Errorf("got %d codes, want %d", len(codes), tt.count)
			}
		})
	}
}

func TestRecoveryCodeFormat(t *testing.T) {
	codes, err := GenerateRecoveryCodes(10)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes error = %v", err)
	}

	for i, code := range codes {
		parts := strings.Split(code, "-")
		if len(parts) != 2 {
			t.Errorf("code[%d] = %q: expected format XXXX-XXXX", i, code)
			continue
		}
		if len(parts[0]) != RecoveryCodeLength || len(parts[1]) != RecoveryCodeLength {
			t.Errorf("code[%d] = %q: part lengths = %d, %d; want %d, %d",
				i, code, len(parts[0]), len(parts[1]), RecoveryCodeLength, RecoveryCodeLength)
		}
	}
}

func TestRecoveryCodesUnique(t *testing.T) {
	codes, err := GenerateRecoveryCodes(100)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes error = %v", err)
	}

	seen := make(map[string]bool, len(codes))
	for _, code := range codes {
		if seen[code] {
			t.Errorf("duplicate code: %s", code)
		}
		seen[code] = true
	}
}

func TestHashAndVerifyRecoveryCode(t *testing.T) {
	code := "ABCD-1234"

	hash, err := HashRecoveryCode(code)
	if err != nil {
		t.Fatalf("HashRecoveryCode error = %v", err)
	}

	if hash == "" {
		t.Fatal("hash is empty")
	}
	if hash == code {
		t.Fatal("hash equals plaintext code")
	}

	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{"exact match", "ABCD-1234", true},
		{"lowercase match", "abcd-1234", true},
		{"no dash match", "ABCD1234", true},
		{"wrong code", "XXXX-9999", false},
		{"empty code", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyRecoveryCode(tt.input, hash)
			if got != tt.expect {
				t.Errorf("VerifyRecoveryCode(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}
