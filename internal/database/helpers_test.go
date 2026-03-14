package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/crypto"
	"github.com/manimovassagh/rampart/internal/model"
)

// --- encryptToken / decryptToken ---

func TestEncryptTokenNilEncryptor(t *testing.T) {
	db := &DB{Encryptor: nil}

	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"plaintext token", "my-secret-token-12345"},
		{"token with special chars", "abc!@#$%^&*()_+-="},
		{"unicode token", "tok\u00e9n-\u00fc\u00f1\u00ee\u00e7\u00f6d\u00e9"},
	}

	for _, tc := range tests {
		t.Run("encrypt/"+tc.name, func(t *testing.T) {
			got, err := db.encryptToken(tc.input)
			if err != nil {
				t.Fatalf("encryptToken: %v", err)
			}
			if got != tc.input {
				t.Errorf("expected passthrough %q, got %q", tc.input, got)
			}
		})
		t.Run("decrypt/"+tc.name, func(t *testing.T) {
			got, err := db.decryptToken(tc.input)
			if err != nil {
				t.Fatalf("decryptToken: %v", err)
			}
			if got != tc.input {
				t.Errorf("expected passthrough %q, got %q", tc.input, got)
			}
		})
	}
}

func TestEncryptDecryptTokenWithEncryptor(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	enc, err := crypto.NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}
	db := &DB{Encryptor: enc}

	tests := []struct {
		name  string
		input string
	}{
		{"simple token", "my-token"},
		{"empty string", ""},
		{"long token", "this-is-a-very-long-token-that-exceeds-typical-lengths-and-should-still-work-correctly-12345"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := db.encryptToken(tc.input)
			if err != nil {
				t.Fatalf("encryptToken: %v", err)
			}

			// Empty input should pass through as-is
			if tc.input == "" {
				if encrypted != "" {
					t.Errorf("expected empty passthrough, got %q", encrypted)
				}
				return
			}

			// Non-empty should be encrypted (prefixed with "enc:")
			if encrypted == tc.input {
				t.Error("expected encrypted output to differ from input")
			}
			if !crypto.IsEncrypted(encrypted) {
				t.Errorf("expected encrypted string to have enc: prefix, got %q", encrypted)
			}

			// Decrypt should round-trip
			decrypted, err := db.decryptToken(encrypted)
			if err != nil {
				t.Fatalf("decryptToken: %v", err)
			}
			if decrypted != tc.input {
				t.Errorf("round-trip failed: got %q, want %q", decrypted, tc.input)
			}
		})
	}
}

func TestDecryptTokenPlaintextPassthrough(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	enc, err := crypto.NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}
	db := &DB{Encryptor: enc}

	// Plaintext values (without "enc:" prefix) should pass through unchanged
	// even when an Encryptor is configured
	plaintext := "not-encrypted-value"
	got, err := db.decryptToken(plaintext)
	if err != nil {
		t.Fatalf("decryptToken: %v", err)
	}
	if got != plaintext {
		t.Errorf("expected plaintext passthrough %q, got %q", plaintext, got)
	}
}

// --- decryptSocialAccount ---

func TestDecryptSocialAccountNilEncryptor(t *testing.T) {
	db := &DB{Encryptor: nil}
	sa := &model.SocialAccount{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
	}

	err := db.decryptSocialAccount(sa)
	if err != nil {
		t.Fatalf("decryptSocialAccount: %v", err)
	}
	if sa.AccessToken != "access-token-123" {
		t.Errorf("access_token: got %q, want %q", sa.AccessToken, "access-token-123")
	}
	if sa.RefreshToken != "refresh-token-456" {
		t.Errorf("refresh_token: got %q, want %q", sa.RefreshToken, "refresh-token-456")
	}
}

func TestDecryptSocialAccountEmptyTokens(t *testing.T) {
	db := &DB{Encryptor: nil}
	sa := &model.SocialAccount{
		AccessToken:  "",
		RefreshToken: "",
	}

	err := db.decryptSocialAccount(sa)
	if err != nil {
		t.Fatalf("decryptSocialAccount: %v", err)
	}
	if sa.AccessToken != "" {
		t.Errorf("access_token: got %q, want empty", sa.AccessToken)
	}
	if sa.RefreshToken != "" {
		t.Errorf("refresh_token: got %q, want empty", sa.RefreshToken)
	}
}

func TestDecryptSocialAccountWithEncryptor(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 10)
	}
	enc, err := crypto.NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}
	db := &DB{Encryptor: enc}

	// Encrypt tokens first
	encAccess, err := db.encryptToken("real-access-token")
	if err != nil {
		t.Fatalf("encryptToken: %v", err)
	}
	encRefresh, err := db.encryptToken("real-refresh-token")
	if err != nil {
		t.Fatalf("encryptToken: %v", err)
	}

	sa := &model.SocialAccount{
		AccessToken:  encAccess,
		RefreshToken: encRefresh,
	}

	err = db.decryptSocialAccount(sa)
	if err != nil {
		t.Fatalf("decryptSocialAccount: %v", err)
	}
	if sa.AccessToken != "real-access-token" {
		t.Errorf("access_token: got %q, want %q", sa.AccessToken, "real-access-token")
	}
	if sa.RefreshToken != "real-refresh-token" {
		t.Errorf("refresh_token: got %q, want %q", sa.RefreshToken, "real-refresh-token")
	}
}

// --- queryCtx edge cases ---

func TestQueryCtxCancelledParent(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	parentCancel() // cancel immediately

	ctx, cancel := queryCtx(parent)
	defer cancel()

	// The derived context should also be done since parent is cancelled
	select {
	case <-ctx.Done():
		// expected
	default:
		t.Error("expected context to be done when parent is cancelled")
	}
}

func TestQueryCtxDefaultTimeout(t *testing.T) {
	ctx, cancel := queryCtx(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline")
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		t.Fatal("expected positive remaining time")
	}
	if remaining > defaultQueryTimeout {
		t.Errorf("expected remaining <= %v, got %v", defaultQueryTimeout, remaining)
	}

	// Should be close to defaultQueryTimeout (within 100ms tolerance)
	if remaining < defaultQueryTimeout-100*time.Millisecond {
		t.Errorf("expected remaining close to %v, got %v", defaultQueryTimeout, remaining)
	}
}

// --- Constants ---

func TestConstants(t *testing.T) {
	if defaultMaxConns != 25 {
		t.Errorf("defaultMaxConns: got %d, want 25", defaultMaxConns)
	}
	if defaultMinConns != 2 {
		t.Errorf("defaultMinConns: got %d, want 2", defaultMinConns)
	}
	if connectTimeout != 10*time.Second {
		t.Errorf("connectTimeout: got %v, want 10s", connectTimeout)
	}
	if defaultQueryTimeout != 5*time.Second {
		t.Errorf("defaultQueryTimeout: got %v, want 5s", defaultQueryTimeout)
	}
	if pgUniqueViolation != "23505" {
		t.Errorf("pgUniqueViolation: got %q, want %q", pgUniqueViolation, "23505")
	}
	if defaultOrgSlug != "default" {
		t.Errorf("defaultOrgSlug: got %q, want %q", defaultOrgSlug, "default")
	}
}

// --- DB struct zero-value ---

func TestDBStructZeroValue(t *testing.T) {
	db := &DB{}

	if db.Pool != nil {
		t.Error("expected Pool to be nil on zero-value DB")
	}
	if db.Encryptor != nil {
		t.Error("expected Encryptor to be nil on zero-value DB")
	}

	// Close should be safe on zero-value
	db.Close() // should not panic

	// Ping should return an error
	err := db.Ping(context.Background())
	if err == nil {
		t.Error("expected error pinging zero-value DB")
	}
}

// --- PasswordResetToken struct ---

func TestPasswordResetTokenStruct(t *testing.T) {
	token := PasswordResetToken{}

	// Verify zero values
	if token.ID != (uuid.UUID{}) {
		t.Error("expected zero-value UUID")
	}
	if token.Used {
		t.Error("expected Used to be false by default")
	}
	if !token.ExpiresAt.IsZero() {
		t.Error("expected ExpiresAt to be zero")
	}
	if !token.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be zero")
	}
}

// --- UserConsent struct ---

func TestUserConsentStruct(t *testing.T) {
	consent := UserConsent{
		ClientID: "test-client-id",
		Scopes:   "openid profile email",
	}

	if consent.ClientID != "test-client-id" {
		t.Errorf("client_id: got %q, want %q", consent.ClientID, "test-client-id")
	}
	if consent.Scopes != "openid profile email" {
		t.Errorf("scopes: got %q, want %q", consent.Scopes, "openid profile email")
	}
	if !consent.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be zero")
	}
	if !consent.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be zero")
	}
}

// --- containsError helper ---

func TestContainsErrorNilErr(t *testing.T) {
	if containsError(nil, nil) {
		t.Error("expected false for nil err")
	}
}

// --- parseRedirectURIs edge cases ---

func TestParseRedirectURIsCarriageReturn(t *testing.T) {
	// Windows-style line endings
	got := parseRedirectURIs("https://a.com/cb\r\nhttps://b.com/cb")
	if len(got) != 2 {
		t.Fatalf("expected 2 URIs, got %d: %v", len(got), got)
	}
}

func TestParseRedirectURIsSingleWithNewline(t *testing.T) {
	got := parseRedirectURIs("https://a.com/cb\n")
	if len(got) != 1 {
		t.Fatalf("expected 1 URI, got %d: %v", len(got), got)
	}
	if got[0] != "https://a.com/cb" {
		t.Errorf("got %q, want %q", got[0], "https://a.com/cb")
	}
}

// --- generateClientID ---

func TestGenerateClientIDLength(t *testing.T) {
	id := generateClientID()
	if len(id) != 32 {
		t.Errorf("expected 32-char hex string, got %d chars", len(id))
	}
}

func TestGenerateClientIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 50; i++ {
		id := generateClientID()
		if ids[id] {
			t.Fatalf("duplicate ID generated: %q", id)
		}
		ids[id] = true
	}
}

// --- ValidateRedirectURI additional edge cases ---

func TestValidateRedirectURISchemeVariations(t *testing.T) {
	client := &model.OAuthClient{
		RedirectURIs: []string{"http://localhost:3000/cb"},
	}

	// Same host but different scheme should not match
	if ValidateRedirectURI(client, "https://localhost:3000/cb") {
		t.Error("expected https scheme not to match http")
	}

	// Different port should not match
	if ValidateRedirectURI(client, "http://localhost:3001/cb") {
		t.Error("expected different port not to match")
	}

	// Fragment should not match
	if ValidateRedirectURI(client, "http://localhost:3000/cb#fragment") {
		t.Error("expected URI with fragment not to match")
	}
}

func TestValidateRedirectURICustomScheme(t *testing.T) {
	// Mobile apps often use custom schemes
	client := &model.OAuthClient{
		RedirectURIs: []string{"myapp://callback"},
	}

	if !ValidateRedirectURI(client, "myapp://callback") {
		t.Error("expected custom scheme to match")
	}
	if ValidateRedirectURI(client, "myapp://other") {
		t.Error("expected different path not to match")
	}
}
