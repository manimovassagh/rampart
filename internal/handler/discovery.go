package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// DiscoveryResponse is the OIDC Discovery metadata (RFC 8414).
type DiscoveryResponse struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	UserinfoEndpoint                 string   `json:"userinfo_endpoint"`
	JWKSURI                          string   `json:"jwks_uri"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	GrantTypesSupported              []string `json:"grant_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                  []string `json:"scopes_supported"`
	ClaimsSupported                  []string `json:"claims_supported"`
	CodeChallengeMethodsSupported    []string `json:"code_challenge_methods_supported"`
}

// DiscoveryHandler returns the OIDC Discovery metadata JSON.
func DiscoveryHandler(issuer string, logger *slog.Logger) http.HandlerFunc {
	resp := DiscoveryResponse{
		Issuer:                           issuer,
		AuthorizationEndpoint:            issuer + "/oauth/authorize",
		TokenEndpoint:                    issuer + "/oauth/token",
		UserinfoEndpoint:                 issuer + "/me",
		JWKSURI:                          issuer + "/.well-known/jwks.json",
		ResponseTypesSupported:           []string{"code"},
		GrantTypesSupported:              []string{"authorization_code", "refresh_token"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
		ScopesSupported:                  []string{"openid", "profile", "email"},
		ClaimsSupported: []string{
			"sub", "iss", "iat", "exp",
			"preferred_username", "email", "email_verified",
			"given_name", "family_name", "org_id",
		},
		CodeChallengeMethodsSupported: []string{"S256"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		logger.Error("failed to marshal discovery response", "error", err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(data); err != nil {
			logger.Error("failed to write discovery response", "error", err)
		}
	}
}
