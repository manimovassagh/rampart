package social

import (
	"context"
	"net/http"
	"time"
)

// defaultHTTPClient is a shared HTTP client with a 10-second timeout,
// used by social providers when no custom HTTPClient is provided.
// This prevents hanging indefinitely on slow upstream servers.
var defaultHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

// UserInfo represents the normalized user info returned by any social provider.
type UserInfo struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
	Name           string
	GivenName      string
	FamilyName     string
	AvatarURL      string
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt *time.Time
}

// Provider defines the interface for social login providers.
type Provider interface {
	// Name returns the provider identifier (e.g., "google", "github", "apple").
	Name() string
	// AuthURL returns the URL to redirect the user to for authentication.
	// state is the CSRF token, redirectURL is the callback URL.
	AuthURL(state, redirectURL string) string
	// Exchange exchanges the authorization code for user information.
	Exchange(ctx context.Context, code, redirectURL string) (*UserInfo, error)
}
