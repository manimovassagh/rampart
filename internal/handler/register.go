package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/model"
)

const (
	maxRequestBodySize   = 1 << 20 // 1 MB
	minResponseDuration  = 250 * time.Millisecond
)

// UserStore defines the database operations required by RegisterHandler.
type UserStore interface {
	GetDefaultOrganizationID(ctx context.Context) (uuid.UUID, error)
	GetUserByEmail(ctx context.Context, email string, orgID uuid.UUID) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string, orgID uuid.UUID) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
}

// RegisterHandler handles user self-registration.
type RegisterHandler struct {
	store  UserStore
	logger *slog.Logger
}

// NewRegisterHandler creates a handler with a user store dependency.
func NewRegisterHandler(store UserStore, logger *slog.Logger) *RegisterHandler {
	return &RegisterHandler{store: store, logger: logger}
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
		apierror.BadRequest(w, "Invalid or malformed JSON request body.")
		return
	}

	// Normalize inputs.
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Username = strings.TrimSpace(req.Username)
	req.GivenName = strings.TrimSpace(req.GivenName)
	req.FamilyName = strings.TrimSpace(req.FamilyName)

	// Validate all fields.
	if fieldErrors := auth.ValidateRegistration(req.Email, req.Password, req.Username); len(fieldErrors) > 0 {
		apiFieldErrors := make([]apierror.FieldError, len(fieldErrors))
		for i, fe := range fieldErrors {
			apiFieldErrors[i] = apierror.FieldError{Field: fe.Field, Message: fe.Message}
		}
		apierror.WriteValidation(w, apiFieldErrors)
		return
	}

	ctx := r.Context()

	// Look up default organization.
	orgID, err := h.store.GetDefaultOrganizationID(ctx)
	if err != nil {
		h.logger.Error("failed to get default organization", "error", err)
		apierror.InternalError(w)
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
		apierror.Conflict(w, "A user with this email already exists.")
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
		apierror.Conflict(w, "A user with this username already exists.")
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
		h.logger.Error("failed to create user", "error", err)
		apierror.InternalError(w)
		return
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(created.ToResponse()); err != nil {
		h.logger.Error("failed to encode registration response", "error", err)
	}
}
