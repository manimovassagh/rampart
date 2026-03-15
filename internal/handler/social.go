package handler

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/audit"
	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/database"
	"github.com/manimovassagh/rampart/internal/metrics"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/oauth"
	"github.com/manimovassagh/rampart/internal/social"
	"github.com/manimovassagh/rampart/internal/store"
)

// errSocialEmailNotVerified is returned when a social provider reports an
// unverified email that matches an existing user account. Auto-linking is
// refused to prevent account takeover.
var errSocialEmailNotVerified = errors.New("social provider email is not verified")

const (
	socialCookieName   = "_rampart_social"
	socialCookieMaxAge = 600
	socialStateBytelen = 32
	socialCookiePath   = "/oauth/social"
)

// SocialStore defines the database operations required by SocialHandler.
type SocialStore interface {
	store.OrgReader
	store.UserReader
	store.UserWriter
	store.SocialAccountStore
	store.AuthCodeStore
	store.OAuthClientReader
	store.OrgSettingsReadWriter
}

// SocialHandler handles social login initiation and callback.
type SocialHandler struct {
	store    SocialStore
	registry *social.Registry
	logger   *slog.Logger
	audit    *audit.Logger
	hmacKey  []byte
	issuer   string
}

// NewSocialHandler creates a new social login handler.
func NewSocialHandler(s SocialStore, registry *social.Registry, logger *slog.Logger, auditLogger *audit.Logger, hmacKey []byte, issuer string) *SocialHandler {
	return &SocialHandler{
		store:    s,
		registry: registry,
		logger:   logger,
		audit:    auditLogger,
		hmacKey:  hmacKey,
		issuer:   issuer,
	}
}

// socialCookiePayload holds the original OAuth flow parameters stored in the signed cookie.
type socialCookiePayload struct {
	ClientID      string `json:"client_id"`
	RedirectURI   string `json:"redirect_uri"`
	Scope         string `json:"scope"`
	State         string `json:"state"`
	CodeChallenge string `json:"code_challenge"`
	ProviderState string `json:"provider_state"`
	Timestamp     int64  `json:"timestamp"`
}

// InitiateLogin handles GET /oauth/social/{provider} — redirects the user to the social provider.
func (h *SocialHandler) InitiateLogin(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, ok := h.registry.Get(providerName)
	if !ok {
		http.Error(w, "Unknown social provider.", http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	scope := q.Get("scope")
	state := q.Get("state")
	codeChallenge := q.Get("code_challenge")

	if clientID == "" || redirectURI == "" {
		http.Error(w, "Missing required parameters: client_id and redirect_uri.", http.StatusBadRequest)
		return
	}

	if state == "" {
		http.Error(w, "Missing required parameter: state.", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	client, err := h.store.GetOAuthClient(ctx, clientID)
	if err != nil {
		h.logger.Error("failed to fetch oauth client", "error", err)
		http.Error(w, msgUnexpectedErr, http.StatusInternalServerError)
		return
	}
	if client == nil {
		http.Error(w, "Unknown client_id.", http.StatusBadRequest)
		return
	}

	if !database.ValidateRedirectURI(client, redirectURI) {
		http.Error(w, "Invalid redirect_uri.", http.StatusBadRequest)
		return
	}

	if scope == "" {
		scope = scopeOpenID
	}

	// Generate a random provider state for CSRF protection on the callback
	providerState, err := generateRandomState()
	if err != nil {
		h.logger.Error("failed to generate provider state", "error", err)
		http.Error(w, msgUnexpectedErr, http.StatusInternalServerError)
		return
	}

	// Store original OAuth flow params in a signed cookie
	payload := socialCookiePayload{
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		Scope:         scope,
		State:         state,
		CodeChallenge: codeChallenge,
		ProviderState: providerState,
		Timestamp:     time.Now().Unix(),
	}

	cookieValue, err := h.signCookiePayload(&payload)
	if err != nil {
		h.logger.Error("failed to sign social cookie", "error", err)
		http.Error(w, msgUnexpectedErr, http.StatusInternalServerError)
		return
	}

	// SameSite=None is required because Apple Sign In uses response_mode=form_post,
	// which sends a cross-site POST back to our callback. SameSite=Lax would block
	// the cookie on cross-site POST requests. Secure=true is mandatory with SameSite=None.
	http.SetCookie(w, &http.Cookie{
		Name:     socialCookieName,
		Value:    cookieValue,
		Path:     socialCookiePath,
		MaxAge:   socialCookieMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})

	// Build the callback URL for the social provider
	callbackURL := h.buildCallbackURL(providerName)

	authURL := provider.AuthURL(providerState, callbackURL)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback handles GET or POST /oauth/social/{provider}/callback — exchanges the code and completes the flow.
// Apple Sign In uses response_mode=form_post and sends code/state as POST form data,
// while other providers use GET query parameters.
func (h *SocialHandler) Callback(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, ok := h.registry.Get(providerName)
	if !ok {
		http.Error(w, "Unknown social provider.", http.StatusBadRequest)
		return
	}

	// Read code and state from either query params (GET) or form body (POST).
	var code, returnedState string
	if r.Method == http.MethodPost {
		// Apple uses form_post: code and state arrive in the POST body.
		code = r.FormValue("code")
		returnedState = r.FormValue("state")
	} else {
		q := r.URL.Query()
		code = q.Get("code")
		returnedState = q.Get("state")
	}

	if code == "" || returnedState == "" {
		http.Error(w, "Missing code or state parameter.", http.StatusBadRequest)
		return
	}

	// Validate the signed cookie
	cookie, err := r.Cookie(socialCookieName)
	if err != nil {
		http.Error(w, "Missing social login state cookie.", http.StatusBadRequest)
		return
	}

	payload, err := h.verifyCookiePayload(cookie.Value)
	if err != nil {
		h.logger.Warn("invalid social cookie signature", "error", err)
		http.Error(w, "Invalid or expired social login state.", http.StatusBadRequest)
		return
	}

	// Verify state matches what we stored
	if returnedState != payload.ProviderState {
		h.logger.Warn("social login state mismatch", "expected", payload.ProviderState, "got", returnedState)
		http.Error(w, "Invalid social login state.", http.StatusBadRequest)
		return
	}

	// Clear the cookie
	http.SetCookie(w, &http.Cookie{
		Name:     socialCookieName,
		Value:    "",
		Path:     socialCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	ctx := r.Context()

	// Exchange the authorization code with the social provider
	callbackURL := h.buildCallbackURL(providerName)
	userInfo, err := provider.Exchange(ctx, code, callbackURL)
	if err != nil {
		h.logger.Error("social provider exchange failed", "provider", providerName, "error", err)
		h.audit.Log(ctx, r, uuid.Nil, model.EventSocialLoginFailed, nil, "", "social", "", providerName, map[string]any{"reason": "exchange_failed", "provider": providerName})
		metrics.AuthTotal.WithLabelValues("failure").Inc()
		http.Error(w, "Failed to authenticate with social provider.", http.StatusBadGateway)
		return
	}

	if userInfo.Email == "" {
		h.logger.Warn("social provider returned no email", "provider", providerName)
		http.Error(w, "Social provider did not return an email address.", http.StatusBadRequest)
		return
	}

	// Resolve user via account linking logic
	user, orgID, err := h.resolveUser(ctx, r, providerName, userInfo)
	if err != nil {
		if errors.Is(err, errSocialEmailNotVerified) {
			h.logger.Warn("social login blocked: email not verified at provider", "provider", providerName, "email", userInfo.Email)
			h.audit.Log(ctx, r, uuid.Nil, model.EventSocialLoginFailed, nil, "", "social", "", providerName, map[string]any{"reason": "email_not_verified", "provider": providerName})
			metrics.AuthTotal.WithLabelValues("failure").Inc()
			http.Error(w, "Your email address has not been verified by the social provider. Please verify your email and try again.", http.StatusForbidden)
			return
		}
		h.logger.Error("failed to resolve social user", "error", err)
		http.Error(w, msgUnexpectedErr, http.StatusInternalServerError)
		return
	}

	// Generate Rampart authorization code
	authCode, err := oauth.GenerateAuthorizationCode()
	if err != nil {
		h.logger.Error("failed to generate authorization code", "error", err)
		http.Error(w, msgUnexpectedErr, http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(authCodeTTL)
	if err := h.store.StoreAuthorizationCode(ctx, authCode, payload.ClientID, user.ID, orgID, payload.RedirectURI, payload.CodeChallenge, payload.Scope, "", expiresAt); err != nil {
		h.logger.Error("failed to store authorization code", "error", err)
		http.Error(w, msgUnexpectedErr, http.StatusInternalServerError)
		return
	}

	if err := h.store.UpdateLastLoginAt(ctx, user.ID); err != nil {
		h.logger.Warn("failed to update last_login_at", "error", err, "user_id", user.ID)
	}

	h.audit.Log(ctx, r, orgID, model.EventSocialLogin, &user.ID, user.Username, "user", user.ID.String(), user.Username, map[string]any{"provider": providerName, "client_id": payload.ClientID})
	metrics.AuthTotal.WithLabelValues("success").Inc()

	// Redirect back to the client with the authorization code and original state
	params := url.Values{"code": {authCode}, "state": {payload.State}}
	http.Redirect(w, r, payload.RedirectURI+"?"+params.Encode(), http.StatusFound)
}

// resolveUser implements the account linking logic:
// 1. Check if social_account exists for this provider+provider_user_id
// 2. If not, check if a user with this email exists in the default org
// 3. If no user exists, create a new user
func (h *SocialHandler) resolveUser(ctx context.Context, r *http.Request, providerName string, userInfo *social.UserInfo) (*model.User, uuid.UUID, error) {
	// Step 1: Check for existing social account
	existing, err := h.store.GetSocialAccount(ctx, providerName, userInfo.ProviderUserID)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("looking up social account: %w", err)
	}

	if existing != nil {
		// Social account already linked — fetch the user
		orgID, err := h.store.GetDefaultOrganizationID(ctx)
		if err != nil {
			return nil, uuid.Nil, fmt.Errorf("getting default org: %w", err)
		}
		user, err := h.store.GetUserByEmail(ctx, strings.ToLower(userInfo.Email), orgID)
		if err != nil {
			return nil, uuid.Nil, fmt.Errorf("looking up user by email: %w", err)
		}
		if user != nil {
			return user, orgID, nil
		}
		// Edge case: social account exists but user was deleted — fall through to create
	}

	// Step 2: Get default org
	orgID, err := h.store.GetDefaultOrganizationID(ctx)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("getting default org: %w", err)
	}

	// Step 3: Check if a user with this email already exists
	user, err := h.store.GetUserByEmail(ctx, strings.ToLower(userInfo.Email), orgID)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("looking up user by email: %w", err)
	}

	if user != nil {
		// Prevent account takeover: only auto-link when the social provider
		// has verified the email address. Without this check an attacker
		// could register with a victim's email at a provider that does not
		// verify emails and gain access to the victim's account.
		if !userInfo.EmailVerified {
			return nil, uuid.Nil, errSocialEmailNotVerified
		}

		// Link social account to existing user
		socialAccount := &model.SocialAccount{
			UserID:         user.ID,
			Provider:       providerName,
			ProviderUserID: userInfo.ProviderUserID,
			Email:          userInfo.Email,
			Name:           userInfo.Name,
			AvatarURL:      userInfo.AvatarURL,
			AccessToken:    userInfo.AccessToken,
			RefreshToken:   userInfo.RefreshToken,
			TokenExpiresAt: userInfo.TokenExpiresAt,
		}
		if _, err := h.store.CreateSocialAccount(ctx, socialAccount); err != nil {
			return nil, uuid.Nil, fmt.Errorf("linking social account: %w", err)
		}
		h.audit.Log(ctx, r, orgID, model.EventSocialAccountLinked, &user.ID, user.Username, "social_account", providerName, userInfo.Email, map[string]any{"provider": providerName})
		return user, orgID, nil
	}

	// Step 4: Create a new user
	// Generate a random, unguessable password hash so the social-only user can
	// never authenticate via the password login flow.  Without this, the
	// password_hash column would be empty, which could allow a bypass if
	// VerifyPassword mishandles empty hashes.
	randomToken := make([]byte, 64)
	if _, err := rand.Read(randomToken); err != nil {
		return nil, uuid.Nil, fmt.Errorf("generating random password token: %w", err)
	}
	randomHash, err := auth.HashPassword(base64.RawURLEncoding.EncodeToString(randomToken))
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("hashing random password: %w", err)
	}

	username := deriveUsername(userInfo.Email)
	newUser := &model.User{
		OrgID:         orgID,
		Username:      username,
		Email:         strings.ToLower(userInfo.Email),
		PasswordHash:  []byte(randomHash),
		EmailVerified: userInfo.EmailVerified,
		GivenName:     userInfo.GivenName,
		FamilyName:    userInfo.FamilyName,
		Picture:       userInfo.AvatarURL,
		Enabled:       true,
	}

	createdUser, err := h.store.CreateUser(ctx, newUser)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("creating user: %w", err)
	}

	// Link the social account to the new user
	socialAccount := &model.SocialAccount{
		UserID:         createdUser.ID,
		Provider:       providerName,
		ProviderUserID: userInfo.ProviderUserID,
		Email:          userInfo.Email,
		Name:           userInfo.Name,
		AvatarURL:      userInfo.AvatarURL,
		AccessToken:    userInfo.AccessToken,
		RefreshToken:   userInfo.RefreshToken,
		TokenExpiresAt: userInfo.TokenExpiresAt,
	}
	if _, err := h.store.CreateSocialAccount(ctx, socialAccount); err != nil {
		return nil, uuid.Nil, fmt.Errorf("creating social account: %w", err)
	}

	h.audit.Log(ctx, r, orgID, model.EventSocialAccountLinked, &createdUser.ID, createdUser.Username, "social_account", providerName, userInfo.Email, map[string]any{"provider": providerName, "new_user": true})

	return createdUser, orgID, nil
}

// signCookiePayload encodes and signs the cookie payload using HMAC-SHA256.
// Format: base64(json) + "." + hex(hmac-sha256)
func (h *SocialHandler) signCookiePayload(payload *socialCookiePayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling cookie payload: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	sig := computeHMAC([]byte(encoded), h.hmacKey)
	return encoded + "." + hex.EncodeToString(sig), nil
}

// verifyCookiePayload verifies the HMAC signature and decodes the cookie payload.
func (h *SocialHandler) verifyCookiePayload(cookieValue string) (*socialCookiePayload, error) {
	parts := strings.SplitN(cookieValue, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed cookie value")
	}
	encoded := parts[0]
	sigHex := parts[1]

	expectedSig := computeHMAC([]byte(encoded), h.hmacKey)
	actualSig, err := hex.DecodeString(sigHex)
	if err != nil {
		return nil, fmt.Errorf("decoding signature: %w", err)
	}

	if !hmac.Equal(expectedSig, actualSig) {
		return nil, fmt.Errorf("signature mismatch")
	}

	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}

	var payload socialCookiePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling payload: %w", err)
	}

	// Reject cookies older than 10 minutes to prevent replay attacks
	if payload.Timestamp == 0 || time.Now().Unix()-payload.Timestamp >= socialCookieMaxAge {
		return nil, fmt.Errorf("cookie expired")
	}

	return &payload, nil
}

// buildCallbackURL constructs the callback URL for a social provider.
func (h *SocialHandler) buildCallbackURL(providerName string) string {
	return h.issuer + "/oauth/social/" + providerName + "/callback"
}

// generateRandomState generates a cryptographically random URL-safe state string.
func generateRandomState() (string, error) {
	b := make([]byte, socialStateBytelen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// computeHMAC computes HMAC-SHA256 of the message with the given key.
func computeHMAC(message, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

// deriveUsername extracts a username from an email address (the part before @).
func deriveUsername(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return email
}
