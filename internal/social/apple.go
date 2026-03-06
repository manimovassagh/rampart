package social

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	appleAuthURL  = "https://appleid.apple.com/auth/authorize"
	appleTokenURL = "https://appleid.apple.com/auth/token"
	appleScopes   = "name email"
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

	claims, err := decodeIDToken(tokenResp.IDToken)
	if err != nil {
		return nil, fmt.Errorf("apple decode id token: %w", err)
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

func (a *AppleProvider) httpClient() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
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

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing token request: %w", err)
	}
	defer resp.Body.Close()

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

// decodeIDToken decodes an Apple ID token JWT without verifying the signature.
// It extracts the payload (second segment) and unmarshals the claims.
// Note: In production, the signature should be verified against Apple's public keys.
func decodeIDToken(idToken string) (*appleIDTokenClaims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT: expected 3 parts, got %d", len(parts))
	}

	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding JWT payload: %w", err)
	}

	var claims appleIDTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parsing JWT claims: %w", err)
	}
	return &claims, nil
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
