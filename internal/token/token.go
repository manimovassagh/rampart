package token

import (
	"crypto/rand"
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
	PreferredUsername  string    `json:"preferred_username"`
	Email             string    `json:"email"`
	EmailVerified     bool      `json:"email_verified"`
	GivenName         string    `json:"given_name,omitempty"`
	FamilyName        string    `json:"family_name,omitempty"`
}

// GenerateAccessToken creates a signed HS256 JWT with user claims.
func GenerateAccessToken(secret string, ttl time.Duration, userID, orgID uuid.UUID, username, email string, emailVerified bool, givenName, familyName string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		OrgID:            orgID,
		PreferredUsername: username,
		Email:            email,
		EmailVerified:    emailVerified,
		GivenName:        givenName,
		FamilyName:       familyName,
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("signing access token: %w", err)
	}
	return signed, nil
}

// VerifyAccessToken parses and validates a signed JWT, returning the claims.
func VerifyAccessToken(secret, tokenString string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(_ *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
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
