package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateAuthorizationCode(t *testing.T) {
	code, err := GenerateAuthorizationCode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code == "" {
		t.Fatal("expected non-empty code")
	}

	// Should be unique
	code2, _ := GenerateAuthorizationCode()
	if code == code2 {
		t.Fatal("expected unique codes")
	}
}

func TestValidateCodeVerifier(t *testing.T) {
	tests := []struct {
		name     string
		verifier string
		want     bool
	}{
		{"valid 43 chars", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqr", true},
		{"valid 128 chars", strings.Repeat("a", 128), true},
		{"too short 42 chars", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop", false},
		{"empty", "", false},
		{"too long 129 chars", strings.Repeat("a", 129), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateCodeVerifier(tt.verifier)
			if got != tt.want {
				t.Errorf("ValidateCodeVerifier(%d chars) = %v, want %v", len(tt.verifier), got, tt.want)
			}
		})
	}
}

func TestComputeS256Challenge(t *testing.T) {
	// RFC 7636 Appendix B test vector
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	expected := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	got := ComputeS256Challenge(verifier)
	if got != expected {
		t.Errorf("ComputeS256Challenge() = %q, want %q", got, expected)
	}

	// Also verify manually
	h := sha256.Sum256([]byte(verifier))
	manual := base64.RawURLEncoding.EncodeToString(h[:])
	if got != manual {
		t.Errorf("result doesn't match manual computation")
	}
}

func TestValidatePKCE(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := ComputeS256Challenge(verifier)

	tests := []struct {
		name      string
		verifier  string
		challenge string
		want      bool
	}{
		{"valid pair", verifier, challenge, true},
		{"wrong verifier", "wrong-verifier-that-is-at-least-43-characters-long!!", challenge, false},
		{"wrong challenge", verifier, "wrong-challenge", false},
		{"empty verifier", "", challenge, false},
		{"too short verifier", "short", challenge, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePKCE(tt.verifier, tt.challenge)
			if got != tt.want {
				t.Errorf("ValidatePKCE() = %v, want %v", got, tt.want)
			}
		})
	}
}
