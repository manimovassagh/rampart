package handler

import (
	"net/http"
)

// OIDCPage handles GET /admin/oidc
func (h *AdminConsoleHandler) OIDCPage(w http.ResponseWriter, r *http.Request) {
	oidc := &DiscoveryResponse{
		Issuer:                           h.issuer,
		AuthorizationEndpoint:            h.issuer + "/oauth/authorize",
		TokenEndpoint:                    h.issuer + "/oauth/token",
		UserinfoEndpoint:                 h.issuer + "/me",
		JWKSURI:                          h.issuer + "/.well-known/jwks.json",
		ResponseTypesSupported:           []string{"code"},
		GrantTypesSupported:              []string{"authorization_code", "refresh_token"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
		ScopesSupported:                  []string{"openid", "profile", "email"},
		ClaimsSupported: []string{
			"sub", "iss", "iat", "exp",
			"preferred_username", "email", "email_verified",
			"given_name", "family_name", "org_id", "roles",
		},
		CodeChallengeMethodsSupported: []string{"S256"},
	}

	h.render(w, r, "oidc", &pageData{
		Title:     "OIDC Configuration",
		ActiveNav: "oidc",
		OIDC:      oidc,
	})
}

// SocialProvidersPage handles GET /admin/social
func (h *AdminConsoleHandler) SocialProvidersPage(w http.ResponseWriter, r *http.Request) {
	type providerDef struct {
		name    string
		envHint string
	}
	allProviders := []providerDef{
		{"google", "RAMPART_GOOGLE_CLIENT_ID, RAMPART_GOOGLE_CLIENT_SECRET"},
		{"github", "RAMPART_GITHUB_CLIENT_ID, RAMPART_GITHUB_CLIENT_SECRET"},
		{"apple", "RAMPART_APPLE_CLIENT_ID, RAMPART_APPLE_TEAM_ID, RAMPART_APPLE_KEY_ID"},
	}

	providers := make([]SocialProviderInfo, 0, len(allProviders))
	for _, p := range allProviders {
		_, enabled := h.socialRegistry.Get(p.name)
		providers = append(providers, SocialProviderInfo{
			Name:    p.name,
			Enabled: enabled,
			EnvHint: p.envHint,
		})
	}

	h.render(w, r, "social_providers", &pageData{
		Title:           "Social Providers",
		ActiveNav:       navSocial,
		SocialProviders: providers,
	})
}
