package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/model"
)

const (
	resetTokenLength = 32 // 32 bytes = 64 hex chars
	resetTokenTTL    = 1 * time.Hour
)

// EmailSender is the interface for sending emails.
type EmailSender interface {
	Send(to, subject, body string) error
	Enabled() bool
}

// PasswordResetStore defines the database operations for password reset.
type PasswordResetStore interface {
	FindUserByEmail(ctx context.Context, email string) (*model.User, error)
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error
	ConsumePasswordResetToken(ctx context.Context, token string) (uuid.UUID, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash []byte) error
}

// PasswordResetHandler handles forgot-password and reset-password flows.
type PasswordResetHandler struct {
	store  PasswordResetStore
	email  EmailSender
	logger *slog.Logger
	issuer string // used to build reset URL
}

// NewPasswordResetHandler creates a new password reset handler.
func NewPasswordResetHandler(store PasswordResetStore, email EmailSender, logger *slog.Logger, issuer string) *PasswordResetHandler {
	return &PasswordResetHandler{
		store:  store,
		email:  email,
		logger: logger,
		issuer: issuer,
	}
}

// ForgotPasswordRequest is the expected JSON body for POST /forgot-password.
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest is the expected JSON body for POST /reset-password.
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ForgotPassword handles POST /forgot-password.
// Always returns 200 to prevent email enumeration.
func (h *PasswordResetHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Write(w, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}

	if req.Email == "" {
		apierror.Write(w, http.StatusBadRequest, "invalid_request", "Email is required.")
		return
	}

	// Always return success to prevent email enumeration
	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "If an account with that email exists, a password reset link has been sent.",
	}); err != nil {
		h.logger.Error("failed to encode forgot-password response", "error", err)
	}

	// Process async-style (but synchronously for simplicity)
	go h.processResetRequest(req.Email)
}

func (h *PasswordResetHandler) processResetRequest(email string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := h.store.FindUserByEmail(ctx, email)
	if err != nil {
		h.logger.Error("failed to find user for password reset", "error", err)
		return
	}
	if user == nil || !user.Enabled {
		return // silently ignore
	}

	// Generate token
	tokenBytes := make([]byte, resetTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		h.logger.Error("failed to generate reset token", "error", err)
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Store token hash
	expiresAt := time.Now().Add(resetTokenTTL)
	if err := h.store.CreatePasswordResetToken(ctx, user.ID, token, expiresAt); err != nil {
		h.logger.Error("failed to store password reset token", "error", err)
		return
	}

	// Send email
	if !h.email.Enabled() {
		h.logger.Warn("SMTP not configured, logging reset token",
			"user_id", user.ID,
			"token", token,
			"expires_at", expiresAt,
		)
		return
	}

	resetURL := h.issuer + "/reset-password?token=" + token
	body := buildResetEmail(user.GivenName, resetURL)

	if err := h.email.Send(email, "Reset your password — Rampart", body); err != nil {
		h.logger.Error("failed to send password reset email", "error", err, "user_id", user.ID)
	}
}

// ResetPassword handles POST /reset-password.
func (h *PasswordResetHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Write(w, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		apierror.Write(w, http.StatusBadRequest, "invalid_request", "Token and new_password are required.")
		return
	}

	if len(req.NewPassword) < 8 {
		apierror.Write(w, http.StatusBadRequest, "invalid_request", "Password must be at least 8 characters.")
		return
	}

	ctx := r.Context()

	userID, err := h.store.ConsumePasswordResetToken(ctx, req.Token)
	if err != nil {
		apierror.Write(w, http.StatusBadRequest, "invalid_token", "Invalid, expired, or already-used reset token.")
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		h.logger.Error("failed to hash new password", "error", err)
		apierror.Write(w, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}

	if err := h.store.UpdatePassword(ctx, userID, []byte(hash)); err != nil {
		h.logger.Error("failed to update password", "error", err)
		apierror.Write(w, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}

	h.logger.Info("password reset successful", "user_id", userID)

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Password has been reset successfully. You can now log in with your new password.",
	}); err != nil {
		h.logger.Error("failed to encode reset-password response", "error", err)
	}
}

func buildResetEmail(name, resetURL string) string {
	if name == "" {
		name = "there"
	}
	return `<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #1a1a1a;">Reset your password</h2>
  <p>Hi ` + name + `,</p>
  <p>We received a request to reset your password. Click the button below to choose a new one:</p>
  <p style="text-align: center; margin: 30px 0;">
    <a href="` + resetURL + `" style="background-color: #4f46e5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 600;">Reset Password</a>
  </p>
  <p style="color: #666; font-size: 14px;">This link will expire in 1 hour. If you didn't request this, you can safely ignore this email.</p>
  <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
  <p style="color: #999; font-size: 12px;">Sent by Rampart IAM</p>
</body>
</html>`
}
