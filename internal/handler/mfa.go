package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/mfa"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
)

const (
	msgNotAuthenticated   = "Not authenticated."
	msgDeviceNotFound     = "TOTP device not found."
	msgInvalidDeviceID    = "Invalid device ID."
	errMsgInvalidTOTPCode = "Invalid TOTP code."
	totpIssuer            = "Rampart"
)

// MFAStore defines the database operations required by the MFA handler.
type MFAStore interface {
	GetTOTPDevicesByUserID(ctx context.Context, userID uuid.UUID) ([]*model.TOTPDevice, error)
	CreateTOTPDevice(ctx context.Context, device *model.TOTPDevice) (*model.TOTPDevice, error)
	VerifyTOTPDevice(ctx context.Context, deviceID uuid.UUID) error
	DeleteTOTPDevice(ctx context.Context, deviceID uuid.UUID) error
	UpdateTOTPDeviceLastUsed(ctx context.Context, deviceID uuid.UUID) error
	CreateRecoveryCodes(ctx context.Context, userID uuid.UUID, codes []*model.RecoveryCode) error
	GetUnusedRecoveryCodes(ctx context.Context, userID uuid.UUID) ([]*model.RecoveryCode, error)
	UseRecoveryCode(ctx context.Context, codeID uuid.UUID) error
	SetUserMFAEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error
}

// MFAHandler handles MFA enrollment and verification endpoints.
type MFAHandler struct {
	store  MFAStore
	logger *slog.Logger
}

// NewMFAHandler creates a new MFAHandler.
func NewMFAHandler(store MFAStore, logger *slog.Logger) *MFAHandler {
	return &MFAHandler{store: store, logger: logger}
}

// enrollTOTPRequest is the request body for TOTP enrollment.
type enrollTOTPRequest struct {
	Name string `json:"name"`
}

// enrollTOTPResponse is the response body for TOTP enrollment.
type enrollTOTPResponse struct {
	Secret   string `json:"secret"`
	QRURI    string `json:"qr_uri"`
	DeviceID string `json:"device_id"`
}

// EnrollTOTP handles POST /api/v1/mfa/totp/enroll.
func (h *MFAHandler) EnrollTOTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthenticatedUser(r.Context())
	if user == nil {
		apierror.Unauthorized(w, msgNotAuthenticated)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req enrollTOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body — name is optional.
		req.Name = "Default"
	}
	if req.Name == "" {
		req.Name = "Default"
	}

	secret, err := mfa.GenerateSecret()
	if err != nil {
		h.logger.Error("failed to generate TOTP secret", "error", err)
		apierror.InternalError(w)
		return
	}

	device := &model.TOTPDevice{
		ID:     uuid.New(),
		UserID: user.UserID,
		Name:   req.Name,
		Secret: secret,
	}

	created, err := h.store.CreateTOTPDevice(r.Context(), device)
	if err != nil {
		h.logger.Error("failed to create TOTP device", "error", err)
		apierror.InternalError(w)
		return
	}

	qrURI := mfa.GenerateQRCodeURI(secret, totpIssuer, user.Email)

	resp := enrollTOTPResponse{
		Secret:   secret,
		QRURI:    qrURI,
		DeviceID: created.ID.String(),
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode enroll response", "error", err)
	}
}

// verifyTOTPRequest is the request body for TOTP verification.
type verifyTOTPRequest struct {
	DeviceID string `json:"device_id"`
	Code     string `json:"code"`
}

// verifyTOTPResponse is the response body for TOTP verification.
type verifyTOTPResponse struct {
	RecoveryCodes []string `json:"recovery_codes"`
}

// VerifyTOTP handles POST /api/v1/mfa/totp/verify.
func (h *MFAHandler) VerifyTOTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthenticatedUser(r.Context())
	if user == nil {
		apierror.Unauthorized(w, msgNotAuthenticated)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req verifyTOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	if req.DeviceID == "" || req.Code == "" {
		apierror.BadRequest(w, "device_id and code are required.")
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		apierror.BadRequest(w, msgInvalidDeviceID)
		return
	}

	// Look up the device to get the secret.
	devices, err := h.store.GetTOTPDevicesByUserID(r.Context(), user.UserID)
	if err != nil {
		h.logger.Error("failed to get TOTP devices", "error", err)
		apierror.InternalError(w)
		return
	}

	var target *model.TOTPDevice
	for _, d := range devices {
		if d.ID == deviceID {
			target = d
			break
		}
	}

	if target == nil {
		apierror.NotFound(w)
		return
	}

	if target.UserID != user.UserID {
		apierror.Forbidden(w, "Cannot verify another user's device.")
		return
	}

	if !mfa.ValidateCode(target.Secret, req.Code) {
		apierror.BadRequest(w, errMsgInvalidTOTPCode)
		return
	}

	if err := h.store.VerifyTOTPDevice(r.Context(), deviceID); err != nil {
		h.logger.Error("failed to verify TOTP device", "error", err)
		apierror.InternalError(w)
		return
	}

	if err := h.store.SetUserMFAEnabled(r.Context(), user.UserID, true); err != nil {
		h.logger.Error("failed to enable MFA on user", "error", err)
		apierror.InternalError(w)
		return
	}

	codes, err := h.generateAndStoreRecoveryCodes(r.Context(), user.UserID)
	if err != nil {
		h.logger.Error("failed to generate recovery codes", "error", err)
		apierror.InternalError(w)
		return
	}

	resp := verifyTOTPResponse{RecoveryCodes: codes}
	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode verify response", "error", err)
	}
}

// validateTOTPRequest is the request body for TOTP validation during login.
type validateTOTPRequest struct {
	Code string `json:"code"`
}

// ValidateTOTP handles POST /api/v1/mfa/totp/validate.
func (h *MFAHandler) ValidateTOTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthenticatedUser(r.Context())
	if user == nil {
		apierror.Unauthorized(w, msgNotAuthenticated)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req validateTOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	if req.Code == "" {
		apierror.BadRequest(w, "code is required.")
		return
	}

	devices, err := h.store.GetTOTPDevicesByUserID(r.Context(), user.UserID)
	if err != nil {
		h.logger.Error("failed to get TOTP devices", "error", err)
		apierror.InternalError(w)
		return
	}

	// Find a verified device whose code matches.
	var matched *model.TOTPDevice
	for _, d := range devices {
		if d.Verified && mfa.ValidateCode(d.Secret, req.Code) {
			matched = d
			break
		}
	}

	if matched == nil {
		apierror.Unauthorized(w, errMsgInvalidTOTPCode)
		return
	}

	if err := h.store.UpdateTOTPDeviceLastUsed(r.Context(), matched.ID); err != nil {
		h.logger.Warn("failed to update TOTP device last_used", "error", err)
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]bool{"valid": true}); err != nil {
		h.logger.Error("failed to encode validate response", "error", err)
	}
}

// DeleteTOTPDevice handles DELETE /api/v1/mfa/totp/{deviceID}.
func (h *MFAHandler) DeleteTOTPDevice(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthenticatedUser(r.Context())
	if user == nil {
		apierror.Unauthorized(w, msgNotAuthenticated)
		return
	}

	deviceIDStr := chi.URLParam(r, "deviceID")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		apierror.BadRequest(w, msgInvalidDeviceID)
		return
	}

	devices, err := h.store.GetTOTPDevicesByUserID(r.Context(), user.UserID)
	if err != nil {
		h.logger.Error("failed to get TOTP devices", "error", err)
		apierror.InternalError(w)
		return
	}

	var found bool
	var verifiedCount int
	for _, d := range devices {
		if d.Verified {
			verifiedCount++
		}
		if d.ID == deviceID {
			found = true
			if d.UserID != user.UserID {
				apierror.Forbidden(w, "Cannot delete another user's device.")
				return
			}
		}
	}

	if !found {
		apierror.Write(w, http.StatusNotFound, "not_found", msgDeviceNotFound)
		return
	}

	if err := h.store.DeleteTOTPDevice(r.Context(), deviceID); err != nil {
		h.logger.Error("failed to delete TOTP device", "error", err)
		apierror.InternalError(w)
		return
	}

	// If this was the last verified device, disable MFA.
	if verifiedCount <= 1 {
		if err := h.store.SetUserMFAEnabled(r.Context(), user.UserID, false); err != nil {
			h.logger.Error("failed to disable MFA on user", "error", err)
			apierror.InternalError(w)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// listDevicesResponse is the response for GET /api/v1/mfa/devices.
type listDevicesResponse struct {
	Devices []*model.TOTPDeviceResponse `json:"devices"`
}

// ListDevices handles GET /api/v1/mfa/devices.
func (h *MFAHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthenticatedUser(r.Context())
	if user == nil {
		apierror.Unauthorized(w, msgNotAuthenticated)
		return
	}

	devices, err := h.store.GetTOTPDevicesByUserID(r.Context(), user.UserID)
	if err != nil {
		h.logger.Error("failed to get TOTP devices", "error", err)
		apierror.InternalError(w)
		return
	}

	var verified []*model.TOTPDeviceResponse
	for _, d := range devices {
		if d.Verified {
			verified = append(verified, d.ToResponse())
		}
	}

	if verified == nil {
		verified = []*model.TOTPDeviceResponse{}
	}

	resp := listDevicesResponse{Devices: verified}
	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode devices response", "error", err)
	}
}

// recoveryCodesStatusResponse is the response for GET /api/v1/mfa/recovery-codes.
type recoveryCodesStatusResponse struct {
	Remaining int `json:"remaining"`
	Total     int `json:"total"`
}

// RecoveryCodesStatus handles GET /api/v1/mfa/recovery-codes.
func (h *MFAHandler) RecoveryCodesStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthenticatedUser(r.Context())
	if user == nil {
		apierror.Unauthorized(w, msgNotAuthenticated)
		return
	}

	codes, err := h.store.GetUnusedRecoveryCodes(r.Context(), user.UserID)
	if err != nil {
		h.logger.Error("failed to get recovery codes", "error", err)
		apierror.InternalError(w)
		return
	}

	resp := recoveryCodesStatusResponse{
		Remaining: len(codes),
		Total:     mfa.RecoveryCodeCount,
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode recovery codes status", "error", err)
	}
}

// regenerateRecoveryCodesResponse is the response for POST /api/v1/mfa/recovery-codes/regenerate.
type regenerateRecoveryCodesResponse struct {
	RecoveryCodes []string `json:"recovery_codes"`
}

// RegenerateRecoveryCodes handles POST /api/v1/mfa/recovery-codes/regenerate.
func (h *MFAHandler) RegenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthenticatedUser(r.Context())
	if user == nil {
		apierror.Unauthorized(w, msgNotAuthenticated)
		return
	}

	codes, err := h.generateAndStoreRecoveryCodes(r.Context(), user.UserID)
	if err != nil {
		h.logger.Error("failed to regenerate recovery codes", "error", err)
		apierror.InternalError(w)
		return
	}

	resp := regenerateRecoveryCodesResponse{RecoveryCodes: codes}
	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode regenerate response", "error", err)
	}
}

// generateAndStoreRecoveryCodes creates new recovery codes, hashes them, stores them, and returns the plaintext codes.
func (h *MFAHandler) generateAndStoreRecoveryCodes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	plaintextCodes, err := mfa.GenerateRecoveryCodes(mfa.RecoveryCodeCount)
	if err != nil {
		return nil, err
	}

	records := make([]*model.RecoveryCode, len(plaintextCodes))
	for i, code := range plaintextCodes {
		hash, hErr := mfa.HashRecoveryCode(code)
		if hErr != nil {
			return nil, hErr
		}
		records[i] = &model.RecoveryCode{
			ID:       uuid.New(),
			UserID:   userID,
			CodeHash: hash,
		}
	}

	if err := h.store.CreateRecoveryCodes(ctx, userID, records); err != nil {
		return nil, err
	}

	return plaintextCodes, nil
}
