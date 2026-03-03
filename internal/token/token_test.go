package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
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

	signed, err := GenerateAccessToken(testPrivKey, testKID, testIssuer, ttl, userID, orgID, "admin", "admin@test.com", true, "Admin", "User")
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
	signed, err := GenerateAccessToken(testPrivKey, testKID, testIssuer, 15*time.Minute, uuid.New(), uuid.New(), "admin", "admin@test.com", false, "", "")
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
	signed, err := GenerateAccessToken(testPrivKey, testKID, testIssuer, 15*time.Minute, uuid.New(), uuid.New(), "admin", "admin@test.com", false, "", "")
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
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
		OrgID:            uuid.New(),
		PreferredUsername: "admin",
		Email:            "admin@test.com",
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
