package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// testRSAKeyPEM is a 2048-bit RSA key used only in tests.
const testRSAKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAvN8Ex780DiM6xO5PgniD7BbnTEGx1IkX1LE0EbrGrZJHcVbX
IiUbxBcnAMl/PqPtpS02pf0IgaGPM1DgO10eNcGxRvUcw/H0hbOEgMFIch71egvD
d/Ag8m18vO0MaoSh7xBlJSIfgRLCpyoWwghurFuMViwMcst6Cg35W8+IOCL7KOkj
OdFWIT7baffJ2w7LGq0i3/TlSmoUNVF+sZzM2vj4QMC6T7bUI9ISx9KP1wvxAz99
c1PoSi1bu2e/Yz3CyXeg00Z1BVWNGEcM98iaajTMGP1QmqsavNMjO72Ub+1XpyQ/
ve36uznlaqHhBZqrtTI+YugLtYIRq3etuI2HmwIDAQABAoIBAB9gzeKBmZxfrfvZ
u8vpScGHbJX2tByjShpD9mqbpTZg/w2NZ+B8WciSMCCpWUKG6YxvnoylJSykMq5L
2XUDW2mC7HjlcAn9wKoV0QWzFt4e1pmYKrlaY57jIb4hg9aOgni9OJCawrEm9L/g
9jb2P6zS6NXIK6lGtNfGyo6+Q9tPa2nMF86xrzscKuT8hLq2B4YN3jdL5tCIsfO/
IcnDwL6eCw+sjjeKfsEXful+9JZSAoKr1sukeW+xSE26hZwhxmwyHDPST/D5QkJo
NvbikjKpRRWwuUTondPbpJ3d2C6vIYVXtJjdzE9RdfKDPeuTpRRbgr/bf9LLO9E6
9k6gEYECgYEA2hHOT3Wm+WnVTILaCh4Z0/qeMrpBVcUgNNfaedXdYS4MND2TY2Wr
cd2IUGYvjFJs2neattXYijR7dC9i3j40wYTE3ak1S8rjVdTjF9eoV73zfWjGLUoT
xKTidmxhaWixJJxOQXoYVxwumsebCoRJLQqs8DNauaA3HzHpRSKIm5MCgYEA3bkR
YndjBzUrWFPZnhY89DUvwTCPrbMyHSUCpRiPlcDkQwgTnqvEgutKUtEKUvWDOu0d
4DVjK/P2LwwL8tuA82WaTOZjFTZ3yGNPC+gJeamKS4PgQviEdldLU53VKQpy7kgg
bxwW+aY+hKpVo+9RerdMJRn2SUhiHaHWdedQuNkCgYEApU9UN5Y3wuEAyiRzx7Gz
4Kce39OkDbIGzShIvY1raezvYXbAUVxUUFggqtob92LQk/iRN0L7CSHp6FS3vUQo
1/6fAm3wMgmWto1QrdVVD1a2y33upYx/WdWoux9D5RVxHBDFngtBgl+h0MG5/Yn0
swlhuiEkCI2025gJfthD+LMCgYAD6u85tC5VxES9zM19k5sEHaR4X2lKgm4SQcMo
M6Tl2oCuBoiCNzrDrXCkwfjSum/VLLdobMkRz7+72RSk9+fxZQwy66c4irvXGJoe
9bylH6/H4c6moEmG5cf49EL99KdPOosIK5DkXGGiangU63efGXoI9cp6RQMmzuNB
NhMhEQKBgQDF+33l7x8rxERo1x0E/nwaPr4kISE2RbQSxXL7UARAg457+ol/bs36
O4bU15NoXuwZ1ShWIKMWxLVcEysw7pDHs3xaiokx1SIrn9nVkaOb9iK8B8qgCQ2o
W/xlOTOB+SaEdBMOGPE8JlFPydshVuTUj5wMzN4GejazNxtEXbbGaQ==
-----END RSA PRIVATE KEY-----`

var (
	testPrivKey *rsa.PrivateKey
	testPubKey  *rsa.PublicKey
)

func init() {
	block, _ := pem.Decode([]byte(testRSAKeyPEM))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic("failed to parse test RSA key: " + err.Error())
	}
	testPrivKey = key
	testPubKey = &key.PublicKey
}

const (
	testKID    = "test-kid-123"
	testIssuer = "http://localhost:8080"
)

func TestGenerateAndVerifyAccessToken(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	ttl := 15 * time.Minute

	signed, err := GenerateAccessToken(testPrivKey, testKID, testIssuer, testIssuer, ttl, userID, orgID, "admin", "admin@test.com", true, "Admin", "User")
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}
	if signed == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := VerifyAccessToken(testPubKey, signed)
	if err != nil {
		t.Fatalf("VerifyAccessToken error: %v", err)
	}

	if claims.Subject != userID.String() {
		t.Errorf("sub = %q, want %q", claims.Subject, userID.String())
	}
	if claims.Issuer != testIssuer {
		t.Errorf("iss = %q, want %q", claims.Issuer, testIssuer)
	}
	aud, _ := claims.GetAudience()
	if len(aud) != 1 || aud[0] != testIssuer {
		t.Errorf("aud = %v, want [%s]", aud, testIssuer)
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

func TestGenerateAccessTokenIncludesKIDHeader(t *testing.T) {
	signed, err := GenerateAccessToken(testPrivKey, testKID, testIssuer, testIssuer, 15*time.Minute, uuid.New(), uuid.New(), "admin", "admin@test.com", false, "", "")
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}

	// Parse without verification to inspect the header
	parser := jwt.NewParser()
	tok, _, err := parser.ParseUnverified(signed, &Claims{})
	if err != nil {
		t.Fatalf("ParseUnverified error: %v", err)
	}

	kid, ok := tok.Header["kid"].(string)
	if !ok || kid != testKID {
		t.Errorf("kid header = %v, want %q", tok.Header["kid"], testKID)
	}

	alg, ok := tok.Header["alg"].(string)
	if !ok || alg != "RS256" {
		t.Errorf("alg header = %v, want RS256", tok.Header["alg"])
	}
}

func TestVerifyAccessTokenWrongKey(t *testing.T) {
	signed, err := GenerateAccessToken(testPrivKey, testKID, testIssuer, testIssuer, 15*time.Minute, uuid.New(), uuid.New(), "admin", "admin@test.com", false, "", "")
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}

	otherKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		t.Fatalf("failed to generate other key: %v", err)
	}

	_, err = VerifyAccessToken(&otherKey.PublicKey, signed)
	if err == nil {
		t.Fatal("expected error verifying with wrong key")
	}
}

func TestVerifyAccessTokenExpired(t *testing.T) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
		},
		OrgID:             uuid.New(),
		PreferredUsername: "admin",
		Email:             "admin@test.com",
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(testPrivKey)
	if err != nil {
		t.Fatalf("signing error: %v", err)
	}

	_, err = VerifyAccessToken(testPubKey, signed)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyAccessTokenMalformed(t *testing.T) {
	_, err := VerifyAccessToken(testPubKey, "not-a-jwt")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestGenerateIDToken(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	ttl := 15 * time.Minute
	audience := "test-client"
	nonce := "test-nonce-abc"

	accessToken := "fake-access-token-for-at-hash"

	signed, err := GenerateIDToken(testPrivKey, testKID, testIssuer, audience, ttl, userID, orgID, "admin", "admin@test.com", true, "Admin", "User", nonce, accessToken)
	if err != nil {
		t.Fatalf("GenerateIDToken error: %v", err)
	}
	if signed == "" {
		t.Fatal("expected non-empty id token")
	}

	// Parse and verify the ID token
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	tok, err := parser.ParseWithClaims(signed, &IDTokenClaims{}, func(_ *jwt.Token) (any, error) {
		return testPubKey, nil
	})
	if err != nil {
		t.Fatalf("failed to parse id token: %v", err)
	}

	claims, ok := tok.Claims.(*IDTokenClaims)
	if !ok || !tok.Valid {
		t.Fatal("invalid token claims")
	}

	if claims.Subject != userID.String() {
		t.Errorf("sub = %q, want %q", claims.Subject, userID.String())
	}
	if claims.Issuer != testIssuer {
		t.Errorf("iss = %q, want %q", claims.Issuer, testIssuer)
	}
	aud, _ := claims.GetAudience()
	if len(aud) != 1 || aud[0] != audience {
		t.Errorf("aud = %v, want [%s]", aud, audience)
	}
	if claims.Nonce != nonce {
		t.Errorf("nonce = %q, want %q", claims.Nonce, nonce)
	}
	if claims.AtHash == "" {
		t.Error("expected non-empty at_hash")
	}
	if claims.PreferredUsername != "admin" {
		t.Errorf("preferred_username = %q, want admin", claims.PreferredUsername)
	}
	if claims.Email != "admin@test.com" {
		t.Errorf("email = %q, want admin@test.com", claims.Email)
	}
	if claims.OrgID != orgID {
		t.Errorf("org_id = %v, want %v", claims.OrgID, orgID)
	}
}

func TestGenerateIDTokenWithoutNonce(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	signed, err := GenerateIDToken(testPrivKey, testKID, testIssuer, "test-client", 15*time.Minute, userID, orgID, "admin", "admin@test.com", true, "", "", "", "some-access-token")
	if err != nil {
		t.Fatalf("GenerateIDToken error: %v", err)
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	tok, err := parser.ParseWithClaims(signed, &IDTokenClaims{}, func(_ *jwt.Token) (any, error) {
		return testPubKey, nil
	})
	if err != nil {
		t.Fatalf("failed to parse id token: %v", err)
	}

	claims := tok.Claims.(*IDTokenClaims)
	if claims.Nonce != "" {
		t.Errorf("nonce = %q, want empty (omitted)", claims.Nonce)
	}
}

func TestComputeAtHash(t *testing.T) {
	// Use a known access token and verify at_hash is deterministic and non-empty
	accessToken := "test-access-token-value"

	hash1 := computeAtHash(accessToken)
	if hash1 == "" {
		t.Fatal("expected non-empty at_hash")
	}

	// Same input should produce same output
	hash2 := computeAtHash(accessToken)
	if hash1 != hash2 {
		t.Errorf("at_hash not deterministic: %q != %q", hash1, hash2)
	}

	// Different input should produce different output
	hash3 := computeAtHash("different-token")
	if hash1 == hash3 {
		t.Errorf("different tokens should produce different at_hash values")
	}

	// Verify it does not contain padding characters
	for _, c := range hash1 {
		if c == '=' {
			t.Errorf("at_hash should not contain padding: %q", hash1)
			break
		}
	}
}

func TestGenerateMFAToken_HasAudienceClaim(t *testing.T) {
	userID := uuid.New()

	tokenStr, err := GenerateMFAToken(testPrivKey, testKID, testIssuer, userID)
	if err != nil {
		t.Fatalf("GenerateMFAToken: %v", err)
	}

	// Parse without validation to inspect raw claims
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	tok, _, err := parser.ParseUnverified(tokenStr, &MFAClaims{})
	if err != nil {
		t.Fatalf("parsing token: %v", err)
	}

	claims, ok := tok.Claims.(*MFAClaims)
	if !ok {
		t.Fatal("unexpected claims type")
	}

	aud, _ := claims.GetAudience()
	if len(aud) != 1 || aud[0] != MFAAudience {
		t.Errorf("audience = %v, want [%s]", aud, MFAAudience)
	}
}

func TestVerifyMFAToken_ValidToken(t *testing.T) {
	userID := uuid.New()

	tokenStr, err := GenerateMFAToken(testPrivKey, testKID, testIssuer, userID)
	if err != nil {
		t.Fatalf("GenerateMFAToken: %v", err)
	}

	got, err := VerifyMFAToken(testPubKey, tokenStr)
	if err != nil {
		t.Fatalf("VerifyMFAToken: %v", err)
	}
	if got != userID {
		t.Errorf("userID = %v, want %v", got, userID)
	}
}

func TestVerifyMFAToken_RejectsTokenWithoutAudience(t *testing.T) {
	userID := uuid.New()

	// Manually craft an MFA token without the audience claim
	now := time.Now()
	claims := MFAClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
		Purpose: "mfa_challenge",
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	tokenStr, err := tok.SignedString(testPrivKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	_, err = VerifyMFAToken(testPubKey, tokenStr)
	if err == nil {
		t.Fatal("expected error for token without audience, got nil")
	}
}

func TestVerifyMFAToken_RejectsWrongAudience(t *testing.T) {
	userID := uuid.New()

	// Craft a token with a wrong audience
	now := time.Now()
	claims := MFAClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{"wrong-audience"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
		Purpose: "mfa_challenge",
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	tokenStr, err := tok.SignedString(testPrivKey)
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	_, err = VerifyMFAToken(testPubKey, tokenStr)
	if err == nil {
		t.Fatal("expected error for token with wrong audience, got nil")
	}
}

func TestVerifyMFAToken_RejectsAccessTokenAsMFA(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	// Generate a regular access token and try to use it as an MFA token
	accessToken, err := GenerateAccessToken(testPrivKey, testKID, testIssuer, testIssuer, 15*time.Minute, userID, orgID, "testuser", "test@example.com", true, "Test", "User", "user")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	_, err = VerifyMFAToken(testPubKey, accessToken)
	if err == nil {
		t.Fatal("expected error when verifying access token as MFA token, got nil")
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

// ---------------------------------------------------------------------------
// Edge-case tests
// ---------------------------------------------------------------------------

func TestVerifyAccessToken_ExpiredTokenContainsExpiryError(t *testing.T) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   uuid.New().String(),
			Audience:  jwt.ClaimStrings{testIssuer},
			IssuedAt:  jwt.NewNumericDate(now.Add(-10 * time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-5 * time.Minute)),
		},
		OrgID:             uuid.New(),
		PreferredUsername: "user",
		Email:             "user@test.com",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(testPrivKey)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}

	_, err = VerifyAccessToken(testPubKey, signed)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected expiry-related error, got: %v", err)
	}
}

func TestVerifyAccessToken_WrongAudiencePassesSinceNotEnforced(t *testing.T) {
	// VerifyAccessToken does not enforce audience — this documents that behavior.
	signed, err := GenerateAccessToken(testPrivKey, testKID, testIssuer, "aud-A", 15*time.Minute,
		uuid.New(), uuid.New(), "u", "u@t.com", false, "", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := VerifyAccessToken(testPubKey, signed)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	aud, _ := claims.GetAudience()
	if len(aud) != 1 || aud[0] != "aud-A" {
		t.Errorf("aud = %v, want [aud-A]", aud)
	}
}

func TestVerifyAccessToken_WrongIssuerPassesSinceNotEnforced(t *testing.T) {
	// VerifyAccessToken does not enforce issuer — this documents that behavior.
	signed, err := GenerateAccessToken(testPrivKey, testKID, "https://evil.example.com", testIssuer,
		15*time.Minute, uuid.New(), uuid.New(), "u", "u@t.com", false, "", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	claims, err := VerifyAccessToken(testPubKey, signed)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Issuer != "https://evil.example.com" {
		t.Errorf("issuer = %q, want https://evil.example.com", claims.Issuer)
	}
}

func TestVerifyAccessToken_SignedWithDifferentKey(t *testing.T) {
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	signed, err := GenerateAccessToken(otherKey, testKID, testIssuer, testIssuer,
		15*time.Minute, uuid.New(), uuid.New(), "u", "u@t.com", false, "", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	_, err = VerifyAccessToken(testPubKey, signed)
	if err == nil {
		t.Fatal("expected error when verifying with different key")
	}
}

func TestVerifyAccessToken_MalformedStrings(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"single dot", "a.b"},
		{"three dots no content", "..."},
		{"random garbage", "not-a-jwt-at-all"},
		{"valid base64 but bad sig", base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`)) + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"x"}`)) + ".invalidsig"},
		{"null bytes", "\x00\x00\x00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyAccessToken(testPubKey, tt.token)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tt.name)
			}
		})
	}
}

func TestVerifyAccessToken_NoExpiry(t *testing.T) {
	// Token without ExpiresAt — jwt/v5 does not require exp by default.
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   testIssuer,
			Subject:  uuid.New().String(),
			Audience: jwt.ClaimStrings{testIssuer},
			IssuedAt: jwt.NewNumericDate(now),
		},
		OrgID:             uuid.New(),
		PreferredUsername: "user",
		Email:             "user@test.com",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(testPrivKey)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}

	result, err := VerifyAccessToken(testPubKey, signed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExpiresAt != nil {
		t.Errorf("expected nil ExpiresAt, got %v", result.ExpiresAt)
	}
}

func TestVerifyMFAToken_ExpiredToken(t *testing.T) {
	now := time.Now()
	claims := MFAClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   uuid.New().String(),
			Audience:  jwt.ClaimStrings{MFAAudience},
			IssuedAt:  jwt.NewNumericDate(now.Add(-10 * time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Minute)),
		},
		Purpose: "mfa_challenge",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(testPrivKey)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}

	_, err = VerifyMFAToken(testPubKey, signed)
	if err == nil {
		t.Fatal("expected error for expired MFA token")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected expiry error, got: %v", err)
	}
}

func TestVerifyMFAToken_WrongPurpose(t *testing.T) {
	now := time.Now()
	claims := MFAClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   uuid.New().String(),
			Audience:  jwt.ClaimStrings{MFAAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
		Purpose: "password_reset",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(testPrivKey)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}

	_, err = VerifyMFAToken(testPubKey, signed)
	if err == nil {
		t.Fatal("expected error for wrong purpose MFA token")
	}
	if !strings.Contains(err.Error(), "invalid MFA token") {
		t.Errorf("expected 'invalid MFA token' error, got: %v", err)
	}
}

func TestVerifyMFAToken_EmptyPurpose(t *testing.T) {
	now := time.Now()
	claims := MFAClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   uuid.New().String(),
			Audience:  jwt.ClaimStrings{MFAAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
		Purpose: "",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(testPrivKey)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}

	_, err = VerifyMFAToken(testPubKey, signed)
	if err == nil {
		t.Fatal("expected error for empty purpose MFA token")
	}
}

func TestVerifyMFAToken_SignedWithDifferentKey(t *testing.T) {
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	tokenStr, err := GenerateMFAToken(otherKey, testKID, testIssuer, uuid.New())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	_, err = VerifyMFAToken(testPubKey, tokenStr)
	if err == nil {
		t.Fatal("expected error verifying MFA token with wrong key")
	}
}

func TestVerifyMFAToken_MalformedStrings(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"garbage", "xyz123"},
		{"two dots", "a.b.c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyMFAToken(testPubKey, tt.token)
			if err == nil {
				t.Fatal("expected error for malformed MFA token")
			}
		})
	}
}

func TestRefreshTokenUniqueness_1000(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := range 1000 {
		tok, err := GenerateRefreshToken()
		if err != nil {
			t.Fatalf("GenerateRefreshToken #%d: %v", i, err)
		}
		if _, dup := seen[tok]; dup {
			t.Fatalf("duplicate refresh token at iteration %d: %s", i, tok)
		}
		seen[tok] = struct{}{}
	}
}

func TestComputeAtHash_Correctness(t *testing.T) {
	// Manually compute expected at_hash per OIDC Core 1.0 section 3.1.3.6:
	// SHA-256 of access token, take left 128 bits (16 bytes), base64url without padding.
	accessToken := "ya29.test-access-token-12345"
	h := sha256.Sum256([]byte(accessToken))
	leftHalf := h[:sha256.Size/2]
	expected := strings.TrimRight(base64.URLEncoding.EncodeToString(leftHalf), "=")

	got := computeAtHash(accessToken)
	if got != expected {
		t.Errorf("at_hash = %q, want %q", got, expected)
	}

	// 16 bytes base64url-encoded without padding = 22 characters
	if len(got) != 22 {
		t.Errorf("at_hash length = %d, want 22", len(got))
	}
}

func TestGenerateIDToken_EmptyNonce(t *testing.T) {
	signed, err := GenerateIDToken(testPrivKey, testKID, testIssuer, "client-1",
		15*time.Minute, uuid.New(), uuid.New(), "user", "u@t.com", true, "", "", "", "some-at")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}))
	tok, err := parser.ParseWithClaims(signed, &IDTokenClaims{}, func(_ *jwt.Token) (any, error) {
		return testPubKey, nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	claims := tok.Claims.(*IDTokenClaims)
	if claims.Nonce != "" {
		t.Errorf("nonce should be empty, got %q", claims.Nonce)
	}
}

func TestGenerateIDToken_VeryLongNonce(t *testing.T) {
	longNonce := strings.Repeat("a", 8192)
	signed, err := GenerateIDToken(testPrivKey, testKID, testIssuer, "client-1",
		15*time.Minute, uuid.New(), uuid.New(), "user", "u@t.com", true, "", "", longNonce, "some-at")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}))
	tok, err := parser.ParseWithClaims(signed, &IDTokenClaims{}, func(_ *jwt.Token) (any, error) {
		return testPubKey, nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	claims := tok.Claims.(*IDTokenClaims)
	if claims.Nonce != longNonce {
		t.Errorf("nonce length = %d, want %d", len(claims.Nonce), len(longNonce))
	}
}

func TestGenerateIDToken_NoAccessToken_NoAtHash(t *testing.T) {
	signed, err := GenerateIDToken(testPrivKey, testKID, testIssuer, "client-1",
		15*time.Minute, uuid.New(), uuid.New(), "user", "u@t.com", true, "", "", "nonce", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}))
	tok, err := parser.ParseWithClaims(signed, &IDTokenClaims{}, func(_ *jwt.Token) (any, error) {
		return testPubKey, nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	claims := tok.Claims.(*IDTokenClaims)
	if claims.AtHash != "" {
		t.Errorf("at_hash should be empty when no access token provided, got %q", claims.AtHash)
	}
}

func TestGenerateAccessToken_AllOptionalFieldsEmpty(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	signed, err := GenerateAccessToken(testPrivKey, testKID, testIssuer, testIssuer,
		15*time.Minute, userID, orgID, "", "", false, "", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := VerifyAccessToken(testPubKey, signed)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}

	if claims.PreferredUsername != "" {
		t.Errorf("preferred_username = %q, want empty", claims.PreferredUsername)
	}
	if claims.Email != "" {
		t.Errorf("email = %q, want empty", claims.Email)
	}
	if claims.EmailVerified {
		t.Error("email_verified = true, want false")
	}
	if claims.GivenName != "" {
		t.Errorf("given_name = %q, want empty", claims.GivenName)
	}
	if claims.FamilyName != "" {
		t.Errorf("family_name = %q, want empty", claims.FamilyName)
	}
	if len(claims.Roles) != 0 {
		t.Errorf("roles = %v, want empty", claims.Roles)
	}
	if claims.Custom != nil {
		t.Errorf("custom = %v, want nil", claims.Custom)
	}
}

func TestGenerateAccessTokenWithCustomClaims_NestedObjects(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	customClaims := map[string]any{
		"tenant": map[string]any{
			"id":   "t-123",
			"tier": "enterprise",
			"limits": map[string]any{
				"api_calls": 10000,
				"storage":   "100GB",
			},
		},
		"feature_flags": []string{"beta", "dark-mode"},
		"score":         42.5,
		"active":        true,
	}

	signed, err := GenerateAccessTokenWithCustomClaims(testPrivKey, testKID, testIssuer, testIssuer,
		15*time.Minute, userID, orgID, "user", "u@t.com", true, "U", "T", customClaims, "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := VerifyAccessToken(testPubKey, signed)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}

	if claims.Custom == nil {
		t.Fatal("custom claims should not be nil")
	}

	// Verify nested object survived round-trip
	tenant, ok := claims.Custom["tenant"].(map[string]any)
	if !ok {
		t.Fatalf("tenant is %T, want map[string]any", claims.Custom["tenant"])
	}
	if tenant["id"] != "t-123" {
		t.Errorf("tenant.id = %v, want t-123", tenant["id"])
	}

	limits, ok := tenant["limits"].(map[string]any)
	if !ok {
		t.Fatalf("tenant.limits is %T, want map[string]any", tenant["limits"])
	}
	// JSON numbers unmarshal as float64
	if limits["api_calls"] != float64(10000) {
		t.Errorf("tenant.limits.api_calls = %v, want 10000", limits["api_calls"])
	}

	if claims.Custom["active"] != true {
		t.Errorf("custom.active = %v, want true", claims.Custom["active"])
	}
	if claims.Custom["score"] != 42.5 {
		t.Errorf("custom.score = %v, want 42.5", claims.Custom["score"])
	}
}

func TestGenerateAccessToken_WithMultipleRoles(t *testing.T) {
	roles := []string{"admin", "editor", "viewer"}
	signed, err := GenerateAccessToken(testPrivKey, testKID, testIssuer, testIssuer,
		15*time.Minute, uuid.New(), uuid.New(), "user", "u@t.com", true, "", "", roles...)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := VerifyAccessToken(testPubKey, signed)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(claims.Roles) != 3 {
		t.Fatalf("roles length = %d, want 3", len(claims.Roles))
	}
	for i, r := range roles {
		if claims.Roles[i] != r {
			t.Errorf("roles[%d] = %q, want %q", i, claims.Roles[i], r)
		}
	}
}
