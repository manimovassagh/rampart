package oauth

import (
	"crypto/subtle"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// SupportedGrantTypes lists grant types per RFC 6749 + refresh_token.
var SupportedGrantTypes = map[string]bool{
	"authorization_code": true,
	"refresh_token":      true,
}

// SupportedResponseTypes lists the OAuth 2.0 response types accepted by the server.
var SupportedResponseTypes = map[string]bool{
	"code": true,
}

// KnownScopes are the scopes recognized by the server.
var KnownScopes = map[string]bool{
	"openid":         true,
	"profile":        true,
	"email":          true,
	"offline_access": true,
}

// codeVerifierCharRegexp matches only unreserved characters per RFC 7636 §4.1:
// [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"
var codeVerifierCharRegexp = regexp.MustCompile(`^[A-Za-z0-9\-._~]+$`)

// scopeTokenRegexp matches a valid scope token per RFC 6749 §3.3:
// %x21 / %x23-5B / %x5D-7E  (printable ASCII except " and \)
var scopeTokenRegexp = regexp.MustCompile(`^[\x21\x23-\x5B\x5D-\x7E]+$`)

// ValidateCodeVerifierChars checks that a code verifier contains only valid
// unreserved characters per RFC 7636 §4.1.
func ValidateCodeVerifierChars(verifier string) bool {
	if verifier == "" {
		return false
	}
	return codeVerifierCharRegexp.MatchString(verifier)
}

// ValidateGrantType returns true if the grant type is supported.
func ValidateGrantType(grantType string) bool {
	return SupportedGrantTypes[grantType]
}

// ValidateResponseType returns true if the response type is supported.
func ValidateResponseType(responseType string) bool {
	return SupportedResponseTypes[responseType]
}

// ParseScopes splits a space-delimited scope string into individual scope tokens,
// deduplicating and skipping empty entries. Returns the unique scopes in order.
func ParseScopes(scopeStr string) []string {
	parts := strings.Split(scopeStr, " ")
	seen := make(map[string]bool, len(parts))
	var result []string
	for _, s := range parts {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// ValidateScopeToken checks that a single scope token contains only valid characters
// per RFC 6749 §3.3.
func ValidateScopeToken(scope string) bool {
	if scope == "" {
		return false
	}
	return scopeTokenRegexp.MatchString(scope)
}

// ValidateScopes parses a scope string and checks that all scopes are known.
// Returns the parsed scopes and a list of unknown scopes.
func ValidateScopes(scopeStr string) (parsed, unknown []string) {
	parsed = ParseScopes(scopeStr)
	for _, s := range parsed {
		if !KnownScopes[s] {
			unknown = append(unknown, s)
		}
	}
	return parsed, unknown
}

// ContainsOpenID returns true if the scope string includes "openid".
func ContainsOpenID(scopeStr string) bool {
	return slices.Contains(ParseScopes(scopeStr), "openid")
}

// HashClientSecret hashes a client secret using bcrypt.
func HashClientSecret(secret string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
}

// VerifyClientSecret compares a plain secret against a bcrypt hash.
// Returns true if they match.
func VerifyClientSecret(secret string, hash []byte) bool {
	return bcrypt.CompareHashAndPassword(hash, []byte(secret)) == nil
}

// ConstantTimeEqual compares two strings in constant time.
func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
