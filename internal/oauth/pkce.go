package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
)

const (
	// CodeVerifierMinLen is the minimum length for a PKCE code verifier (RFC 7636 §4.1).
	CodeVerifierMinLen = 43
	// CodeVerifierMaxLen is the maximum length for a PKCE code verifier (RFC 7636 §4.1).
	CodeVerifierMaxLen = 128
	// AuthorizationCodeBytes is the number of random bytes for an authorization code.
	AuthorizationCodeBytes = 32
)

// GenerateAuthorizationCode creates a cryptographically random URL-safe authorization code.
func GenerateAuthorizationCode() (string, error) {
	b := make([]byte, AuthorizationCodeBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating authorization code: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ValidateCodeVerifier checks that a code verifier meets RFC 7636 length requirements.
func ValidateCodeVerifier(verifier string) bool {
	n := len(verifier)
	return n >= CodeVerifierMinLen && n <= CodeVerifierMaxLen
}

// ComputeS256Challenge computes the S256 code challenge from a code verifier.
// challenge = BASE64URL(SHA256(verifier))
func ComputeS256Challenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// ValidatePKCE validates a code verifier against a stored code challenge using S256.
// Returns true if the verifier produces the expected challenge.
func ValidatePKCE(codeVerifier, storedChallenge string) bool {
	if !ValidateCodeVerifier(codeVerifier) {
		return false
	}
	computed := ComputeS256Challenge(codeVerifier)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(storedChallenge)) == 1
}
