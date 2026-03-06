package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	googleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
	googleScopes      = "openid email profile"
)

// GoogleProvider implements the Provider interface for Google OAuth 2.0.
type GoogleProvider struct {
	ClientID     string
	ClientSecret string
	// HTTPClient is optional; if nil, http.DefaultClient is used.
	HTTPClient *http.Client
}

// compile-time check that GoogleProvider implements Provider.
var _ Provider = (*GoogleProvider)(nil)

// Name returns "google".
func (g *GoogleProvider) Name() string {
	return "google"
}

// AuthURL returns the Google OAuth 2.0 authorization URL.
func (g *GoogleProvider) AuthURL(state, redirectURL string) string {
	params := url.Values{
		"client_id":     {g.ClientID},
		"redirect_uri":  {redirectURL},
		"response_type": {"code"},
		"scope":         {googleScopes},
		"state":         {state},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
	}
	return googleAuthURL + "?" + params.Encode()
}

// Exchange exchanges the authorization code for user information from Google.
func (g *GoogleProvider) Exchange(ctx context.Context, code, redirectURL string) (*UserInfo, error) {
	client := g.httpClient()

	tokenResp, err := g.exchangeToken(ctx, client, code, redirectURL)
	if err != nil {
		return nil, fmt.Errorf("google token exchange: %w", err)
	}

	userInfo, err := g.fetchUserInfo(ctx, client, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("google fetch user info: %w", err)
	}

	info := &UserInfo{
		ProviderUserID: userInfo.Sub,
		Email:          userInfo.Email,
		EmailVerified:  userInfo.EmailVerified,
		Name:           userInfo.Name,
		GivenName:      userInfo.GivenName,
		FamilyName:     userInfo.FamilyName,
		AvatarURL:      userInfo.Picture,
		AccessToken:    tokenResp.AccessToken,
		RefreshToken:   tokenResp.RefreshToken,
	}
	if tokenResp.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		info.TokenExpiresAt = &exp
	}
	return info, nil
}

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type googleUserInfoResponse struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

func (g *GoogleProvider) httpClient() *http.Client {
	if g.HTTPClient != nil {
		return g.HTTPClient
	}
	return http.DefaultClient
}

func (g *GoogleProvider) exchangeToken(ctx context.Context, client *http.Client, code, redirectURL string) (*googleTokenResponse, error) {
	data := url.Values{
		"code":          {code},
		"client_id":     {g.ClientID},
		"client_secret": {g.ClientSecret},
		"redirect_uri":  {redirectURL},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(data.Encode()))
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

	var tokenResp googleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}
	return &tokenResp, nil
}

func (g *GoogleProvider) fetchUserInfo(ctx context.Context, client *http.Client, accessToken string) (*googleUserInfoResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing userinfo request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo googleUserInfoResponse
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("parsing userinfo response: %w", err)
	}
	return &userInfo, nil
}
