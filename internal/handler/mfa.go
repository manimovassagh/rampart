package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/audit"
	"github.com/manimovassagh/rampart/internal/mfa"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/store"
)

// MFAHandlerStore defines the database operations required by MFAHandler.
type MFAHandlerStore interface {
	store.UserReader
	store.MFADeviceStore
}

// MFAHandler handles MFA enrollment and management endpoints.
type MFAHandler struct {
	store  MFAHandlerStore
	logger *slog.Logger
	audit  *audit.Logger
	issuer string
}

// NewMFAHandler creates a new MFA handler.
func NewMFAHandler(s MFAHandlerStore, logger *slog.Logger, auditLogger *audit.Logger, issuer string) *MFAHandler {
	return &MFAHandler{store: s, logger: logger, audit: auditLogger, issuer: issuer}
}

// EnrollTOTP handles POST /mfa/totp/enroll.
// Initiates TOTP enrollment by generating a secret and returning the provisioning URI.
// Requires authentication.
func (h *MFAHandler) EnrollTOTP(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetAuthenticatedUser(r.Context())
	if claims == nil {
		apierror.Write(w, http.StatusUnauthorized, "unauthorized", "Authentication required.")
		return
	}

	ctx := r.Context()
	userID := claims.UserID

	// Check if user already has MFA enabled
	existing, err := h.store.GetVerifiedMFADevice(ctx, userID)
	if err != nil {
		h.logger.Error("failed to check existing MFA", "error", err)
		apierror.Write(w, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}
	if existing != nil {
		apierror.Write(w, http.StatusConflict, "mfa_already_enabled", "MFA is already enabled. Disable it first to re-enroll.")
		return
	}

	// Clean up any previous pending enrollment
	_ = h.store.DeleteUnverifiedMFADevices(ctx, userID)

	// Generate secret
	secret, err := mfa.GenerateSecret()
	if err != nil {
		h.logger.Error("failed to generate TOTP secret", "error", err)
		apierror.Write(w, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}

	// Get user email for provisioning URI
	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		h.logger.Error("failed to get user for MFA enrollment", "error", err)
		apierror.Write(w, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}

	// Store device (unverified)
	device, err := h.store.CreateMFADevice(ctx, userID, "totp", "Authenticator", secret)
	if err != nil {
		h.logger.Error("failed to create MFA device", "error", err)
		apierror.Write(w, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}

	uri := mfa.ProvisioningURI(secret, user.Email, h.issuer)

	h.audit.LogSimple(ctx, r, user.OrgID, model.EventMFAEnrolled, &user.ID, user.Email, "user", user.ID.String(), user.Email)

	resp := model.TOTPEnrollResponse{
		Secret:          secret,
		ProvisioningURI: uri,
		DeviceID:        device.ID,
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode TOTP enroll response", "error", err)
	}
}

// VerifyTOTPSetup handles POST /mfa/totp/verify-setup.
// Verifies the TOTP code to complete enrollment and returns backup codes.
func (h *MFAHandler) VerifyTOTPSetup(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	claims := middleware.GetAuthenticatedUser(r.Context())
	if claims == nil {
		apierror.Write(w, http.StatusUnauthorized, "unauthorized", "Authentication required.")
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		apierror.Write(w, http.StatusBadRequest, "invalid_request", "A TOTP code is required.")
		return
	}

	ctx := r.Context()
	userID := claims.UserID

	// Get pending device
	device, err := h.store.GetPendingMFADevice(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get pending MFA device", "error", err)
		apierror.Write(w, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}
	if device == nil {
		apierror.Write(w, http.StatusBadRequest, "no_pending_enrollment", "No pending MFA enrollment found. Call /mfa/totp/enroll first.")
		return
	}

	// Validate TOTP code
	if !mfa.ValidateCode(device.Secret, req.Code) {
		apierror.Write(w, http.StatusBadRequest, "invalid_code", "Invalid TOTP code. Please try again.")
		return
	}

	// Mark device as verified and enable MFA on user
	if err := h.store.VerifyMFADevice(ctx, device.ID, userID); err != nil {
		h.logger.Error("failed to verify MFA device", "error", err)
		apierror.Write(w, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}

	// Generate and store backup codes
	backupCodes, err := mfa.GenerateBackupCodes()
	if err != nil {
		h.logger.Error("failed to generate backup codes", "error", err)
		apierror.Write(w, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}

	hashes := make([][]byte, len(backupCodes))
	for i, code := range backupCodes {
		hashes[i] = mfa.HashBackupCode(code)
	}
	if err := h.store.StoreBackupCodes(ctx, userID, hashes); err != nil {
		h.logger.Error("failed to store backup codes", "error", err)
	}

	h.logger.Info("MFA enabled", "user_id", userID)

	// Fetch user to get org ID for audit logging
	verifyUser, _ := h.store.GetUserByID(ctx, userID)
	if verifyUser != nil {
		h.audit.LogSimple(ctx, r, verifyUser.OrgID, model.EventMFAVerified, &userID, verifyUser.Email, "user", userID.String(), verifyUser.Email)
	}

	resp := model.TOTPVerifySetupResponse{
		Message:     "MFA has been enabled successfully.",
		BackupCodes: backupCodes,
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode TOTP verify-setup response", "error", err)
	}
}

// DisableTOTP handles POST /mfa/totp/disable.
// Requires a valid TOTP code or backup code to disable MFA.
func (h *MFAHandler) DisableTOTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	claims := middleware.GetAuthenticatedUser(r.Context())
	if claims == nil {
		apierror.Write(w, http.StatusUnauthorized, "unauthorized", "Authentication required.")
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		apierror.Write(w, http.StatusBadRequest, "invalid_request", "A TOTP code is required to disable MFA.")
		return
	}

	ctx := r.Context()
	userID := claims.UserID

	device, err := h.store.GetVerifiedMFADevice(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get MFA device", "error", err)
		apierror.Write(w, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}
	if device == nil {
		apierror.Write(w, http.StatusBadRequest, "mfa_not_enabled", "MFA is not enabled.")
		return
	}

	if !mfa.ValidateCode(device.Secret, req.Code) {
		apierror.Write(w, http.StatusBadRequest, "invalid_code", "Invalid TOTP code.")
		return
	}

	if err := h.store.DisableMFA(ctx, userID); err != nil {
		h.logger.Error("failed to disable MFA", "error", err)
		apierror.Write(w, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}

	h.logger.Info("MFA disabled", "user_id", userID)

	// Fetch user to get org ID for audit logging
	disableUser, _ := h.store.GetUserByID(ctx, userID)
	if disableUser != nil {
		h.audit.LogSimple(ctx, r, disableUser.OrgID, model.EventMFADisabled, &userID, disableUser.Email, "user", userID.String(), disableUser.Email)
	}

	resp := model.TOTPDisableResponse{
		Message: "MFA has been disabled.",
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode TOTP disable response", "error", err)
	}
}
