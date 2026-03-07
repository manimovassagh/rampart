package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims are the JWT claims included in access tokens.
// Fields are OIDC-aligned for future compatibility.
type Claims struct {
	jwt.RegisteredClaims
	OrgID             uuid.UUID `json:"org_id"`
	PreferredUsername string    `json:"preferred_username"`
	Email             string    `json:"email"`
	EmailVerified     bool      `json:"email_verified"`
	GivenName         string    `json:"given_name,omitempty"`
	FamilyName        string    `json:"family_name,omitempty"`
	Roles             []string  `json:"roles,omitempty"`
}

// GenerateAccessToken creates a signed RS256 JWT with user claims.
func GenerateAccessToken(key *rsa.PrivateKey, kid, issuer string, ttl time.Duration, userID, orgID uuid.UUID, username, email string, emailVerified bool, givenName, familyName string, roles ...string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		OrgID:             orgID,
		PreferredUsername: username,
		Email:             email,
		EmailVerified:     emailVerified,
		GivenName:         givenName,
		FamilyName:        familyName,
		Roles:             roles,
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid

	signed, err := tok.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("signing access token: %w", err)
	}
	return signed, nil
}

// VerifyAccessToken parses and validates a signed RS256 JWT, returning the claims.
func VerifyAccessToken(pubKey *rsa.PublicKey, tokenString string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(_ *jwt.Token) (any, error) {
		return pubKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("parsing access token: %w", err)
	}

	claims, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// IDTokenClaims are the JWT claims for OIDC ID tokens.
type IDTokenClaims struct {
	jwt.RegisteredClaims
	Nonce             string    `json:"nonce,omitempty"`
	AtHash            string    `json:"at_hash,omitempty"`
	OrgID             uuid.UUID `json:"org_id"`
	PreferredUsername string    `json:"preferred_username"`
	Email             string    `json:"email"`
	EmailVerified     bool      `json:"email_verified"`
	GivenName         string    `json:"given_name,omitempty"`
	FamilyName        string    `json:"family_name,omitempty"`
}

// GenerateIDToken creates a signed RS256 ID token per OpenID Connect Core 1.0.
func GenerateIDToken(key *rsa.PrivateKey, kid, issuer, audience string, ttl time.Duration, userID, orgID uuid.UUID, username, email string, emailVerified bool, givenName, familyName, nonce, accessToken string) (string, error) {
	now := time.Now()
	claims := IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		OrgID:             orgID,
		PreferredUsername: username,
		Email:             email,
		EmailVerified:     emailVerified,
		GivenName:         givenName,
		FamilyName:        familyName,
	}

	if nonce != "" {
		claims.Nonce = nonce
	}

	if accessToken != "" {
		claims.AtHash = computeAtHash(accessToken)
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid

	signed, err := tok.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("signing id token: %w", err)
	}
	return signed, nil
}

// computeAtHash computes the at_hash claim per OIDC Core 1.0 §3.1.3.6.
// For RS256: SHA-256 of the access token, take left half, base64url-encode.
func computeAtHash(accessToken string) string {
	h := sha256.Sum256([]byte(accessToken))
	leftHalf := h[:sha256.Size/2]
	return strings.TrimRight(base64.URLEncoding.EncodeToString(leftHalf), "=")
}

// MFAClaims are the JWT claims for short-lived MFA challenge tokens.
type MFAClaims struct {
	jwt.RegisteredClaims
	Purpose string `json:"purpose"`
}

// GenerateMFAToken creates a short-lived JWT (5 minutes) proving the user passed
// password authentication and needs to complete MFA verification.
func GenerateMFAToken(key *rsa.PrivateKey, kid, issuer string, userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := MFAClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
		Purpose: "mfa_challenge",
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid

	signed, err := tok.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("signing MFA token: %w", err)
	}
	return signed, nil
}

// VerifyMFAToken parses and validates an MFA challenge token, returning the user ID.
func VerifyMFAToken(pubKey *rsa.PublicKey, tokenString string) (uuid.UUID, error) {
	tok, err := jwt.ParseWithClaims(tokenString, &MFAClaims{}, func(_ *jwt.Token) (any, error) {
		return pubKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	if err != nil {
		return uuid.Nil, fmt.Errorf("parsing MFA token: %w", err)
	}

	claims, ok := tok.Claims.(*MFAClaims)
	if !ok || !tok.Valid || claims.Purpose != "mfa_challenge" {
		return uuid.Nil, fmt.Errorf("invalid MFA token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid MFA token subject: %w", err)
	}
	return userID, nil
}

// GenerateRefreshToken creates a cryptographically random hex string.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
