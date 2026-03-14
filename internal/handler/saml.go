package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"github.com/crewjam/saml"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/audit"
	"github.com/manimovassagh/rampart/internal/metrics"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
	"github.com/manimovassagh/rampart/internal/store"
	"github.com/manimovassagh/rampart/internal/token"
)

// SAMLStore defines the database operations required by SAMLHandler.
type SAMLStore interface {
	store.SAMLProviderStore
	store.SAMLRequestStore
	store.UserReader
	store.UserWriter
	store.OrgReader
	store.OrgSettingsReadWriter
	store.GroupReader
	store.SocialAccountStore
}

// SAMLHandler handles SAML SP endpoints for enterprise SSO.
type SAMLHandler struct {
	store      SAMLStore
	sessions   session.Store
	logger     *slog.Logger
	audit      *audit.Logger
	privateKey *rsa.PrivateKey
	cert       *x509.Certificate
	kid        string
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewSAMLHandler creates a new SAML handler.
func NewSAMLHandler(
	s SAMLStore,
	sessions session.Store,
	logger *slog.Logger,
	auditLogger *audit.Logger,
	privateKey *rsa.PrivateKey,
	cert *x509.Certificate,
	kid, issuer string,
	accessTTL, refreshTTL time.Duration,
) *SAMLHandler {
	return &SAMLHandler{
		store:      s,
		sessions:   sessions,
		logger:     logger,
		audit:      auditLogger,
		privateKey: privateKey,
		cert:       cert,
		kid:        kid,
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Metadata handles GET /saml/{providerID}/metadata — returns SP metadata XML.
func (h *SAMLHandler) Metadata(w http.ResponseWriter, r *http.Request) {
	providerID, err := uuid.Parse(chi.URLParam(r, "providerID"))
	if err != nil {
		apierror.BadRequest(w, "Invalid provider ID.")
		return
	}

	provider, err := h.store.GetSAMLProviderByID(r.Context(), providerID)
	if err != nil || provider == nil {
		apierror.NotFound(w)
		return
	}

	sp, err := h.buildSP(provider)
	if err != nil {
		h.logger.Error("failed to build SAML SP", "error", err)
		apierror.InternalError(w)
		return
	}

	metadata := sp.Metadata()
	w.Header().Set("Content-Type", "application/xml")
	if err := xml.NewEncoder(w).Encode(metadata); err != nil {
		h.logger.Error("failed to encode SAML metadata", "error", err)
	}
}

// InitiateLogin handles GET /saml/{providerID}/login — redirects to the IdP.
func (h *SAMLHandler) InitiateLogin(w http.ResponseWriter, r *http.Request) {
	providerID, err := uuid.Parse(chi.URLParam(r, "providerID"))
	if err != nil {
		apierror.BadRequest(w, "Invalid provider ID.")
		return
	}

	provider, err := h.store.GetSAMLProviderByID(r.Context(), providerID)
	if err != nil || provider == nil || !provider.Enabled {
		apierror.NotFound(w)
		return
	}

	sp, err := h.buildSP(provider)
	if err != nil {
		h.logger.Error("failed to build SAML SP", "error", err)
		apierror.InternalError(w)
		return
	}

	authnRequest, err := sp.MakeAuthenticationRequest(
		sp.GetSSOBindingLocation(saml.HTTPRedirectBinding),
		saml.HTTPRedirectBinding,
		saml.HTTPPostBinding,
	)
	if err != nil {
		h.logger.Error("failed to create SAML authn request", "error", err)
		apierror.InternalError(w)
		return
	}

	// Store the request ID so we can validate InResponseTo in the ACS callback.
	requestExpiry := time.Now().Add(10 * time.Minute)
	if err := h.store.StoreSAMLRequest(r.Context(), authnRequest.ID, providerID, requestExpiry); err != nil {
		h.logger.Error("failed to store SAML request ID", "error", err)
		apierror.InternalError(w)
		return
	}

	redirectURL, err := authnRequest.Redirect("", sp)
	if err != nil {
		h.logger.Error("failed to create SAML redirect URL", "error", err)
		apierror.InternalError(w)
		return
	}

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// ACS handles POST /saml/{providerID}/acs — Assertion Consumer Service callback.
func (h *SAMLHandler) ACS(w http.ResponseWriter, r *http.Request) {
	providerID, err := uuid.Parse(chi.URLParam(r, "providerID"))
	if err != nil {
		apierror.BadRequest(w, "Invalid provider ID.")
		return
	}

	ctx := r.Context()
	provider, err := h.store.GetSAMLProviderByID(ctx, providerID)
	if err != nil || provider == nil || !provider.Enabled {
		apierror.NotFound(w)
		return
	}

	sp, err := h.buildSP(provider)
	if err != nil {
		h.logger.Error("failed to build SAML SP", "error", err)
		apierror.InternalError(w)
		return
	}

	if err := r.ParseForm(); err != nil {
		apierror.BadRequest(w, "Invalid form data.")
		return
	}

	// Extract InResponseTo from the raw SAMLResponse to validate against stored request IDs.
	inResponseTo := extractInResponseTo(r.FormValue("SAMLResponse"))

	var possibleRequestIDs []string
	if inResponseTo != "" {
		valid, cErr := h.store.ConsumeSAMLRequest(ctx, inResponseTo, providerID)
		if cErr != nil {
			h.logger.Error("failed to validate SAML request ID", "error", cErr)
			apierror.InternalError(w)
			return
		}
		if !valid {
			h.logger.Warn("SAML response references unknown or expired request", "in_response_to", inResponseTo, "provider", provider.Name)
			apierror.Write(w, http.StatusForbidden, "saml_error", "SAML response references an unknown or expired request.")
			return
		}
		possibleRequestIDs = []string{inResponseTo}
	}

	assertion, err := sp.ParseResponse(r, possibleRequestIDs)
	if err != nil {
		h.logger.Error("failed to parse SAML response", "error", err, "provider", provider.Name)
		apierror.Write(w, http.StatusForbidden, "saml_error", "SAML authentication failed: "+err.Error())
		return
	}

	// Check for assertion replay — reject if this assertion ID was already consumed.
	assertionID := assertion.ID
	if assertionID != "" {
		consumed, aErr := h.store.IsSAMLAssertionConsumed(ctx, assertionID, providerID)
		if aErr != nil {
			h.logger.Error("failed to check SAML assertion replay", "error", aErr)
			apierror.InternalError(w)
			return
		}
		if consumed {
			h.logger.Warn("SAML assertion replay detected", "assertion_id", assertionID, "provider", provider.Name)
			apierror.Write(w, http.StatusForbidden, "saml_error", "SAML assertion has already been consumed.")
			return
		}
		// Record this assertion ID with a 10-minute expiry window.
		assertionExpiry := time.Now().Add(10 * time.Minute)
		if sErr := h.store.StoreSAMLAssertionID(ctx, assertionID, providerID, assertionExpiry); sErr != nil {
			h.logger.Error("failed to record SAML assertion ID", "error", sErr)
			// Non-fatal — continue processing but log the failure
		}
	}

	// Extract user attributes from the assertion
	email := h.extractAttribute(assertion, provider, "email")
	if email == "" {
		// Fall back to NameID if email attribute not found
		if assertion.Subject != nil && assertion.Subject.NameID != nil {
			email = assertion.Subject.NameID.Value
		}
	}
	if email == "" {
		h.logger.Error("no email in SAML assertion", "provider", provider.Name)
		apierror.Write(w, http.StatusForbidden, "saml_error", "No email address in SAML response.")
		return
	}

	givenName := h.extractAttribute(assertion, provider, "given_name")
	familyName := h.extractAttribute(assertion, provider, "family_name")
	username := h.extractAttribute(assertion, provider, "username")
	if username == "" {
		username = email
	}

	// Find or create the user
	user, err := h.store.GetUserByEmail(ctx, email, provider.OrgID)
	if err != nil {
		h.logger.Error("failed to look up user", "error", err)
		apierror.InternalError(w)
		return
	}

	if user == nil {
		// Auto-provision the user from SAML assertion
		newUser := &model.User{
			OrgID:         provider.OrgID,
			Username:      username,
			Email:         email,
			EmailVerified: true, // SAML IdP verified the identity
			GivenName:     givenName,
			FamilyName:    familyName,
			Enabled:       true,
			PasswordHash:  nil, // No local password — SAML-only user
		}

		user, err = h.store.CreateUser(ctx, newUser)
		if err != nil {
			h.logger.Error("failed to create SAML user", "error", err)
			apierror.InternalError(w)
			return
		}
		h.logger.Info("auto-provisioned SAML user", "email", email, "provider", provider.Name)
	}

	if !user.Enabled {
		apierror.Write(w, http.StatusForbidden, "account_disabled", "Account is disabled.")
		return
	}

	// Issue tokens
	accessTTL := h.accessTTL
	refreshTTL := h.refreshTTL
	if settings, sErr := h.store.GetOrgSettings(ctx, user.OrgID); sErr == nil && settings != nil {
		if settings.AccessTokenTTL > 0 {
			accessTTL = settings.AccessTokenTTL
		}
		if settings.RefreshTokenTTL > 0 {
			refreshTTL = settings.RefreshTokenTTL
		}
	}

	roles, _ := h.store.GetEffectiveUserRoles(ctx, user.ID)

	accessToken, err := token.GenerateAccessToken(
		h.privateKey, h.kid, h.issuer, h.issuer, accessTTL,
		user.ID, user.OrgID,
		user.Username, user.Email, user.EmailVerified,
		user.GivenName, user.FamilyName,
		roles...,
	)
	if err != nil {
		h.logger.Error("failed to generate access token", "error", err)
		apierror.InternalError(w)
		return
	}

	refreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		h.logger.Error("failed to generate refresh token", "error", err)
		apierror.InternalError(w)
		return
	}

	expiresAt := time.Now().Add(refreshTTL)
	if _, err := h.sessions.Create(ctx, user.ID, refreshToken, expiresAt); err != nil {
		h.logger.Error("failed to create session", "error", err)
		apierror.InternalError(w)
		return
	}

	if err := h.store.UpdateLastLoginAt(ctx, user.ID); err != nil {
		h.logger.Warn("failed to update last_login_at", "error", err)
	}

	h.audit.LogSimple(ctx, r, user.OrgID, model.EventUserLogin, &user.ID, user.Username, "user", user.ID.String(), user.Username)
	metrics.AuthTotal.WithLabelValues("success").Inc()
	metrics.TokensIssued.WithLabelValues("access").Inc()
	metrics.TokensIssued.WithLabelValues("refresh").Inc()
	metrics.ActiveSessions.Inc()

	// Return tokens as JSON (client-side will handle storage)
	resp := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenTypeBearer,
		ExpiresIn:    int(accessTTL.Seconds()),
		User:         user.ToResponse(),
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode SAML login response", "error", err)
	}
}

// ListProviders handles GET /saml/providers — returns available SAML providers for login.
func (h *SAMLHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	orgSlug := r.URL.Query().Get("org")
	if orgSlug == "" {
		apierror.BadRequest(w, "org query parameter is required.")
		return
	}

	ctx := r.Context()
	orgID, err := h.store.GetOrganizationIDBySlug(ctx, orgSlug)
	if err != nil {
		apierror.NotFound(w)
		return
	}

	providers, err := h.store.GetEnabledSAMLProviders(ctx, orgID)
	if err != nil {
		h.logger.Error("failed to list SAML providers", "error", err)
		apierror.InternalError(w)
		return
	}

	type providerResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		URL  string `json:"login_url"`
	}

	resp := make([]providerResponse, len(providers))
	for i, p := range providers {
		resp[i] = providerResponse{
			ID:   p.ID.String(),
			Name: p.Name,
			URL:  fmt.Sprintf("%s/saml/%s/login", h.issuer, p.ID),
		}
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode SAML providers response", "error", err)
	}
}

// buildSP constructs a crewjam/saml.ServiceProvider from the stored config.
func (h *SAMLHandler) buildSP(provider *model.SAMLProvider) (*saml.ServiceProvider, error) {
	metadataURL, _ := url.Parse(fmt.Sprintf("%s/saml/%s/metadata", h.issuer, provider.ID))
	acsURL, _ := url.Parse(fmt.Sprintf("%s/saml/%s/acs", h.issuer, provider.ID))

	// Parse IdP certificate
	block, _ := pem.Decode([]byte(provider.Certificate))
	if block == nil {
		// Try treating it as raw base64
		block = &pem.Block{Type: "CERTIFICATE", Bytes: []byte(provider.Certificate)}
	}
	idpCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing IdP certificate: %w", err)
	}

	ssoURL, _ := url.Parse(provider.SSOURL)

	idpMetadata := &saml.EntityDescriptor{
		EntityID: provider.EntityID,
		IDPSSODescriptors: []saml.IDPSSODescriptor{
			{
				SSODescriptor: saml.SSODescriptor{
					RoleDescriptor: saml.RoleDescriptor{
						KeyDescriptors: []saml.KeyDescriptor{
							{
								Use: "signing",
								KeyInfo: saml.KeyInfo{
									X509Data: saml.X509Data{
										X509Certificates: []saml.X509Certificate{
											{Data: base64.StdEncoding.EncodeToString(idpCert.Raw)},
										},
									},
								},
							},
						},
					},
				},
				SingleSignOnServices: []saml.Endpoint{
					{
						Binding:  saml.HTTPRedirectBinding,
						Location: ssoURL.String(),
					},
					{
						Binding:  saml.HTTPPostBinding,
						Location: ssoURL.String(),
					},
				},
			},
		},
	}

	sp := &saml.ServiceProvider{
		EntityID:    fmt.Sprintf("%s/saml/%s", h.issuer, provider.ID),
		Key:         h.privateKey,
		Certificate: h.cert,
		MetadataURL: *metadataURL,
		AcsURL:      *acsURL,
		IDPMetadata: idpMetadata,
	}

	return sp, nil
}

// extractAttribute extracts a SAML attribute using the provider's attribute mapping.
func (h *SAMLHandler) extractAttribute(assertion *saml.Assertion, provider *model.SAMLProvider, field string) string {
	// Check attribute mapping for custom attribute name
	attrName := field
	if mapped, ok := provider.AttributeMapping[field]; ok && mapped != "" {
		attrName = mapped
	}

	// Common attribute name aliases
	aliases := map[string][]string{
		"email":       {"email", "Email", "mail", "emailAddress", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"},
		"given_name":  {"givenName", "firstName", "given_name", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname"},
		"family_name": {"surname", "lastName", "family_name", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname"},
		"username":    {"uid", "username", "sAMAccountName", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"},
	}

	// Try mapped name first, then aliases
	names := []string{attrName}
	if aliasList, ok := aliases[field]; ok {
		names = append(names, aliasList...)
	}

	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			for _, name := range names {
				if attr.Name == name || attr.FriendlyName == name {
					if len(attr.Values) > 0 {
						return attr.Values[0].Value
					}
				}
			}
		}
	}

	return ""
}

// extractInResponseTo extracts the InResponseTo attribute from a base64-encoded SAMLResponse.
// Returns empty string if the attribute is not found or the response cannot be decoded.
func extractInResponseTo(samlResponse string) string {
	raw, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return ""
	}
	// Quick XML struct to extract just the InResponseTo attribute.
	var resp struct {
		InResponseTo string `xml:"InResponseTo,attr"`
	}
	if err := xml.Unmarshal(raw, &resp); err != nil {
		return ""
	}
	return resp.InResponseTo
}

// ParseCertFromKey generates a self-signed X.509 certificate from the RSA private key.
// This is used as the SP certificate for SAML request signing.
func ParseCertFromKey(key *rsa.PrivateKey) (*x509.Certificate, error) {
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		// Fall back to using a TLS cert pair
		tlsCert, tlsErr := tls.X509KeyPair(
			pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}),
			pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}),
		)
		if tlsErr != nil {
			return nil, fmt.Errorf("creating cert: %w", err)
		}
		return x509.ParseCertificate(tlsCert.Certificate[0])
	}

	return x509.ParseCertificate(certDER)
}
