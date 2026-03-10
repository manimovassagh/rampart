package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/store"
)

const (
	verifyTokenLength = 32 // 32 bytes = 64 hex chars
	verifyTokenTTL    = 24 * time.Hour
)

// EmailVerificationHandlerStore defines the database operations for email verification.
type EmailVerificationHandlerStore interface {
	store.OrgReader
	store.UserReader
	store.EmailVerificationTokenStore
	store.OrgSettingsReadWriter
}

// EmailVerificationHandler handles email verification endpoints.
type EmailVerificationHandler struct {
	store  EmailVerificationHandlerStore
	email  EmailSender
	logger *slog.Logger
	issuer string
}

// NewEmailVerificationHandler creates a new email verification handler.
func NewEmailVerificationHandler(s EmailVerificationHandlerStore, email EmailSender, logger *slog.Logger, issuer string) *EmailVerificationHandler {
	return &EmailVerificationHandler{
		store:  s,
		email:  email,
		logger: logger,
		issuer: issuer,
	}
}

// SendVerification handles POST /verify-email/send.
// Generates a verification token and sends it via email.
// Always returns 200 to prevent email enumeration.
func (h *EmailVerificationHandler) SendVerification(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	if req.Email == "" {
		apierror.BadRequest(w, "Email is required.")
		return
	}

	// Always return success to prevent email enumeration
	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "If an account with that email exists, a verification link has been sent.",
	}); err != nil {
		h.logger.Error("failed to encode send-verification response", "error", err)
	}

	go h.processVerificationRequest(req.Email)
}

func (h *EmailVerificationHandler) processVerificationRequest(email string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := h.store.FindUserByEmail(ctx, email)
	if err != nil {
		h.logger.Error("failed to find user for email verification", "error", err)
		return
	}
	if user == nil || !user.Enabled {
		return // silently ignore
	}

	if user.EmailVerified {
		return // already verified
	}

	// Generate token
	tokenBytes := make([]byte, verifyTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		h.logger.Error("failed to generate verification token", "error", err)
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Store token hash
	expiresAt := time.Now().Add(verifyTokenTTL)
	if err := h.store.CreateEmailVerificationToken(ctx, user.ID, token, expiresAt); err != nil {
		h.logger.Error("failed to store email verification token", "error", err)
		return
	}

	// Send email
	if !h.email.Enabled() {
		h.logger.Warn("SMTP not configured — email verification token generated but cannot be delivered",
			"user_id", user.ID,
			"token_prefix", token[:8],
		)
		return
	}

	verifyURL := h.issuer + "/verify-email?token=" + token
	body := buildVerificationEmail(user.GivenName, verifyURL)

	if err := h.email.Send(email, "Verify your email — Rampart", body); err != nil {
		h.logger.Error("failed to send verification email", "error", err, "user_id", user.ID)
	}
}

// SendVerificationForUser generates and sends a verification email for a specific user.
// Called internally after registration when email verification is required.
func (h *EmailVerificationHandler) SendVerificationForUser(user *model.User) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tokenBytes := make([]byte, verifyTokenLength)
		if _, err := rand.Read(tokenBytes); err != nil {
			h.logger.Error("failed to generate verification token", "error", err)
			return
		}
		token := hex.EncodeToString(tokenBytes)

		expiresAt := time.Now().Add(verifyTokenTTL)
		if err := h.store.CreateEmailVerificationToken(ctx, user.ID, token, expiresAt); err != nil {
			h.logger.Error("failed to store email verification token", "error", err)
			return
		}

		if !h.email.Enabled() {
			h.logger.Warn("SMTP not configured — email verification token generated but cannot be delivered",
				"user_id", user.ID,
				"token_prefix", token[:8],
			)
			return
		}

		verifyURL := h.issuer + "/verify-email?token=" + token
		body := buildVerificationEmail(user.GivenName, verifyURL)

		if err := h.email.Send(user.Email, "Verify your email — Rampart", body); err != nil {
			h.logger.Error("failed to send verification email", "error", err, "user_id", user.ID)
		}
	}()
}

// VerifyEmail handles GET /verify-email?token=xxx.
// Consumes the token and marks the user's email as verified.
func (h *EmailVerificationHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		apierror.BadRequest(w, "Verification token is required.")
		return
	}

	ctx := r.Context()

	userID, err := h.store.ConsumeEmailVerificationToken(ctx, token)
	if err != nil {
		apierror.Write(w, http.StatusBadRequest, "invalid_token", "Invalid, expired, or already-used verification token.")
		return
	}

	if err := h.store.MarkEmailVerified(ctx, userID); err != nil {
		h.logger.Error("failed to mark email verified", "error", err, "user_id", userID)
		apierror.InternalError(w)
		return
	}

	h.logger.Info("email verified", "user_id", userID)

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Email verified successfully. You can now log in.",
	}); err != nil {
		h.logger.Error("failed to encode verify-email response", "error", err)
	}
}

func buildVerificationEmail(name, verifyURL string) string {
	if name == "" {
		name = "there"
	}
	return `<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #1a1a1a;">Verify your email address</h2>
  <p>Hi ` + name + `,</p>
  <p>Thank you for registering. Please verify your email address by clicking the button below:</p>
  <p style="text-align: center; margin: 30px 0;">
    <a href="` + verifyURL + `" style="background-color: #4f46e5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 600;">Verify Email</a>
  </p>
  <p style="color: #666; font-size: 14px;">This link will expire in 24 hours. If you didn't create an account, you can safely ignore this email.</p>
  <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
  <p style="color: #999; font-size: 12px;">Sent by Rampart IAM</p>
</body>
</html>`
}
