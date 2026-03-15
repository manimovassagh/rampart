package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/store"
)

const (
	maxRequestBodySize  = 1 << 20 // 1 MB
	minResponseDuration = 250 * time.Millisecond
)

// UserStore defines the database operations required by RegisterHandler.
type UserStore interface {
	store.OrgReader
	store.UserReader
	store.UserWriter
	store.OrgSettingsReadWriter
}

// RegisterHandler handles user self-registration.
type RegisterHandler struct {
	store         UserStore
	logger        *slog.Logger
	emailVerifier *EmailVerificationHandler // optional, nil if email verification disabled
}

// NewRegisterHandler creates a handler with a user store dependency.
func NewRegisterHandler(s UserStore, logger *slog.Logger) *RegisterHandler {
	return &RegisterHandler{store: s, logger: logger}
}

// SetEmailVerifier sets the email verification handler for post-registration flows.
func (h *RegisterHandler) SetEmailVerifier(v *EmailVerificationHandler) {
	h.emailVerifier = v
}

// Register handles POST /register.
func (h *RegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		// Timing safety: ensure consistent response time to prevent user enumeration.
		elapsed := time.Since(start)
		if elapsed < minResponseDuration {
			time.Sleep(minResponseDuration - elapsed)
		}
	}()

	// Limit request body size to prevent memory exhaustion.
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req model.RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	// Normalize inputs.
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	req.GivenName = strings.TrimSpace(req.GivenName)
	req.FamilyName = strings.TrimSpace(req.FamilyName)
	req.OrgSlug = strings.ToLower(strings.TrimSpace(req.OrgSlug))

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
		h.logger.Error("failed to resolve organization", "error", err, "org_slug", req.OrgSlug)
		apierror.BadRequest(w, "Organization not found.")
		return
	}

	// Fetch per-org password policy (fall back to defaults if settings not found).
	passwordPolicy := auth.DefaultPasswordPolicy()
	if settings, err := h.store.GetOrgSettings(ctx, orgID); err != nil {
		h.logger.Warn("failed to fetch org settings, using defaults", "error", err)
	} else if settings != nil {
		passwordPolicy = auth.PasswordPolicy{
			MinLength:        settings.PasswordMinLength,
			RequireUppercase: settings.PasswordRequireUppercase,
			RequireLowercase: settings.PasswordRequireLowercase,
			RequireNumbers:   settings.PasswordRequireNumbers,
			RequireSymbols:   settings.PasswordRequireSymbols,
		}
	}

	// Validate fields with per-org password policy.
	var fieldErrors []auth.FieldError
	if fe := auth.ValidateEmail(req.Email); fe != nil {
		fieldErrors = append(fieldErrors, *fe)
	}
	if fe := auth.ValidatePasswordWithPolicy(req.Password, passwordPolicy); fe != nil {
		fieldErrors = append(fieldErrors, *fe)
	}
	if fe := auth.ValidateUsername(req.Username); fe != nil {
		fieldErrors = append(fieldErrors, *fe)
	}
	if fe := auth.ValidateName("given_name", req.GivenName); fe != nil {
		fieldErrors = append(fieldErrors, *fe)
	}
	if fe := auth.ValidateName("family_name", req.FamilyName); fe != nil {
		fieldErrors = append(fieldErrors, *fe)
	}
	if len(fieldErrors) > 0 {
		apiFieldErrors := make([]apierror.FieldError, len(fieldErrors))
		for i, fe := range fieldErrors {
			apiFieldErrors[i] = apierror.FieldError{Field: fe.Field, Message: fe.Message}
		}
		apierror.WriteValidation(w, apiFieldErrors)
		return
	}

	// Check for existing email.
	existing, err := h.store.GetUserByEmail(ctx, req.Email, orgID)
	if err != nil {
		h.logger.Error("failed to check existing email", "error", err)
		apierror.InternalError(w)
		return
	}
	if existing != nil {
		apierror.Conflict(w, "Registration failed. Please try again or use a different email/username.")
		return
	}

	// Check for existing username.
	existing, err = h.store.GetUserByUsername(ctx, req.Username, orgID)
	if err != nil {
		h.logger.Error("failed to check existing username", "error", err)
		apierror.InternalError(w)
		return
	}
	if existing != nil {
		apierror.Conflict(w, "Registration failed. Please try again or use a different email/username.")
		return
	}

	// Hash password.
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		apierror.InternalError(w)
		return
	}

	// Create user.
	user := &model.User{
		OrgID:        orgID,
		Username:     req.Username,
		Email:        req.Email,
		GivenName:    req.GivenName,
		FamilyName:   req.FamilyName,
		PasswordHash: []byte(hash),
	}

	created, err := h.store.CreateUser(ctx, user)
	if err != nil {
		if errors.Is(err, store.ErrDuplicateKey) {
			apierror.Conflict(w, "Registration failed. Please try again or use a different email/username.")
			return
		}
		h.logger.Error("failed to create user", "error", err)
		apierror.InternalError(w)
		return
	}

	// Send verification email if email verification is required
	if h.emailVerifier != nil {
		if settings, sErr := h.store.GetOrgSettings(ctx, orgID); sErr == nil && settings != nil && settings.EmailVerificationRequired {
			h.emailVerifier.SendVerificationForUser(created)
		}
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(created.ToResponse()); err != nil {
		h.logger.Error("failed to encode registration response", "error", err)
	}
}
