package handler

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/audit"
	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/database"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
	"github.com/manimovassagh/rampart/internal/token"
)

const (
	invalidCredentialsMsg      = "Invalid credentials."
	defaultMaxFailedAttempts   = 5
	defaultLockoutDurationMins = 15
)

// defaultLockoutPolicy returns the lockout policy from org settings, or defaults.
func defaultLockoutPolicy(settings *model.OrgSettings) (maxAttempts int, lockoutDuration time.Duration) {
	maxAttempts = defaultMaxFailedAttempts
	lockoutDuration = defaultLockoutDurationMins * time.Minute
	if settings != nil {
		if settings.MaxFailedLoginAttempts > 0 {
			maxAttempts = settings.MaxFailedLoginAttempts
		}
		if settings.LockoutDuration > 0 {
			lockoutDuration = settings.LockoutDuration
		}
	}
	return
}

// LoginRequest is the expected JSON body for POST /login.
type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
	OrgSlug    string `json:"org_slug,omitempty"`
}

// LoginResponse is returned on successful authentication.
type LoginResponse struct {
	AccessToken  string              `json:"access_token"`
	RefreshToken string              `json:"refresh_token"`
	TokenType    string              `json:"token_type"`
	ExpiresIn    int                 `json:"expires_in"`
	User         *model.UserResponse `json:"user"`
}

// RefreshRequest is the expected JSON body for POST /token/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse is returned on successful token refresh.
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// LogoutRequest is the expected JSON body for POST /logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LoginStore defines the database operations required by LoginHandler.
type LoginStore interface {
	GetDefaultOrganizationID(ctx context.Context) (uuid.UUID, error)
	GetOrganizationIDBySlug(ctx context.Context, slug string) (uuid.UUID, error)
	GetUserByEmail(ctx context.Context, email string, orgID uuid.UUID) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string, orgID uuid.UUID) (*model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateLastLoginAt(ctx context.Context, userID uuid.UUID) error
	GetOrgSettings(ctx context.Context, orgID uuid.UUID) (*model.OrgSettings, error)
	GetEffectiveUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
	IncrementFailedLogins(ctx context.Context, userID uuid.UUID, maxAttempts int, lockoutDuration time.Duration) error
	ResetFailedLogins(ctx context.Context, userID uuid.UUID) error
	GetVerifiedMFADevice(ctx context.Context, userID uuid.UUID) (*database.MFADevice, error)
	ConsumeBackupCode(ctx context.Context, userID uuid.UUID, codeHash []byte) (bool, error)
}

// LoginHandler handles authentication endpoints.
type LoginHandler struct {
	store      LoginStore
	sessions   session.Store
	logger     *slog.Logger
	audit      *audit.Logger
	privateKey *rsa.PrivateKey
	kid        string
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewLoginHandler creates a handler with all authentication dependencies.
func NewLoginHandler(store LoginStore, sessions session.Store, logger *slog.Logger, auditLogger *audit.Logger, privateKey *rsa.PrivateKey, kid, issuer string, accessTTL, refreshTTL time.Duration) *LoginHandler {
	return &LoginHandler{
		store:      store,
		sessions:   sessions,
		logger:     logger,
		audit:      auditLogger,
		privateKey: privateKey,
		kid:        kid,
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Login handles POST /login.
func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed < minResponseDuration {
			time.Sleep(minResponseDuration - elapsed)
		}
	}()

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	req.Identifier = strings.TrimSpace(req.Identifier)
	req.OrgSlug = strings.ToLower(strings.TrimSpace(req.OrgSlug))
	if req.Identifier == "" || req.Password == "" {
		apierror.BadRequest(w, "Identifier and password are required.")
		return
	}

	ctx := r.Context()

	// Resolve organization: use org_slug if provided, otherwise default.
	var orgID uuid.UUID
	var err error
	if req.OrgSlug != "" {
		orgID, err = h.store.GetOrganizationIDBySlug(ctx, req.OrgSlug)
	} else {
		orgID, err = h.store.GetDefaultOrganizationID(ctx)
	}
	if err != nil {
		// Don't reveal whether the org exists — return generic credentials error.
		apierror.Unauthorized(w, invalidCredentialsMsg)
		return
	}

	// Try email first, then username.
	user, err := h.store.GetUserByEmail(ctx, strings.ToLower(req.Identifier), orgID)
	if err != nil {
		h.logger.Error("failed to lookup user by email", "error", err)
		apierror.InternalError(w)
		return
	}
	if user == nil {
		user, err = h.store.GetUserByUsername(ctx, req.Identifier, orgID)
		if err != nil {
			h.logger.Error("failed to lookup user by username", "error", err)
			apierror.InternalError(w)
			return
		}
	}

	if user == nil {
		h.audit.Log(ctx, r, orgID, model.EventUserLoginFailed, nil, req.Identifier, "user", "", req.Identifier, map[string]any{"reason": "user_not_found"})
		apierror.Unauthorized(w, invalidCredentialsMsg)
		return
	}

	if !user.Enabled {
		h.audit.Log(ctx, r, orgID, model.EventUserLoginFailed, &user.ID, user.Username, "user", user.ID.String(), user.Username, map[string]any{"reason": "account_disabled"})
		apierror.Unauthorized(w, invalidCredentialsMsg)
		return
	}

	// Check account lockout
	if user.IsLocked() {
		h.audit.Log(ctx, r, orgID, model.EventUserLoginFailed, &user.ID, user.Username, "user", user.ID.String(), user.Username, map[string]any{"reason": "account_locked"})
		apierror.Unauthorized(w, "Account is temporarily locked. Please try again later.")
		return
	}

	// Fetch org settings early — needed for lockout policy and TTLs
	var settings *model.OrgSettings
	if s, sErr := h.store.GetOrgSettings(ctx, orgID); sErr != nil {
		h.logger.Warn("failed to fetch org settings, using defaults", "error", sErr)
	} else {
		settings = s
	}

	ok, err := auth.VerifyPassword(req.Password, string(user.PasswordHash))
	if err != nil {
		h.logger.Error("failed to verify password", "error", err)
		apierror.InternalError(w)
		return
	}
	if !ok {
		h.audit.Log(ctx, r, orgID, model.EventUserLoginFailed, &user.ID, user.Username, "user", user.ID.String(), user.Username, map[string]any{"reason": "invalid_password"})
		// Increment failed login counter with lockout policy
		maxAttempts, lockoutDur := defaultLockoutPolicy(settings)
		if maxAttempts > 0 {
			if lErr := h.store.IncrementFailedLogins(ctx, user.ID, maxAttempts, lockoutDur); lErr != nil {
				h.logger.Warn("failed to increment failed logins", "error", lErr)
			}
		}
		apierror.Unauthorized(w, invalidCredentialsMsg)
		return
	}

	// Reset failed login counter on success
	if user.FailedLoginAttempts > 0 {
		if lErr := h.store.ResetFailedLogins(ctx, user.ID); lErr != nil {
			h.logger.Warn("failed to reset failed logins", "error", lErr)
		}
	}

	// Check MFA requirement
	if user.MFAEnabled {
		device, mfaErr := h.store.GetVerifiedMFADevice(ctx, user.ID)
		if mfaErr != nil {
			h.logger.Error("failed to check MFA device", "error", mfaErr)
			apierror.InternalError(w)
			return
		}
		if device != nil {
			// Generate a short-lived MFA challenge token
			mfaToken, tErr := token.GenerateMFAToken(h.privateKey, h.kid, h.issuer, user.ID)
			if tErr != nil {
				h.logger.Error("failed to generate MFA token", "error", tErr)
				apierror.InternalError(w)
				return
			}
			w.Header().Set("Content-Type", apierror.ContentTypeJSON)
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(map[string]any{
				"mfa_required": true,
				"mfa_token":    mfaToken,
				"message":      "MFA verification required. Submit TOTP code to /mfa/totp/verify.",
			}); err != nil {
				h.logger.Error("failed to encode MFA challenge response", "error", err)
			}
			return
		}
	}

	// Resolve per-org session TTLs (fall back to server defaults).
	accessTTL := h.accessTTL
	refreshTTL := h.refreshTTL
	if settings != nil {
		if settings.AccessTokenTTL > 0 {
			accessTTL = settings.AccessTokenTTL
		}
		if settings.RefreshTokenTTL > 0 {
			refreshTTL = settings.RefreshTokenTTL
		}
	}

	// Fetch user roles (direct login API — include all roles)
	roles, rErr := h.store.GetEffectiveUserRoles(ctx, user.ID)
	if rErr != nil {
		h.logger.Warn("failed to fetch user roles", "error", rErr)
		roles = nil
	}

	accessToken, err := token.GenerateAccessToken(
		h.privateKey, h.kid, h.issuer, accessTTL,
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
		h.logger.Warn("failed to update last_login_at", "error", err, "user_id", user.ID)
	}

	h.audit.LogSimple(ctx, r, orgID, model.EventUserLogin, &user.ID, user.Username, "user", user.ID.String(), user.Username)

	resp := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenTypeBearer,
		ExpiresIn:    int(accessTTL.Seconds()),
		User:         user.ToResponse(),
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode login response", "error", err)
	}
}

// Refresh handles POST /token/refresh.
func (h *LoginHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	if req.RefreshToken == "" {
		apierror.Unauthorized(w, "Refresh token is required.")
		return
	}

	ctx := r.Context()

	sess, err := h.sessions.FindByRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		h.logger.Error("failed to find session", "error", err)
		apierror.InternalError(w)
		return
	}
	if sess == nil {
		apierror.Unauthorized(w, "Invalid or expired refresh token.")
		return
	}

	user, err := h.store.GetUserByID(ctx, sess.UserID)
	if err != nil {
		h.logger.Error("failed to get user for refresh", "error", err)
		apierror.InternalError(w)
		return
	}
	if user == nil || !user.Enabled {
		apierror.Unauthorized(w, "User account is disabled.")
		return
	}

	// Fetch user roles for refreshed token
	refreshRoles, rErr := h.store.GetEffectiveUserRoles(ctx, user.ID)
	if rErr != nil {
		h.logger.Warn("failed to fetch user roles for refresh", "error", rErr)
		refreshRoles = nil
	}

	accessToken, err := token.GenerateAccessToken(
		h.privateKey, h.kid, h.issuer, h.accessTTL,
		user.ID, user.OrgID,
		user.Username, user.Email, user.EmailVerified,
		user.GivenName, user.FamilyName,
		refreshRoles...,
	)
	if err != nil {
		h.logger.Error("failed to generate access token", "error", err)
		apierror.InternalError(w)
		return
	}

	resp := RefreshResponse{
		AccessToken: accessToken,
		TokenType:   tokenTypeBearer,
		ExpiresIn:   int(h.accessTTL.Seconds()),
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode refresh response", "error", err)
	}
}

// Logout handles POST /logout.
func (h *LoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	if req.RefreshToken == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx := r.Context()

	sess, err := h.sessions.FindByRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		h.logger.Error("failed to find session for logout", "error", err)
		apierror.InternalError(w)
		return
	}
	if sess == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.sessions.Delete(ctx, sess.ID); err != nil {
		h.logger.Error("failed to delete session", "error", err)
		apierror.InternalError(w)
		return
	}

	// Look up user for audit event
	if user, uErr := h.store.GetUserByID(ctx, sess.UserID); uErr == nil && user != nil {
		h.audit.LogSimple(ctx, r, user.OrgID, model.EventSessionRevoked, &user.ID, user.Username, "session", sess.ID.String(), "")
	}

	w.WriteHeader(http.StatusNoContent)
}
