package oauth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

// RFC 7636 Appendix B test vector verifier.
const rfcTestVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

// ---------------------------------------------------------------------------
// PKCE edge cases
// ---------------------------------------------------------------------------

func TestPKCE_VerifierTooShort(t *testing.T) {
	// 42 chars — one below the minimum of 43
	short := strings.Repeat("a", 42)
	if ValidateCodeVerifier(short) {
		t.Errorf("expected verifier of length %d to be rejected", len(short))
	}
	// Verify ValidatePKCE also rejects it
	challenge := ComputeS256Challenge(strings.Repeat("a", 43))
	if ValidatePKCE(short, challenge) {
		t.Error("expected ValidatePKCE to reject short verifier")
	}
}

func TestPKCE_VerifierTooLong(t *testing.T) {
	// 129 chars — one above the maximum of 128
	long := strings.Repeat("b", 129)
	if ValidateCodeVerifier(long) {
		t.Errorf("expected verifier of length %d to be rejected", len(long))
	}
	if ValidatePKCE(long, "any-challenge") {
		t.Error("expected ValidatePKCE to reject long verifier")
	}
}

func TestPKCE_VerifierInvalidCharacters(t *testing.T) {
	invalid := []struct {
		name     string
		verifier string
	}{
		{"space in middle", "abcdefghijklmnopqrstuvwxyz0123456789ABCDE FG"},
		{"tab character", "abcdefghijklmnopqrstuvwxyz0123456789ABCDE\tFG"},
		{"non-ascii unicode", "abcdefghijklmnopqrstuvwxyz0123456789ABCDéFGH"},
		{"plus sign", "abcdefghijklmnopqrstuvwxyz0123456789ABCDE+FGH"},
		{"equals sign", "abcdefghijklmnopqrstuvwxyz0123456789ABCDE=FGH"},
		{"slash", "abcdefghijklmnopqrstuvwxyz0123456789ABCDE/FGH"},
		{"backslash", "abcdefghijklmnopqrstuvwxyz0123456789ABCDE\\FG"},
		{"emoji", "abcdefghijklmnopqrstuvwxyz0123456789ABCDE😎FG"},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			if ValidateCodeVerifierChars(tt.verifier) {
				t.Errorf("expected verifier with %s to be rejected", tt.name)
			}
		})
	}
}

func TestPKCE_VerifierValidCharacters(t *testing.T) {
	// All unreserved characters per RFC 7636
	valid := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	if !ValidateCodeVerifierChars(valid) {
		t.Error("expected valid verifier to be accepted")
	}

	// Verifier with tilde, hyphen, dot, underscore
	withSpecials := "abcdefghijklmnopqrstuvwxyz0123456789-._~abc"
	if !ValidateCodeVerifierChars(withSpecials) {
		t.Error("expected verifier with unreserved special chars to be accepted")
	}
}

func TestPKCE_S256ChallengeComputation(t *testing.T) {
	// Verify with a known verifier that the S256 computation matches manual SHA-256 + base64url
	verifier := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM_test"
	h := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(h[:])

	got := ComputeS256Challenge(verifier)
	if got != expected {
		t.Errorf("S256 mismatch: got %q, want %q", got, expected)
	}

	// Verify against RFC 7636 Appendix B test vector
	rfcVerifier := rfcTestVerifier
	rfcExpected := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if result := ComputeS256Challenge(rfcVerifier); result != rfcExpected {
		t.Errorf("RFC test vector failed: got %q, want %q", result, rfcExpected)
	}
}

func TestPKCE_S256NoPadding(t *testing.T) {
	// Verify the output uses raw URL encoding (no padding characters)
	verifier := strings.Repeat("x", 43)
	challenge := ComputeS256Challenge(verifier)
	if strings.Contains(challenge, "=") {
		t.Error("S256 challenge must not contain padding '='")
	}
	if strings.Contains(challenge, "+") || strings.Contains(challenge, "/") {
		t.Error("S256 challenge must use URL-safe base64 (no + or /)")
	}
}

func TestPKCE_WrongVerifierFails(t *testing.T) {
	verifier := rfcTestVerifier
	challenge := ComputeS256Challenge(verifier)

	// Same length but different content
	wrong := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXX"
	if ValidatePKCE(wrong, challenge) {
		t.Error("expected wrong verifier to fail PKCE validation")
	}

	// Off-by-one character
	offByOne := verifier[:len(verifier)-1] + "X"
	if ValidatePKCE(offByOne, challenge) {
		t.Error("expected off-by-one verifier to fail PKCE validation")
	}
}

func TestPKCE_BoundaryLengths(t *testing.T) {
	// Exactly 43 (minimum)
	minVerifier := strings.Repeat("a", 43)
	if !ValidateCodeVerifier(minVerifier) {
		t.Errorf("expected minimum-length verifier (%d) to be accepted", len(minVerifier))
	}

	// Exactly 128 (maximum)
	maxVerifier := strings.Repeat("z", 128)
	if !ValidateCodeVerifier(maxVerifier) {
		t.Errorf("expected maximum-length verifier (%d) to be accepted", len(maxVerifier))
	}

	// PKCE round-trip at boundaries
	challengeMin := ComputeS256Challenge(minVerifier)
	if !ValidatePKCE(minVerifier, challengeMin) {
		t.Error("expected PKCE validation to pass at min boundary")
	}

	challengeMax := ComputeS256Challenge(maxVerifier)
	if !ValidatePKCE(maxVerifier, challengeMax) {
		t.Error("expected PKCE validation to pass at max boundary")
	}
}

func TestPKCE_EmptyChallenge(t *testing.T) {
	verifier := strings.Repeat("a", 43)
	if ValidatePKCE(verifier, "") {
		t.Error("expected empty challenge to fail PKCE validation")
	}
}

// ---------------------------------------------------------------------------
// Scope parsing
// ---------------------------------------------------------------------------

func TestParseScopes_EmptyString(t *testing.T) {
	result := ParseScopes("")
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty scope string, got %v", result)
	}
}

func TestParseScopes_SingleScope(t *testing.T) {
	result := ParseScopes("openid")
	if len(result) != 1 || result[0] != "openid" {
		t.Errorf("expected [openid], got %v", result)
	}
}

func TestParseScopes_MultipleScopes(t *testing.T) {
	result := ParseScopes("openid profile email")
	expected := []string{"openid", "profile", "email"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d scopes, got %d: %v", len(expected), len(result), result)
	}
	for i, s := range expected {
		if result[i] != s {
			t.Errorf("scope[%d] = %q, want %q", i, result[i], s)
		}
	}
}

func TestParseScopes_DuplicateScopes(t *testing.T) {
	result := ParseScopes("openid profile openid email profile")
	expected := []string{"openid", "profile", "email"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d unique scopes, got %d: %v", len(expected), len(result), result)
	}
	for i, s := range expected {
		if result[i] != s {
			t.Errorf("scope[%d] = %q, want %q", i, result[i], s)
		}
	}
}

func TestParseScopes_ExtraWhitespace(t *testing.T) {
	// Multiple spaces between scopes
	result := ParseScopes("openid  profile   email")
	if len(result) != 3 {
		t.Errorf("expected 3 scopes, got %d: %v", len(result), result)
	}
}

func TestParseScopes_OnlySpaces(t *testing.T) {
	result := ParseScopes("   ")
	if len(result) != 0 {
		t.Errorf("expected empty slice for whitespace-only string, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// Scope token validation (RFC 6749 §3.3 characters)
// ---------------------------------------------------------------------------

func TestValidateScopeToken_ValidTokens(t *testing.T) {
	valid := []string{"openid", "profile", "email", "read:user", "api.access"}
	for _, s := range valid {
		if !ValidateScopeToken(s) {
			t.Errorf("expected scope %q to be valid", s)
		}
	}
}

func TestValidateScopeToken_InvalidCharacters(t *testing.T) {
	invalid := []struct {
		name  string
		scope string
	}{
		{"empty string", ""},
		{"space", "open id"},
		{"double quote", `open"id`},
		{"backslash", `open\id`},
		{"null byte", "open\x00id"},
		{"DEL character", "open\x7Fid"},
		{"non-ascii", "opénid"},
		{"control char", "open\x01id"},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			if ValidateScopeToken(tt.scope) {
				t.Errorf("expected scope %q to be invalid", tt.scope)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Scope validation (known scopes)
// ---------------------------------------------------------------------------

func TestValidateScopes_AllKnown(t *testing.T) {
	parsed, unknown := ValidateScopes("openid profile email offline_access")
	if len(unknown) != 0 {
		t.Errorf("expected no unknown scopes, got %v", unknown)
	}
	if len(parsed) != 4 {
		t.Errorf("expected 4 parsed scopes, got %d", len(parsed))
	}
}

func TestValidateScopes_UnknownScopes(t *testing.T) {
	_, unknown := ValidateScopes("openid custom_scope another_unknown")
	if len(unknown) != 2 {
		t.Errorf("expected 2 unknown scopes, got %d: %v", len(unknown), unknown)
	}
	if unknown[0] != "custom_scope" || unknown[1] != "another_unknown" {
		t.Errorf("unexpected unknown scopes: %v", unknown)
	}
}

func TestValidateScopes_EmptyString(t *testing.T) {
	parsed, unknown := ValidateScopes("")
	if len(parsed) != 0 {
		t.Errorf("expected no parsed scopes, got %v", parsed)
	}
	if len(unknown) != 0 {
		t.Errorf("expected no unknown scopes, got %v", unknown)
	}
}

func TestContainsOpenID(t *testing.T) {
	tests := []struct {
		scope string
		want  bool
	}{
		{"openid", true},
		{"openid profile", true},
		{"profile openid email", true},
		{"profile email", false},
		{"", false},
		{"openid_extended", false}, // not an exact match
	}
	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			if got := ContainsOpenID(tt.scope); got != tt.want {
				t.Errorf("ContainsOpenID(%q) = %v, want %v", tt.scope, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Grant type validation
// ---------------------------------------------------------------------------

func TestValidateGrantType_AllValid(t *testing.T) {
	valid := []string{"authorization_code", "refresh_token"}
	for _, gt := range valid {
		if !ValidateGrantType(gt) {
			t.Errorf("expected grant type %q to be valid", gt)
		}
	}
}

func TestValidateGrantType_UnknownRejected(t *testing.T) {
	invalid := []string{
		"",
		"implicit",
		"client_credentials",
		"password",
		"urn:ietf:params:oauth:grant-type:device_code",
		"AUTHORIZATION_CODE",     // case-sensitive
		"authorization_code ",    // trailing space
		" authorization_code",    // leading space
		"authorization_code\x00", // null byte
	}
	for _, gt := range invalid {
		t.Run(gt, func(t *testing.T) {
			if ValidateGrantType(gt) {
				t.Errorf("expected grant type %q to be rejected", gt)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Response type validation
// ---------------------------------------------------------------------------

func TestValidateResponseType_Valid(t *testing.T) {
	if !ValidateResponseType("code") {
		t.Error("expected 'code' response type to be valid")
	}
}

func TestValidateResponseType_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"token",
		"id_token",
		"code token",
		"code id_token",
		"CODE",  // case-sensitive
		"code ", // trailing space
	}
	for _, rt := range invalid {
		t.Run(rt, func(t *testing.T) {
			if ValidateResponseType(rt) {
				t.Errorf("expected response type %q to be rejected", rt)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Client authentication
// ---------------------------------------------------------------------------

func TestVerifyClientSecret_CorrectSecret(t *testing.T) {
	secret := "super-secret-value-12345"
	hash, err := HashClientSecret(secret)
	if err != nil {
		t.Fatalf("failed to hash secret: %v", err)
	}
	if !VerifyClientSecret(secret, hash) {
		t.Error("expected correct secret to verify successfully")
	}
}

func TestVerifyClientSecret_WrongSecret(t *testing.T) {
	secret := "super-secret-value-12345"
	hash, err := HashClientSecret(secret)
	if err != nil {
		t.Fatalf("failed to hash secret: %v", err)
	}
	if VerifyClientSecret("wrong-secret", hash) {
		t.Error("expected wrong secret to fail verification")
	}
}

func TestVerifyClientSecret_EmptySecret(t *testing.T) {
	hash, err := HashClientSecret("real-secret")
	if err != nil {
		t.Fatalf("failed to hash secret: %v", err)
	}
	if VerifyClientSecret("", hash) {
		t.Error("expected empty secret to fail verification")
	}
}

func TestVerifyClientSecret_EmptyHash(t *testing.T) {
	if VerifyClientSecret("some-secret", []byte{}) {
		t.Error("expected empty hash to fail verification")
	}
}

func TestVerifyClientSecret_NilHash(t *testing.T) {
	if VerifyClientSecret("some-secret", nil) {
		t.Error("expected nil hash to fail verification")
	}
}

func TestHashClientSecret_DifferentHashesForSameSecret(t *testing.T) {
	secret := "same-secret"
	hash1, err := HashClientSecret(secret)
	if err != nil {
		t.Fatalf("failed to hash: %v", err)
	}
	hash2, err := HashClientSecret(secret)
	if err != nil {
		t.Fatalf("failed to hash: %v", err)
	}
	// bcrypt produces different hashes each time due to salt
	if bytes.Equal(hash1, hash2) {
		t.Error("expected different bcrypt hashes for same secret (different salts)")
	}
	// But both should verify
	if !VerifyClientSecret(secret, hash1) || !VerifyClientSecret(secret, hash2) {
		t.Error("expected both hashes to verify against original secret")
	}
}

func TestConstantTimeEqual(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"abc", "abc", true},
		{"abc", "abd", false},
		{"abc", "ab", false},
		{"", "", true},
		{"a", "", false},
	}
	for _, tt := range tests {
		if got := ConstantTimeEqual(tt.a, tt.b); got != tt.want {
			t.Errorf("ConstantTimeEqual(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
