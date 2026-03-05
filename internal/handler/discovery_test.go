package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const contentTypeJSON = "application/json"

func TestDiscoveryReturnsMetadata(t *testing.T) {
	h := DiscoveryHandler("https://auth.example.com", noopLogger())

	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", http.NoBody)
	w := httptest.NewRecorder()

	h(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	ct := w.Header().Get("Content-Type")
	if ct != contentTypeJSON {
		t.Errorf("Content-Type = %q, want %s", ct, contentTypeJSON)
	}

	cc := w.Header().Get("Cache-Control")
	if cc != "public, max-age=3600" {
		t.Errorf("Cache-Control = %q, want public, max-age=3600", cc)
	}

	var resp DiscoveryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Issuer != "https://auth.example.com" {
		t.Errorf("issuer = %q, want https://auth.example.com", resp.Issuer)
	}
	if resp.JWKSURI != "https://auth.example.com/.well-known/jwks.json" {
		t.Errorf("jwks_uri = %q, want https://auth.example.com/.well-known/jwks.json", resp.JWKSURI)
	}
	if resp.AuthorizationEndpoint != "https://auth.example.com/oauth/authorize" {
		t.Errorf("authorization_endpoint = %q, want https://auth.example.com/oauth/authorize", resp.AuthorizationEndpoint)
	}
	if resp.TokenEndpoint != "https://auth.example.com/oauth/token" {
		t.Errorf("token_endpoint = %q, want https://auth.example.com/oauth/token", resp.TokenEndpoint)
	}
	if resp.UserinfoEndpoint != "https://auth.example.com/me" {
		t.Errorf("userinfo_endpoint = %q, want https://auth.example.com/me", resp.UserinfoEndpoint)
	}
}

func TestDiscoveryIncludesRS256(t *testing.T) {
	h := DiscoveryHandler("http://localhost:8080", noopLogger())

	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", http.NoBody)
	w := httptest.NewRecorder()

	h(w, req)

	var resp DiscoveryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	found := false
	for _, alg := range resp.IDTokenSigningAlgValuesSupported {
		if alg == "RS256" {
			found = true
			break
		}
	}
	if !found {
		t.Error("RS256 not in id_token_signing_alg_values_supported")
	}
}

func TestDiscoveryClaimsSupported(t *testing.T) {
	h := DiscoveryHandler("http://localhost:8080", noopLogger())

	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", http.NoBody)
	w := httptest.NewRecorder()

	h(w, req)

	var resp DiscoveryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	required := []string{"sub", "iss", "email", "preferred_username", "org_id"}
	claimSet := make(map[string]bool)
	for _, c := range resp.ClaimsSupported {
		claimSet[c] = true
	}

	for _, claim := range required {
		if !claimSet[claim] {
			t.Errorf("missing required claim: %s", claim)
		}
	}
}

func TestDiscoveryIsDeterministic(t *testing.T) {
	h := DiscoveryHandler("http://localhost:8080", noopLogger())

	w1 := httptest.NewRecorder()
	h(w1, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	w2 := httptest.NewRecorder()
	h(w2, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	if w1.Body.String() != w2.Body.String() {
		t.Error("discovery response should be deterministic")
	}
}
