package handler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/oauth"
	"github.com/manimovassagh/rampart/internal/session"
	"github.com/manimovassagh/rampart/internal/token"
)

const adminClientID = "rampart-admin"

// AdminLoginStore defines database operations for admin login flow.
type AdminLoginStore interface {
	GetOAuthClient(ctx context.Context, clientID string) (*model.OAuthClient, error)
	ConsumeAuthorizationCode(ctx context.Context, code string) (*model.AuthorizationCode, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetOrgSettings(ctx context.Context, orgID uuid.UUID) (*model.OrgSettings, error)
}

// AdminLoginHandler handles the admin OAuth login flow.
type AdminLoginHandler struct {
	store      AdminLoginStore
	sessions   session.Store
	logger     *slog.Logger
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	hmacKey    []byte
}

// NewAdminLoginHandler creates a new admin login handler.
func NewAdminLoginHandler(
	store AdminLoginStore,
	sessions session.Store,
	logger *slog.Logger,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	kid, issuer string,
	accessTTL, refreshTTL time.Duration,
	hmacKey []byte,
) *AdminLoginHandler {
	return &AdminLoginHandler{
		store:      store,
		sessions:   sessions,
		logger:     logger,
		privateKey: privateKey,
		publicKey:  publicKey,
		kid:        kid,
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		hmacKey:    hmacKey,
	}
}

// Login redirects to the OAuth authorization endpoint with PKCE.
func (h *AdminLoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Generate PKCE code verifier + challenge
	verifier, err := generateCodeVerifier()
	if err != nil {
		h.logger.Error("failed to generate code verifier", "error", err)
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
		return
	}
	challenge := oauth.ComputeS256Challenge(verifier)

	// Generate state
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		h.logger.Error("failed to generate state", "error", err)
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
		return
	}
	state := hex.EncodeToString(stateBytes)

	// Store verifier+state in a short-lived cookie so callback can use them
	cookieVal := state + "." + verifier
	http.SetCookie(w, &http.Cookie{
		Name:     "rampart_admin_oauth",
		Value:    cookieVal,
		Path:     "/admin/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	redirectURI := h.issuer + "/admin/callback"
	authURL := fmt.Sprintf(
		"%s/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=openid&state=%s&code_challenge=%s&code_challenge_method=S256",
		h.issuer, adminClientID, redirectURI, state, challenge,
	)

	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback handles the OAuth callback, exchanges the code for tokens, and sets the session cookie.
func (h *AdminLoginHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		h.logger.Warn("admin callback missing code or state")
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	// Retrieve stored state+verifier from cookie
	oauthCookie, err := r.Cookie("rampart_admin_oauth")
	if err != nil || oauthCookie.Value == "" {
		h.logger.Warn("admin callback missing oauth cookie")
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	// Clear the oauth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "rampart_admin_oauth",
		Value:    "",
		Path:     "/admin/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Parse state.verifier
	parts := splitFirst(oauthCookie.Value, '.')
	if len(parts) != 2 {
		h.logger.Warn("admin callback invalid oauth cookie format")
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	storedState := parts[0]
	codeVerifier := parts[1]

	if state != storedState {
		h.logger.Warn("admin callback state mismatch")
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	// Exchange code for tokens internally (no HTTP call)
	ctx := r.Context()

	// Consume the authorization code
	authCode, err := h.store.ConsumeAuthorizationCode(ctx, code)
	if err != nil {
		h.logger.Error("failed to consume authorization code", "error", err)
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}
	if authCode == nil {
		h.logger.Warn("admin callback invalid or expired authorization code")
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	// Verify client_id matches
	if authCode.ClientID != adminClientID {
		h.logger.Warn("admin callback client_id mismatch")
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	// Verify redirect_uri matches
	expectedRedirect := h.issuer + "/admin/callback"
	if authCode.RedirectURI != expectedRedirect {
		h.logger.Warn("admin callback redirect_uri mismatch")
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	// Validate PKCE
	if !oauth.ValidatePKCE(codeVerifier, authCode.CodeChallenge) {
		h.logger.Warn("admin callback PKCE validation failed")
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	// Fetch user
	user, err := h.store.GetUserByID(ctx, authCode.UserID)
	if err != nil || user == nil || !user.Enabled {
		h.logger.Error("failed to fetch user for admin login", "error", err)
		http.Redirect(w, r, "/admin/login", http.StatusFound)
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

	// Generate access token
	accessToken, err := token.GenerateAccessToken(
		h.privateKey, h.kid, h.issuer, accessTTL,
		user.ID, user.OrgID,
		user.Username, user.Email, user.EmailVerified,
		user.GivenName, user.FamilyName,
	)
	if err != nil {
		h.logger.Error("failed to generate access token", "error", err)
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
		return
	}

	// Generate refresh token and store session
	refreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		h.logger.Error("failed to generate refresh token", "error", err)
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(refreshTTL)
	if _, err := h.sessions.Create(ctx, user.ID, refreshToken, expiresAt); err != nil {
		h.logger.Error("failed to create session", "error", err)
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
		return
	}

	// Set the session cookie with the access token
	middleware.SetAdminSession(w, accessToken, h.hmacKey, int(accessTTL.Seconds()))

	http.Redirect(w, r, "/admin/", http.StatusFound)
}

// Logout clears the admin session and redirects to login.
func (h *AdminLoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	middleware.ClearAdminSession(w)
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating code verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func splitFirst(s string, sep byte) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

