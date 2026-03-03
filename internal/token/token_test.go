package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "this-is-a-test-secret-that-is-at-least-32-bytes-long"

func TestGenerateAndVerifyAccessToken(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	ttl := 15 * time.Minute

	signed, err := GenerateAccessToken(testSecret, ttl, userID, orgID, "admin", "admin@test.com", true, "Admin", "User")
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}
	if signed == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := VerifyAccessToken(testSecret, signed)
	if err != nil {
		t.Fatalf("VerifyAccessToken error: %v", err)
	}

	if claims.Subject != userID.String() {
		t.Errorf("sub = %q, want %q", claims.Subject, userID.String())
	}
	if claims.OrgID != orgID {
		t.Errorf("org_id = %v, want %v", claims.OrgID, orgID)
	}
	if claims.PreferredUsername != "admin" {
		t.Errorf("preferred_username = %q, want admin", claims.PreferredUsername)
	}
	if claims.Email != "admin@test.com" {
		t.Errorf("email = %q, want admin@test.com", claims.Email)
	}
	if !claims.EmailVerified {
		t.Error("email_verified = false, want true")
	}
	if claims.GivenName != "Admin" {
		t.Errorf("given_name = %q, want Admin", claims.GivenName)
	}
	if claims.FamilyName != "User" {
		t.Errorf("family_name = %q, want User", claims.FamilyName)
	}
}

func TestVerifyAccessTokenWrongSecret(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	signed, err := GenerateAccessToken(testSecret, 15*time.Minute, userID, orgID, "admin", "admin@test.com", false, "", "")
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}

	_, err = VerifyAccessToken("wrong-secret-that-is-at-least-32-bytes", signed)
	if err == nil {
		t.Fatal("expected error verifying with wrong secret")
	}
}

func TestVerifyAccessTokenExpired(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	// Create token with negative TTL (already expired)
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
		},
		OrgID:            orgID,
		PreferredUsername: "admin",
		Email:            "admin@test.com",
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("signing error: %v", err)
	}

	_, err = VerifyAccessToken(testSecret, signed)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyAccessTokenMalformed(t *testing.T) {
	_, err := VerifyAccessToken(testSecret, "not-a-jwt")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	tok1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken error: %v", err)
	}
	if len(tok1) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("refresh token length = %d, want 64", len(tok1))
	}

	tok2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken error: %v", err)
	}
	if tok1 == tok2 {
		t.Error("two refresh tokens should not be identical")
	}
}
