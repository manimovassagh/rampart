package mfa

import (
	"encoding/base32"
	"testing"
	"time"
)

func TestGenerateSecret(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret() error = %v", err)
	}

	// Verify it's valid base32.
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		t.Fatalf("generated secret is not valid base32: %v", err)
	}

	if len(decoded) != SecretSize {
		t.Errorf("decoded secret length = %d, want %d", len(decoded), SecretSize)
	}

	// Two calls should produce different secrets.
	secret2, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret() second call error = %v", err)
	}
	if secret == secret2 {
		t.Error("two calls to GenerateSecret returned identical secrets")
	}
}

func TestGenerateQRCodeURI(t *testing.T) {
	uri := GenerateQRCodeURI("JBSWY3DPEHPK3PXP", "Rampart", "user@example.com")

	if uri == "" {
		t.Fatal("GenerateQRCodeURI returned empty string")
	}

	tests := []struct {
		name     string
		contains string
	}{
		{"scheme", "otpauth://totp/"},
		{"issuer label", "Rampart"},
		{"account", "user@example.com"},
		{"secret param", "secret=JBSWY3DPEHPK3PXP"},
		{"issuer param", "issuer=Rampart"},
		{"algorithm", "algorithm=SHA1"},
		{"digits", "digits=6"},
		{"period", "period=30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !contains(uri, tt.contains) {
				t.Errorf("URI %q does not contain %q", uri, tt.contains)
			}
		})
	}
}

func TestValidateCodeAt(t *testing.T) {
	// Use a known secret and time to produce a deterministic code.
	secret := "JBSWY3DPEHPK3PXP"
	fixedTime := time.Unix(1234567890, 0)

	// Generate the expected code at this time.
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		t.Fatalf("decoding test secret: %v", err)
	}
	counter := fixedTime.Unix() / Period
	expectedCode := generateCode(secretBytes, counter)

	tests := []struct {
		name   string
		code   string
		time   time.Time
		expect bool
	}{
		{"correct code at exact time", expectedCode, fixedTime, true},
		{"correct code one step ahead", expectedCode, fixedTime.Add(-Period * time.Second), true},
		{"correct code one step behind", expectedCode, fixedTime.Add(Period * time.Second), true},
		{"wrong code", "000000", fixedTime, false},
		{"too short code", "12345", fixedTime, false},
		{"too long code", "1234567", fixedTime, false},
		{"empty code", "", fixedTime, false},
		{"code two steps away", expectedCode, fixedTime.Add(2 * Period * time.Second), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateCodeAt(secret, tt.code, tt.time)
			if got != tt.expect {
				t.Errorf("ValidateCodeAt(%q, %q, %v) = %v, want %v",
					secret, tt.code, tt.time, got, tt.expect)
			}
		})
	}
}

func TestValidateCodeInvalidSecret(t *testing.T) {
	// An invalid base32 secret should not panic, just return false.
	got := ValidateCodeAt("not-valid-base32!!!", "123456", time.Now())
	if got {
		t.Error("ValidateCodeAt with invalid secret returned true, want false")
	}
}

func TestGenerateCodeDeterministic(t *testing.T) {
	// RFC 6238 test vector: SHA1, time = 59, secret = "12345678901234567890"
	// Expected TOTP: 287082
	secret := []byte("12345678901234567890")
	counter := int64(59) / Period // = 1
	code := generateCode(secret, counter)
	if code != "287082" {
		t.Errorf("RFC 6238 test vector: got %q, want %q", code, "287082")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
