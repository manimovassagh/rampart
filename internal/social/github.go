package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	githubAuthURL  = "https://github.com/login/oauth/authorize"
	githubTokenURL = "https://github.com/login/oauth/access_token"
	githubUserURL  = "https://api.github.com/user"
	githubEmailURL = "https://api.github.com/user/emails"
	githubScopes   = "user:email read:user"
)

// GitHubProvider implements the Provider interface for GitHub OAuth.
type GitHubProvider struct {
	ClientID     string
	ClientSecret string
	// HTTPClient is optional; if nil, http.DefaultClient is used.
	HTTPClient *http.Client
}

// compile-time check that GitHubProvider implements Provider.
var _ Provider = (*GitHubProvider)(nil)

// Name returns "github".
func (g *GitHubProvider) Name() string {
	return "github"
}

// AuthURL returns the GitHub OAuth authorization URL.
func (g *GitHubProvider) AuthURL(state, redirectURL string) string {
	params := url.Values{
		"client_id":    {g.ClientID},
		"redirect_uri": {redirectURL},
		"scope":        {githubScopes},
		"state":        {state},
	}
	return githubAuthURL + "?" + params.Encode()
}

// Exchange exchanges the authorization code for user information from GitHub.
func (g *GitHubProvider) Exchange(ctx context.Context, code, redirectURL string) (*UserInfo, error) {
	client := g.httpClient()

	tokenResp, err := g.exchangeToken(ctx, client, code, redirectURL)
	if err != nil {
		return nil, fmt.Errorf("github token exchange: %w", err)
	}

	user, err := g.fetchUser(ctx, client, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("github fetch user: %w", err)
	}

	email, emailVerified, err := g.fetchPrimaryEmail(ctx, client, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("github fetch email: %w", err)
	}

	info := &UserInfo{
		ProviderUserID: strconv.FormatInt(user.ID, 10),
		Email:          email,
		EmailVerified:  emailVerified,
		Name:           user.Name,
		AvatarURL:      user.AvatarURL,
		AccessToken:    tokenResp.AccessToken,
		RefreshToken:   tokenResp.RefreshToken,
	}
	if tokenResp.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		info.TokenExpiresAt = &exp
	}
	return info, nil
}

type githubTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

type githubUserResponse struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmailResponse struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (g *GitHubProvider) httpClient() *http.Client {
	if g.HTTPClient != nil {
		return g.HTTPClient
	}
	return defaultHTTPClient
}

func (g *GitHubProvider) exchangeToken(ctx context.Context, client *http.Client, code, redirectURL string) (*githubTokenResponse, error) {
	data := url.Values{
		"client_id":     {g.ClientID},
		"client_secret": {g.ClientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURL},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req) //nolint:bodyclose,gosec // closed on next line; URL is a hardcoded constant
	if err != nil {
		return nil, fmt.Errorf("executing token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp githubTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("empty access token in response: %s", string(body))
	}

	return &tokenResp, nil
}

func (g *GitHubProvider) fetchUser(ctx context.Context, client *http.Client, accessToken string) (*githubUserResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubUserURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req) //nolint:bodyclose,gosec // closed on next line; URL is a hardcoded constant
	if err != nil {
		return nil, fmt.Errorf("executing user request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading user response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var user githubUserResponse
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parsing user response: %w", err)
	}
	return &user, nil
}

func (g *GitHubProvider) fetchPrimaryEmail(ctx context.Context, client *http.Client, accessToken string) (email string, verified bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubEmailURL, http.NoBody)
	if err != nil {
		return "", false, fmt.Errorf("creating email request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req) //nolint:bodyclose,gosec // closed on next line; URL is a hardcoded constant
	if err != nil {
		return "", false, fmt.Errorf("executing email request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", false, fmt.Errorf("reading email response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("email endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var emails []githubEmailResponse
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", false, fmt.Errorf("parsing email response: %w", err)
	}

	for _, e := range emails {
		if e.Primary {
			return e.Email, e.Verified, nil
		}
	}

	if len(emails) > 0 {
		return emails[0].Email, emails[0].Verified, nil
	}

	return "", false, fmt.Errorf("no email addresses found")
}
