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
	"github.com/manimovassagh/rampart/internal/session"
	"github.com/manimovassagh/rampart/internal/token"
)

const invalidCredentialsMsg = "Invalid credentials."

// LoginRequest is the expected JSON body for POST /login.
type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// LoginResponse is returned on successful authentication.
type LoginResponse struct {
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token"`
	TokenType    string             `json:"token_type"`
	ExpiresIn    int                `json:"expires_in"`
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
	GetUserByEmail(ctx context.Context, email string, orgID uuid.UUID) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string, orgID uuid.UUID) (*model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateLastLoginAt(ctx context.Context, userID uuid.UUID) error
}

// LoginHandler handles authentication endpoints.
type LoginHandler struct {
	store        LoginStore
	sessions     session.Store
	logger       *slog.Logger
	jwtSecret    string
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

// NewLoginHandler creates a handler with all authentication dependencies.
func NewLoginHandler(store LoginStore, sessions session.Store, logger *slog.Logger, jwtSecret string, accessTTL, refreshTTL time.Duration) *LoginHandler {
	return &LoginHandler{
		store:      store,
		sessions:   sessions,
		logger:     logger,
		jwtSecret:  jwtSecret,
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
		apierror.BadRequest(w, "Invalid or malformed JSON request body.")
		return
	}

	req.Identifier = strings.TrimSpace(req.Identifier)
	if req.Identifier == "" || req.Password == "" {
		apierror.Unauthorized(w, invalidCredentialsMsg)
		return
	}

	ctx := r.Context()

	orgID, err := h.store.GetDefaultOrganizationID(ctx)
	if err != nil {
		h.logger.Error("failed to get default organization", "error", err)
		apierror.InternalError(w)
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
		apierror.Unauthorized(w, invalidCredentialsMsg)
		return
	}

	if !user.Enabled {
		apierror.Unauthorized(w, invalidCredentialsMsg)
		return
	}

	ok, err := auth.VerifyPassword(req.Password, string(user.PasswordHash))
	if err != nil {
		h.logger.Error("failed to verify password", "error", err)
		apierror.InternalError(w)
		return
	}
	if !ok {
		apierror.Unauthorized(w, invalidCredentialsMsg)
		return
	}

	accessToken, err := token.GenerateAccessToken(
		h.jwtSecret, h.accessTTL,
		user.ID, user.OrgID,
		user.Username, user.Email, user.EmailVerified,
		user.GivenName, user.FamilyName,
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

	expiresAt := time.Now().Add(h.refreshTTL)
	if _, err := h.sessions.Create(ctx, user.ID, refreshToken, expiresAt); err != nil {
		h.logger.Error("failed to create session", "error", err)
		apierror.InternalError(w)
		return
	}

	if err := h.store.UpdateLastLoginAt(ctx, user.ID); err != nil {
		h.logger.Warn("failed to update last_login_at", "error", err, "user_id", user.ID)
	}

	resp := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.accessTTL.Seconds()),
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
		apierror.BadRequest(w, "Invalid or malformed JSON request body.")
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

	accessToken, err := token.GenerateAccessToken(
		h.jwtSecret, h.accessTTL,
		user.ID, user.OrgID,
		user.Username, user.Email, user.EmailVerified,
		user.GivenName, user.FamilyName,
	)
	if err != nil {
		h.logger.Error("failed to generate access token", "error", err)
		apierror.InternalError(w)
		return
	}

	resp := RefreshResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
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
		apierror.BadRequest(w, "Invalid or malformed JSON request body.")
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

	w.WriteHeader(http.StatusNoContent)
}
