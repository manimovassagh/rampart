// admin_console_oidc.go contains admin console handlers for OIDC configuration
// and social provider management: OIDCPage, SocialProvidersPage,
// UpdateSocialProviderAction, refreshSocialProvider.
package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/social"
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

// socialProviderDef defines the known providers and their extra fields.
type socialProviderDef struct {
	name        string
	label       string
	extraFields []SocialProviderField
}

var knownProviders = []socialProviderDef{
	{"google", "Google OAuth 2.0", nil},
	{"github", "GitHub OAuth", nil},
	{"apple", "Apple Sign In", []SocialProviderField{
		{Key: "team_id", Label: "Team ID"},
		{Key: "key_id", Label: "Key ID"},
	}},
}

// SocialProvidersPage handles GET /admin/social
func (h *AdminConsoleHandler) SocialProvidersPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	dbConfigs, err := h.store.ListSocialProviderConfigs(ctx, orgID)
	if err != nil {
		h.logger.Error("failed to list social provider configs", "error", err)
	}
	configMap := make(map[string]*model.SocialProviderConfig, len(dbConfigs))
	for _, c := range dbConfigs {
		configMap[c.Provider] = c
	}

	providers := make([]SocialProviderInfo, 0, len(knownProviders))
	for _, def := range knownProviders {
		info := SocialProviderInfo{
			Name:        def.name,
			Label:       def.label,
			ExtraFields: def.extraFields,
		}

		if cfg, ok := configMap[def.name]; ok {
			info.Enabled = cfg.Enabled
			info.ClientID = cfg.ClientID
			info.HasSecret = cfg.ClientSecret != ""
			info.ExtraConfig = cfg.ExtraConfig
			for i, f := range info.ExtraFields {
				if v, exists := cfg.ExtraConfig[f.Key]; exists {
					info.ExtraFields[i].Value = v
				}
			}
		} else {
			_, info.Enabled = h.socialRegistry.Get(def.name)
		}

		providers = append(providers, info)
	}

	h.render(w, r, "social_providers", &pageData{
		Title:           "Social Providers",
		ActiveNav:       navSocial,
		Issuer:          h.issuer,
		SocialProviders: providers,
		Flash:           r.URL.Query().Get("flash"),
		Error:           r.URL.Query().Get("error"),
	})
}

// UpdateSocialProviderAction handles POST /admin/social/{provider}
func (h *AdminConsoleHandler) UpdateSocialProviderAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID
	provider := chi.URLParam(r, "provider")

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/social?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	clientID := strings.TrimSpace(r.FormValue("client_id"))
	clientSecret := strings.TrimSpace(r.FormValue("client_secret"))
	enabled := r.FormValue("enabled") == "on"

	cfg := &model.SocialProviderConfig{
		OrgID:        orgID,
		Provider:     provider,
		Enabled:      enabled,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		ExtraConfig:  make(map[string]string),
	}

	for _, def := range knownProviders {
		if def.name == provider {
			for _, f := range def.extraFields {
				if v := strings.TrimSpace(r.FormValue(f.Key)); v != "" {
					cfg.ExtraConfig[f.Key] = v
				}
			}
			break
		}
	}

	if err := h.store.UpsertSocialProviderConfig(ctx, cfg); err != nil {
		h.logger.Error("failed to save social provider config", "provider", provider, "error", err)
		http.Redirect(w, r, "/admin/social?error=Failed+to+save+configuration", http.StatusSeeOther)
		return
	}

	h.refreshSocialProvider(provider, cfg)
	http.Redirect(w, r, "/admin/social?flash=Provider+configuration+saved", http.StatusSeeOther)
}

// refreshSocialProvider updates the in-memory registry after a config change.
func (h *AdminConsoleHandler) refreshSocialProvider(provider string, cfg *model.SocialProviderConfig) {
	if !cfg.Enabled || cfg.ClientID == "" {
		h.socialRegistry.Unregister(provider)
		return
	}

	switch provider {
	case "google": //nolint:goconst // matches external provider name
		secret := cfg.ClientSecret
		if secret == "" {
			if existing, ok := h.socialRegistry.Get(provider); ok {
				if gp, isGoogle := existing.(*social.GoogleProvider); isGoogle {
					secret = gp.ClientSecret
				}
			}
		}
		h.socialRegistry.Register(&social.GoogleProvider{
			ClientID:     cfg.ClientID,
			ClientSecret: secret,
		})
	case "github": //nolint:goconst // matches external provider name
		secret := cfg.ClientSecret
		if secret == "" {
			if existing, ok := h.socialRegistry.Get(provider); ok {
				if gp, isGitHub := existing.(*social.GitHubProvider); isGitHub {
					secret = gp.ClientSecret
				}
			}
		}
		h.socialRegistry.Register(&social.GitHubProvider{
			ClientID:     cfg.ClientID,
			ClientSecret: secret,
		})
	case "apple":
		h.socialRegistry.Register(&social.AppleProvider{
			ClientID: cfg.ClientID,
			TeamID:   cfg.ExtraConfig["team_id"],
			KeyID:    cfg.ExtraConfig["key_id"],
		})
	}
}
