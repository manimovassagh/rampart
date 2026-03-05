package handler

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/oauth"
	"github.com/manimovassagh/rampart/internal/session"
	"github.com/manimovassagh/rampart/internal/token"
)

// TokenStore defines the database operations required by TokenHandler.
type TokenStore interface {
	GetOAuthClient(ctx context.Context, clientID string) (*model.OAuthClient, error)
	ConsumeAuthorizationCode(ctx context.Context, code string) (*model.AuthorizationCode, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetOrgSettings(ctx context.Context, orgID uuid.UUID) (*model.OrgSettings, error)
	GetEffectiveUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
}

// TokenHandler handles the OAuth 2.0 token endpoint.
type TokenHandler struct {
	store      TokenStore
	sessions   session.Store
	logger     *slog.Logger
	privateKey *rsa.PrivateKey
	kid        string
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewTokenHandler creates a new token endpoint handler.
func NewTokenHandler(store TokenStore, sessions session.Store, logger *slog.Logger, privateKey *rsa.PrivateKey, kid, issuer string, accessTTL, refreshTTL time.Duration) *TokenHandler {
	return &TokenHandler{
		store:      store,
		sessions:   sessions,
		logger:     logger,
		privateKey: privateKey,
		kid:        kid,
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// TokenResponse is the OAuth 2.0 token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Token handles POST /oauth/token (authorization code exchange).
// Expects application/x-www-form-urlencoded per RFC 6749.
func (h *TokenHandler) Token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.Write(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid form data.")
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType != "authorization_code" {
		h.writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "Only authorization_code grant type is supported on this endpoint.")
		return
	}

	code := r.FormValue("code")
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	codeVerifier := r.FormValue("code_verifier")

	if code == "" || clientID == "" || redirectURI == "" || codeVerifier == "" {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Missing required parameters: code, client_id, redirect_uri, code_verifier.")
		return
	}

	ctx := r.Context()

	// Verify client exists
	client, err := h.store.GetOAuthClient(ctx, clientID)
	if err != nil {
		h.logger.Error("failed to fetch oauth client", "error", err)
		h.writeOAuthError(w, http.StatusInternalServerError, oauthServerError, msgInternalServer)
		return
	}
	if client == nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_client", "Unknown client_id.")
		return
	}

	// Consume the authorization code (atomic single-use)
	authCode, err := h.store.ConsumeAuthorizationCode(ctx, code)
	if err != nil {
		h.logger.Error("failed to consume authorization code", "error", err)
		h.writeOAuthError(w, http.StatusInternalServerError, oauthServerError, msgInternalServer)
		return
	}
	if authCode == nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "Invalid, expired, or already-used authorization code.")
		return
	}

	// Verify client_id matches
	if authCode.ClientID != clientID {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "client_id does not match the authorization code.")
		return
	}

	// Verify redirect_uri matches exactly
	if authCode.RedirectURI != redirectURI {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "redirect_uri does not match the authorization code.")
		return
	}

	// Validate PKCE
	if !oauth.ValidatePKCE(codeVerifier, authCode.CodeChallenge) {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "Invalid code_verifier.")
		return
	}

	// Fetch user
	user, err := h.store.GetUserByID(ctx, authCode.UserID)
	if err != nil {
		h.logger.Error("failed to fetch user for token exchange", "error", err)
		h.writeOAuthError(w, http.StatusInternalServerError, oauthServerError, msgInternalServer)
		return
	}
	if user == nil || !user.Enabled {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "User account is disabled or not found.")
		return
	}

	// Resolve per-org TTLs
	accessTTL := h.accessTTL
	refreshTTL := h.refreshTTL
	if settings, sErr := h.store.GetOrgSettings(ctx, authCode.OrgID); sErr != nil {
		h.logger.Warn("failed to fetch org settings, using defaults", "error", sErr)
	} else if settings != nil {
		if settings.AccessTokenTTL > 0 {
			accessTTL = settings.AccessTokenTTL
		}
		if settings.RefreshTokenTTL > 0 {
			refreshTTL = settings.RefreshTokenTTL
		}
	}

	// Fetch user roles (excluding internal admin role for external clients)
	roles, rErr := h.store.GetEffectiveUserRoles(ctx, user.ID)
	if rErr != nil {
		h.logger.Warn("failed to fetch user roles for token", "error", rErr)
		roles = nil
	}
	if authCode.ClientID != adminClientID {
		roles = filterInternalRoles(roles)
	}

	// Generate access token
	accessToken, err := token.GenerateAccessToken(
		h.privateKey, h.kid, h.issuer, accessTTL,
		user.ID, user.OrgID,
		user.Username, user.Email, user.EmailVerified,
		user.GivenName, user.FamilyName,
		roles...,
	)
	if err != nil {
		h.logger.Error("failed to generate access token", "error", err)
		h.writeOAuthError(w, http.StatusInternalServerError, oauthServerError, msgInternalServer)
		return
	}

	// Generate refresh token
	refreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		h.logger.Error("failed to generate refresh token", "error", err)
		h.writeOAuthError(w, http.StatusInternalServerError, oauthServerError, msgInternalServer)
		return
	}

	// Store session
	expiresAt := time.Now().Add(refreshTTL)
	if _, err := h.sessions.Create(ctx, user.ID, refreshToken, expiresAt); err != nil {
		h.logger.Error("failed to create session", "error", err)
		h.writeOAuthError(w, http.StatusInternalServerError, oauthServerError, msgInternalServer)
		return
	}

	resp := TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenTypeBearer,
		ExpiresIn:    int(accessTTL.Seconds()),
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.Header().Set("Cache-Control", cacheNoStore)
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode token response", "error", err)
	}
}

// writeOAuthError writes an OAuth 2.0 error response per RFC 6749 §5.2.
func (h *TokenHandler) writeOAuthError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.Header().Set("Cache-Control", cacheNoStore)
	w.WriteHeader(status)

	resp := map[string]string{
		"error":             code,
		"error_description": description,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode oauth error response", "error", err)
	}
}

// internalRoles are roles that should not be exposed to external OAuth clients.
var internalRoles = map[string]bool{"admin": true}

// filterInternalRoles removes internal-only roles (like admin) from the role list.
// External OAuth clients should not receive admin privileges — admin access is only
// available through the Rampart admin console itself, similar to Keycloak's approach.
func filterInternalRoles(roles []string) []string {
	filtered := make([]string, 0, len(roles))
	for _, r := range roles {
		if !internalRoles[r] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
