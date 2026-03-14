package social

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	appleAuthURL  = "https://appleid.apple.com/auth/authorize"
	appleTokenURL = "https://appleid.apple.com/auth/token"
	appleJWKSURL  = "https://appleid.apple.com/auth/keys"
	appleIssuer   = "https://appleid.apple.com"
	appleScopes   = "name email"

	// jwksCacheTTL controls how long Apple's JWKS is cached before re-fetching.
	jwksCacheTTL = 1 * time.Hour
)

// AppleProvider implements the Provider interface for Apple Sign In.
type AppleProvider struct {
	ClientID   string
	TeamID     string
	KeyID      string
	PrivateKey string
	// HTTPClient is optional; if nil, http.DefaultClient is used.
	HTTPClient *http.Client
	// ClientSecretFunc generates the client_secret JWT for Apple.
	// If nil, the PrivateKey, TeamID, and KeyID fields are required but
	// the actual JWT generation must be provided by the caller.
	ClientSecretFunc func() (string, error)

	// JWKSURL overrides the Apple JWKS endpoint (for testing).
	// If empty, the default appleJWKSURL is used.
	JWKSURL string

	jwksCache   *appleJWKS
	jwksCacheMu sync.RWMutex
}

// compile-time check that AppleProvider implements Provider.
var _ Provider = (*AppleProvider)(nil)

// Name returns "apple".
func (a *AppleProvider) Name() string {
	return "apple"
}

// AuthURL returns the Apple Sign In authorization URL.
func (a *AppleProvider) AuthURL(state, redirectURL string) string {
	params := url.Values{
		"client_id":     {a.ClientID},
		"redirect_uri":  {redirectURL},
		"response_type": {"code"},
		"scope":         {appleScopes},
		"state":         {state},
		"response_mode": {"form_post"},
	}
	return appleAuthURL + "?" + params.Encode()
}

// Exchange exchanges the authorization code for user information from Apple.
func (a *AppleProvider) Exchange(ctx context.Context, code, redirectURL string) (*UserInfo, error) {
	client := a.httpClient()

	clientSecret, err := a.getClientSecret()
	if err != nil {
		return nil, fmt.Errorf("apple generate client secret: %w", err)
	}

	tokenResp, err := a.exchangeToken(ctx, client, code, redirectURL, clientSecret)
	if err != nil {
		return nil, fmt.Errorf("apple token exchange: %w", err)
	}

	claims, err := a.verifyIDToken(ctx, client, tokenResp.IDToken)
	if err != nil {
		return nil, fmt.Errorf("apple verify id token: %w", err)
	}

	info := &UserInfo{
		ProviderUserID: claims.Sub,
		Email:          claims.Email,
		EmailVerified:  claims.EmailVerified,
		AccessToken:    tokenResp.AccessToken,
		RefreshToken:   tokenResp.RefreshToken,
	}
	if tokenResp.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		info.TokenExpiresAt = &exp
	}
	return info, nil
}

type appleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// appleIDTokenClaims holds the relevant claims from Apple's ID token.
type appleIDTokenClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// UnmarshalJSON implements custom unmarshalling for appleIDTokenClaims
// to handle email_verified being either a boolean or a string.
func (c *appleIDTokenClaims) UnmarshalJSON(data []byte) error {
	// Use an alias to avoid infinite recursion.
	type alias struct {
		Sub           string      `json:"sub"`
		Email         string      `json:"email"`
		EmailVerified interface{} `json:"email_verified"`
	}
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Sub = raw.Sub
	c.Email = raw.Email

	switch v := raw.EmailVerified.(type) {
	case bool:
		c.EmailVerified = v
	case string:
		c.EmailVerified = v == "true"
	default:
		c.EmailVerified = false
	}
	return nil
}

// appleJWKS represents a cached set of Apple's JSON Web Keys.
type appleJWKS struct {
	Keys      []appleJWK
	FetchedAt time.Time
}

// appleJWK represents a single JSON Web Key from Apple's JWKS endpoint.
type appleJWK struct {
	KTY string `json:"kty"`
	KID string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// appleJWKSResponse is the response from Apple's /auth/keys endpoint.
type appleJWKSResponse struct {
	Keys []appleJWK `json:"keys"`
}

func (a *AppleProvider) httpClient() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return defaultHTTPClient
}

func (a *AppleProvider) getClientSecret() (string, error) {
	if a.ClientSecretFunc != nil {
		return a.ClientSecretFunc()
	}
	return "", fmt.Errorf("ClientSecretFunc is not configured; provide a function that generates the Apple client_secret JWT")
}

func (a *AppleProvider) exchangeToken(ctx context.Context, client *http.Client, code, redirectURL, clientSecret string) (*appleTokenResponse, error) {
	data := url.Values{
		"client_id":     {a.ClientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURL},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, appleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req) //nolint:bodyclose,gosec // closed on next line; URL is a hardcoded constant
	if err != nil {
		return nil, fmt.Errorf("executing token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp appleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}
	return &tokenResp, nil
}

// jwksEndpoint returns the JWKS URL, using the override if set (for testing).
func (a *AppleProvider) jwksEndpoint() string {
	if a.JWKSURL != "" {
		return a.JWKSURL
	}
	return appleJWKSURL
}

// verifyIDToken verifies the Apple ID token JWT signature against Apple's
// public keys (JWKS) and validates the standard claims (iss, aud, exp).
func (a *AppleProvider) verifyIDToken(ctx context.Context, client *http.Client, idToken string) (*appleIDTokenClaims, error) {
	keys, err := a.fetchJWKS(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("fetching Apple JWKS: %w", err)
	}

	// Parse and verify the JWT using the JWKS key function.
	token, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is RS256.
		if token.Method.Alg() != "RS256" {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("missing kid in token header")
		}

		// Find the matching key.
		for i := range keys {
			if keys[i].KID == kid {
				pubKey, keyErr := jwkToRSAPublicKey(&keys[i])
				if keyErr != nil {
					return nil, fmt.Errorf("converting JWK to RSA public key: %w", keyErr)
				}
				return pubKey, nil
			}
		}
		return nil, fmt.Errorf("no matching key found for kid %q", kid)
	},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(appleIssuer),
		jwt.WithAudience(a.ClientID),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("verifying JWT: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid JWT token")
	}

	// Extract claims from the verified token.
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("unexpected claims type")
	}

	claimsJSON, err := json.Marshal(mapClaims)
	if err != nil {
		return nil, fmt.Errorf("marshaling claims: %w", err)
	}

	var claims appleIDTokenClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("parsing claims: %w", err)
	}

	return &claims, nil
}

// fetchJWKS fetches Apple's JWKS (JSON Web Key Set) with caching.
func (a *AppleProvider) fetchJWKS(ctx context.Context, client *http.Client) ([]appleJWK, error) {
	// Check cache first (read lock).
	a.jwksCacheMu.RLock()
	if a.jwksCache != nil && time.Since(a.jwksCache.FetchedAt) < jwksCacheTTL {
		keys := a.jwksCache.Keys
		a.jwksCacheMu.RUnlock()
		return keys, nil
	}
	a.jwksCacheMu.RUnlock()

	// Cache miss or expired — fetch fresh keys (write lock).
	a.jwksCacheMu.Lock()
	defer a.jwksCacheMu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have refreshed).
	if a.jwksCache != nil && time.Since(a.jwksCache.FetchedAt) < jwksCacheTTL {
		return a.jwksCache.Keys, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.jwksEndpoint(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating JWKS request: %w", err)
	}

	resp, err := client.Do(req) //nolint:bodyclose,gosec // closed on next line; URL is from config
	if err != nil {
		return nil, fmt.Errorf("executing JWKS request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading JWKS response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var jwksResp appleJWKSResponse
	if err := json.Unmarshal(body, &jwksResp); err != nil {
		return nil, fmt.Errorf("parsing JWKS response: %w", err)
	}

	if len(jwksResp.Keys) == 0 {
		return nil, fmt.Errorf("JWKS response contains no keys")
	}

	a.jwksCache = &appleJWKS{
		Keys:      jwksResp.Keys,
		FetchedAt: time.Now(),
	}
	return jwksResp.Keys, nil
}

// jwkToRSAPublicKey converts an Apple JWK to an *rsa.PublicKey.
func jwkToRSAPublicKey(key *appleJWK) (*rsa.PublicKey, error) {
	if key.KTY != "RSA" {
		return nil, fmt.Errorf("unsupported key type: %s", key.KTY)
	}

	nBytes, err := base64URLDecode(key.N)
	if err != nil {
		return nil, fmt.Errorf("decoding modulus: %w", err)
	}

	eBytes, err := base64URLDecode(key.E)
	if err != nil {
		return nil, fmt.Errorf("decoding exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// base64URLDecode decodes a base64url-encoded string with optional padding.
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed.
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
