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
	"github.com/manimovassagh/rampart/internal/audit"
	"github.com/manimovassagh/rampart/internal/database"
	"github.com/manimovassagh/rampart/internal/mfa"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
	"github.com/manimovassagh/rampart/internal/token"
)

// MFAVerifyStore defines the database operations required by MFAVerifyHandler.
type MFAVerifyStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetVerifiedMFADevice(ctx context.Context, userID uuid.UUID) (*database.MFADevice, error)
	ConsumeBackupCode(ctx context.Context, userID uuid.UUID, codeHash []byte) (bool, error)
	UpdateLastLoginAt(ctx context.Context, userID uuid.UUID) error
	GetOrgSettings(ctx context.Context, orgID uuid.UUID) (*model.OrgSettings, error)
	GetEffectiveUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
	ResetFailedLogins(ctx context.Context, userID uuid.UUID) error
}

// MFAVerifyHandler handles the MFA verification step during login.
type MFAVerifyHandler struct {
	store      MFAVerifyStore
	sessions   session.Store
	logger     *slog.Logger
	audit      *audit.Logger
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewMFAVerifyHandler creates a handler for MFA verification during login.
func NewMFAVerifyHandler(store MFAVerifyStore, sessions session.Store, logger *slog.Logger, auditLogger *audit.Logger, privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, kid, issuer string, accessTTL, refreshTTL time.Duration) *MFAVerifyHandler {
	return &MFAVerifyHandler{
		store:      store,
		sessions:   sessions,
		logger:     logger,
		audit:      auditLogger,
		privateKey: privateKey,
		publicKey:  publicKey,
		kid:        kid,
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// VerifyTOTP handles POST /mfa/totp/verify.
// Validates a TOTP code (or backup code) against the MFA challenge token
// and issues real access/refresh tokens on success.
func (h *MFAVerifyHandler) VerifyTOTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed < minResponseDuration {
			time.Sleep(minResponseDuration - elapsed)
		}
	}()

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req struct {
		MFAToken string `json:"mfa_token"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	if req.MFAToken == "" || req.Code == "" {
		apierror.BadRequest(w, "mfa_token and code are required.")
		return
	}

	// Verify the MFA challenge token
	userID, err := token.VerifyMFAToken(h.publicKey, req.MFAToken)
	if err != nil {
		apierror.Unauthorized(w, "Invalid or expired MFA token.")
		return
	}

	ctx := r.Context()

	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get user for MFA verify", "error", err)
		apierror.InternalError(w)
		return
	}
	if user == nil || !user.Enabled {
		apierror.Unauthorized(w, "Invalid or expired MFA token.")
		return
	}

	// Get verified MFA device
	device, err := h.store.GetVerifiedMFADevice(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get MFA device", "error", err)
		apierror.InternalError(w)
		return
	}
	if device == nil {
		apierror.BadRequest(w, "MFA is not configured for this account.")
		return
	}

	// Try TOTP code first, then backup code
	valid := mfa.ValidateCode(device.Secret, req.Code)
	if !valid {
		// Try as backup code
		codeHash := mfa.HashBackupCode(req.Code)
		consumed, bErr := h.store.ConsumeBackupCode(ctx, userID, codeHash)
		if bErr != nil {
			h.logger.Error("failed to check backup code", "error", bErr)
			apierror.InternalError(w)
			return
		}
		if consumed {
			valid = true
			h.logger.Info("backup code used for MFA", "user_id", userID)
		}
	}

	if !valid {
		h.audit.Log(ctx, r, user.OrgID, model.EventUserLoginFailed, &user.ID, user.Username, "user", user.ID.String(), user.Username, map[string]any{"reason": "invalid_mfa_code"})
		apierror.Unauthorized(w, "Invalid MFA code.")
		return
	}

	// MFA passed — issue real tokens
	if user.FailedLoginAttempts > 0 {
		if lErr := h.store.ResetFailedLogins(ctx, user.ID); lErr != nil {
			h.logger.Warn("failed to reset failed logins after MFA", "error", lErr)
		}
	}

	// Resolve per-org session TTLs
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

	h.audit.LogSimple(ctx, r, user.OrgID, model.EventUserLogin, &user.ID, user.Username, "user", user.ID.String(), user.Username)

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
		h.logger.Error("failed to encode MFA verify response", "error", err)
	}
}
