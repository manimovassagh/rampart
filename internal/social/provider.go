package social

import (
	"context"
	"time"
)

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
