package handler

import (
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/audit"
	"github.com/manimovassagh/rampart/internal/metrics"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
	"github.com/manimovassagh/rampart/internal/store"
	"github.com/manimovassagh/rampart/internal/token"
)

// WebAuthnStore defines the database operations required by WebAuthnHandler.
type WebAuthnStore interface {
	store.UserReader
	store.UserWriter
	store.OrgSettingsReadWriter
	store.GroupReader
	store.WebAuthnCredentialStore
	store.WebAuthnSessionStore
	store.MFADeviceStore
}

// WebAuthnHandler handles WebAuthn/Passkey registration and authentication.
type WebAuthnHandler struct {
	store      WebAuthnStore
	sessions   session.Store
	logger     *slog.Logger
	audit      *audit.Logger
	wa         *webauthn.WebAuthn
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewWebAuthnHandler creates a new WebAuthn handler.
func NewWebAuthnHandler(
	s WebAuthnStore,
	sessions session.Store,
	logger *slog.Logger,
	auditLogger *audit.Logger,
	wa *webauthn.WebAuthn,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	kid, issuer string,
	accessTTL, refreshTTL time.Duration,
) *WebAuthnHandler {
	return &WebAuthnHandler{
		store:      s,
		sessions:   sessions,
		logger:     logger,
		audit:      auditLogger,
		wa:         wa,
		privateKey: privateKey,
		publicKey:  publicKey,
		kid:        kid,
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// BeginRegistration handles POST /mfa/webauthn/register/begin.
// Starts WebAuthn credential registration (requires authentication).
func (h *WebAuthnHandler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetAuthenticatedUser(r.Context())
	if claims == nil {
		apierror.Unauthorized(w, msgAuthRequired)
		return
	}

	ctx := r.Context()
	user, err := h.store.GetUserByID(ctx, claims.UserID)
	if err != nil || user == nil {
		apierror.InternalError(w)
		return
	}

	// Get existing credentials to exclude them
	existingCreds, err := h.store.GetWebAuthnCredentialsByUserID(ctx, user.ID)
	if err != nil {
		h.logger.Error("failed to get webauthn credentials", "error", err)
		apierror.InternalError(w)
		return
	}

	libCreds := make([]webauthn.Credential, len(existingCreds))
	for i, c := range existingCreds {
		libCreds[i] = c.ToLibCredential()
	}

	waUser := &model.WebAuthnUser{User: user, Credentials: libCreds}

	creation, sessionData, err := h.wa.BeginRegistration(waUser,
		webauthn.WithExclusions(webauthn.Credentials(libCreds).CredentialDescriptors()),
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementPreferred),
	)
	if err != nil {
		h.logger.Error("failed to begin webauthn registration", "error", err)
		apierror.Write(w, http.StatusInternalServerError, "webauthn_error", "Failed to start passkey registration.")
		return
	}

	// Store session data for the finish step
	sdBytes, err := json.Marshal(sessionData)
	if err != nil {
		apierror.InternalError(w)
		return
	}
	if err := h.store.StoreWebAuthnSessionData(ctx, user.ID, sdBytes, "registration", time.Now().Add(5*time.Minute)); err != nil {
		h.logger.Error("failed to store webauthn session", "error", err)
		apierror.InternalError(w)
		return
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(creation); err != nil {
		h.logger.Error("failed to encode webauthn creation options", "error", err)
	}
}

// FinishRegistration handles POST /mfa/webauthn/register/complete.
// Completes WebAuthn credential registration (requires authentication).
func (h *WebAuthnHandler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetAuthenticatedUser(r.Context())
	if claims == nil {
		apierror.Unauthorized(w, msgAuthRequired)
		return
	}

	ctx := r.Context()
	user, err := h.store.GetUserByID(ctx, claims.UserID)
	if err != nil || user == nil {
		apierror.InternalError(w)
		return
	}

	// Retrieve session data
	sdBytes, err := h.store.GetWebAuthnSessionData(ctx, user.ID, "registration")
	if err != nil {
		h.logger.Error("failed to get webauthn session", "error", err)
		apierror.Write(w, http.StatusBadRequest, "session_expired", "Registration session expired. Please start over.")
		return
	}

	var sessionData webauthn.SessionData
	if err := json.Unmarshal(sdBytes, &sessionData); err != nil {
		apierror.InternalError(w)
		return
	}

	existingCreds, _ := h.store.GetWebAuthnCredentialsByUserID(ctx, user.ID)
	libCreds := make([]webauthn.Credential, len(existingCreds))
	for i, c := range existingCreds {
		libCreds[i] = c.ToLibCredential()
	}
	waUser := &model.WebAuthnUser{User: user, Credentials: libCreds}

	credential, err := h.wa.FinishRegistration(waUser, sessionData, r)
	if err != nil {
		h.logger.Error("failed to finish webauthn registration", "error", err)
		apierror.Write(w, http.StatusBadRequest, "verification_failed", "Failed to verify passkey. Please try again.")
		return
	}

	// Read optional name from query parameter
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "Passkey"
	}

	// Convert transports to strings
	transports := make([]string, len(credential.Transport))
	for i, t := range credential.Transport {
		transports[i] = string(t)
	}

	// Store credential
	dbCred := &model.WebAuthnCredential{
		UserID:          user.ID,
		CredentialID:    credential.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		Transport:       transports,
		FlagsRaw:        uint8(credential.Flags.ProtocolValue()),
		AAGUID:          credential.Authenticator.AAGUID,
		SignCount:       credential.Authenticator.SignCount,
		Name:            name,
	}

	if err := h.store.CreateWebAuthnCredential(ctx, dbCred); err != nil {
		h.logger.Error("failed to store webauthn credential", "error", err)
		apierror.InternalError(w)
		return
	}

	// Enable MFA on the user if not already enabled
	if !user.MFAEnabled {
		// Create a dummy verified MFA device to set the mfa_enabled flag
		// This ensures the login flow knows to challenge for MFA
		existing, _ := h.store.GetVerifiedMFADevice(ctx, user.ID)
		if existing == nil {
			// Use VerifyMFADevice path: create + verify to set mfa_enabled
			device, dErr := h.store.CreateMFADevice(ctx, user.ID, "webauthn", "Passkey", "")
			if dErr == nil {
				_ = h.store.VerifyMFADevice(ctx, device.ID, user.ID)
			}
		}
	}

	h.logger.Info("webauthn credential registered", "user_id", user.ID, "name", name)

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Passkey registered successfully.",
		"name":    name,
	}); err != nil {
		h.logger.Error("failed to encode webauthn register response", "error", err)
	}
}

// BeginLogin handles POST /mfa/webauthn/login/begin.
// Starts WebAuthn authentication (uses MFA token from login flow).
func (h *WebAuthnHandler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req struct {
		MFAToken string `json:"mfa_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MFAToken == "" {
		apierror.BadRequest(w, "mfa_token is required.")
		return
	}

	userID, err := token.VerifyMFAToken(h.publicKey, req.MFAToken)
	if err != nil {
		apierror.Unauthorized(w, "Invalid or expired MFA token.")
		return
	}

	ctx := r.Context()
	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil || user == nil || !user.Enabled {
		apierror.Unauthorized(w, "Invalid or expired MFA token.")
		return
	}

	creds, err := h.store.GetWebAuthnCredentialsByUserID(ctx, userID)
	if err != nil || len(creds) == 0 {
		apierror.Write(w, http.StatusBadRequest, "no_passkeys", "No passkeys registered for this account.")
		return
	}

	libCreds := make([]webauthn.Credential, len(creds))
	for i, c := range creds {
		libCreds[i] = c.ToLibCredential()
	}
	waUser := &model.WebAuthnUser{User: user, Credentials: libCreds}

	assertion, sessionData, err := h.wa.BeginLogin(waUser)
	if err != nil {
		h.logger.Error("failed to begin webauthn login", "error", err)
		apierror.InternalError(w)
		return
	}

	sdBytes, err := json.Marshal(sessionData)
	if err != nil {
		apierror.InternalError(w)
		return
	}
	if err := h.store.StoreWebAuthnSessionData(ctx, userID, sdBytes, "login", time.Now().Add(5*time.Minute)); err != nil {
		h.logger.Error("failed to store webauthn session", "error", err)
		apierror.InternalError(w)
		return
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(assertion); err != nil {
		h.logger.Error("failed to encode webauthn assertion options", "error", err)
	}
}

// FinishLogin handles POST /mfa/webauthn/login/complete.
// Completes WebAuthn authentication and issues tokens.
func (h *WebAuthnHandler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	// Extract mfa_token from query since the body is the authenticator response
	mfaToken := r.URL.Query().Get("mfa_token")
	if mfaToken == "" {
		apierror.BadRequest(w, "mfa_token query parameter is required.")
		return
	}

	userID, err := token.VerifyMFAToken(h.publicKey, mfaToken)
	if err != nil {
		apierror.Unauthorized(w, "Invalid or expired MFA token.")
		return
	}

	ctx := r.Context()
	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil || user == nil || !user.Enabled {
		apierror.Unauthorized(w, "Invalid or expired MFA token.")
		return
	}

	sdBytes, err := h.store.GetWebAuthnSessionData(ctx, userID, "login")
	if err != nil {
		h.logger.Error("failed to get webauthn login session", "error", err)
		apierror.Write(w, http.StatusBadRequest, "session_expired", "Login session expired. Please start over.")
		return
	}

	var sessionData webauthn.SessionData
	if err := json.Unmarshal(sdBytes, &sessionData); err != nil {
		apierror.InternalError(w)
		return
	}

	creds, _ := h.store.GetWebAuthnCredentialsByUserID(ctx, userID)
	libCreds := make([]webauthn.Credential, len(creds))
	for i, c := range creds {
		libCreds[i] = c.ToLibCredential()
	}
	waUser := &model.WebAuthnUser{User: user, Credentials: libCreds}

	credential, err := h.wa.FinishLogin(waUser, sessionData, r)
	if err != nil {
		h.logger.Error("failed to finish webauthn login", "error", err)
		h.audit.Log(ctx, r, user.OrgID, model.EventUserLoginFailed, &user.ID, user.Username,
			"user", user.ID.String(), user.Username, map[string]any{"reason": "webauthn_failed"})
		metrics.AuthTotal.WithLabelValues("failure").Inc()
		apierror.Unauthorized(w, "Passkey verification failed.")
		return
	}

	// Update sign count
	if err := h.store.UpdateWebAuthnSignCount(ctx, credential.ID, credential.Authenticator.SignCount); err != nil {
		h.logger.Warn("failed to update webauthn sign count", "error", err)
	}

	// Issue tokens (same flow as TOTP MFA verify)
	if user.FailedLoginAttempts > 0 {
		if lErr := h.store.ResetFailedLogins(ctx, user.ID); lErr != nil {
			h.logger.Warn("failed to reset failed logins after WebAuthn", "error", lErr)
		}
	}

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
	if _, err := h.sessions.Create(ctx, user.ID, "", refreshToken, expiresAt); err != nil {
		h.logger.Error("failed to create session", "error", err)
		apierror.InternalError(w)
		return
	}

	if err := h.store.UpdateLastLoginAt(ctx, user.ID); err != nil {
		h.logger.Warn("failed to update last_login_at", "error", err, "user_id", user.ID)
	}

	h.audit.LogSimple(ctx, r, user.OrgID, model.EventUserLogin, &user.ID, user.Username, "user", user.ID.String(), user.Username)
	metrics.AuthTotal.WithLabelValues("success").Inc()
	metrics.TokensIssued.WithLabelValues("access").Inc()
	metrics.TokensIssued.WithLabelValues("refresh").Inc()
	metrics.ActiveSessions.Inc()

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
		h.logger.Error("failed to encode webauthn login response", "error", err)
	}
}

// ListCredentials handles GET /mfa/webauthn/credentials.
// Returns the user's registered passkeys (requires authentication).
func (h *WebAuthnHandler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetAuthenticatedUser(r.Context())
	if claims == nil {
		apierror.Unauthorized(w, msgAuthRequired)
		return
	}

	creds, err := h.store.GetWebAuthnCredentialsByUserID(r.Context(), claims.UserID)
	if err != nil {
		h.logger.Error("failed to list webauthn credentials", "error", err)
		apierror.InternalError(w)
		return
	}

	type credResponse struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
	}

	resp := make([]credResponse, len(creds))
	for i, c := range creds {
		resp[i] = credResponse{
			ID:        c.ID.String(),
			Name:      c.Name,
			CreatedAt: c.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode webauthn credentials response", "error", err)
	}
}

// DeleteCredential handles DELETE /mfa/webauthn/credentials/{id}.
// Removes a passkey (requires authentication).
func (h *WebAuthnHandler) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetAuthenticatedUser(r.Context())
	if claims == nil {
		apierror.Unauthorized(w, msgAuthRequired)
		return
	}

	credID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierror.BadRequest(w, "Invalid credential ID.")
		return
	}

	if err := h.store.DeleteWebAuthnCredential(r.Context(), credID, claims.UserID); err != nil {
		h.logger.Error("failed to delete webauthn credential", "error", err)
		apierror.InternalError(w)
		return
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Passkey deleted.",
	}); err != nil {
		h.logger.Error("failed to encode delete response", "error", err)
	}
}
