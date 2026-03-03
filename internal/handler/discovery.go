package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// DiscoveryResponse is the OIDC Discovery metadata (RFC 8414).
type DiscoveryResponse struct {
	Issuer                           string   `json:"issuer"`
	JWKSURI                          string   `json:"jwks_uri"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	UserinfoEndpoint                 string   `json:"userinfo_endpoint"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                  []string `json:"scopes_supported"`
	ClaimsSupported                  []string `json:"claims_supported"`
}

// DiscoveryHandler returns the OIDC Discovery metadata JSON.
func DiscoveryHandler(issuer string, logger *slog.Logger) http.HandlerFunc {
	resp := DiscoveryResponse{
		Issuer:                           issuer,
		JWKSURI:                          issuer + "/.well-known/jwks.json",
		TokenEndpoint:                    issuer + "/login",
		UserinfoEndpoint:                 issuer + "/me",
		ResponseTypesSupported:           []string{"id_token"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
		ScopesSupported:                  []string{"openid", "profile", "email"},
		ClaimsSupported: []string{
			"sub", "iss", "iat", "exp",
			"preferred_username", "email", "email_verified",
			"given_name", "family_name", "org_id",
		},
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
