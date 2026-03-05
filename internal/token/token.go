package token

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
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

// GenerateRefreshToken creates a cryptographically random hex string.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
