package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/auth"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

// AdminUserStore defines the database operations required by AdminHandler.
type AdminUserStore interface {
	GetUserByEmail(ctx context.Context, email string, orgID uuid.UUID) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string, orgID uuid.UUID) (*model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	ListUsers(ctx context.Context, orgID uuid.UUID, search, status string, limit, offset int) ([]*model.User, int, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req *model.UpdateUserRequest) (*model.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash []byte) error
	CountUsers(ctx context.Context, orgID uuid.UUID) (int, error)
	CountRecentUsers(ctx context.Context, orgID uuid.UUID, days int) (int, error)
	CountOrganizations(ctx context.Context) (int, error)
	GetOrgSettings(ctx context.Context, orgID uuid.UUID) (*model.OrgSettings, error)
}

// AdminSessionStore defines the session operations required by AdminHandler.
type AdminSessionStore interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*session.Session, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	CountActive(ctx context.Context) (int, error)
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

// AdminHandler handles admin console endpoints.
type AdminHandler struct {
	store    AdminUserStore
	sessions AdminSessionStore
	logger   *slog.Logger
}

// NewAdminHandler creates a handler with admin dependencies.
func NewAdminHandler(store AdminUserStore, sessions AdminSessionStore, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{store: store, sessions: sessions, logger: logger}
}

// Stats handles GET /api/v1/admin/stats.
func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	authUser := middleware.GetAuthenticatedUser(ctx)
	if authUser == nil {
		apierror.Unauthorized(w, "Authentication required.")
		return
	}
	orgID := authUser.OrgID

	totalUsers, err := h.store.CountUsers(ctx, orgID)
	if err != nil {
		h.logger.Error("failed to count users", "error", err)
		apierror.InternalError(w)
		return
	}

	activeSessions, err := h.sessions.CountActive(ctx)
	if err != nil {
		h.logger.Error("failed to count active sessions", "error", err)
		apierror.InternalError(w)
		return
	}

	recentUsers, err := h.store.CountRecentUsers(ctx, orgID, 7)
	if err != nil {
		h.logger.Error("failed to count recent users", "error", err)
		apierror.InternalError(w)
		return
	}

	totalOrgs, err := h.store.CountOrganizations(ctx)
	if err != nil {
		h.logger.Error("failed to count organizations", "error", err)
		apierror.InternalError(w)
		return
	}

	stats := model.DashboardStats{
		TotalUsers:         totalUsers,
		ActiveSessions:     activeSessions,
		RecentUsers:        recentUsers,
		TotalOrganizations: totalOrgs,
	}

	writeJSON(w, http.StatusOK, stats, h.logger)
}

// ListUsers handles GET /api/v1/admin/users.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	authUser := middleware.GetAuthenticatedUser(ctx)
	if authUser == nil {
		apierror.Unauthorized(w, "Authentication required.")
		return
	}
	orgID := authUser.OrgID

	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", defaultPageLimit)

	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	users, total, err := h.store.ListUsers(ctx, orgID, search, status, limit, offset)
	if err != nil {
		h.logger.Error("failed to list users", "error", err)
		apierror.InternalError(w)
		return
	}

	adminUsers := make([]*model.AdminUserResponse, len(users))
	for i, u := range users {
		count, err := h.sessions.CountByUserID(ctx, u.ID)
		if err != nil {
			h.logger.Warn("failed to count sessions for user", "user_id", u.ID, "error", err)
		}
		adminUsers[i] = u.ToAdminResponse(count)
	}

	resp := model.ListUsersResponse{
		Users: adminUsers,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	writeJSON(w, http.StatusOK, resp, h.logger)
}

// CreateUser handles POST /api/v1/admin/users.
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid or malformed JSON request body.")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Username = strings.TrimSpace(req.Username)
	req.GivenName = strings.TrimSpace(req.GivenName)
	req.FamilyName = strings.TrimSpace(req.FamilyName)

	ctx := r.Context()

	authUser := middleware.GetAuthenticatedUser(ctx)
	if authUser == nil {
		apierror.Unauthorized(w, "Authentication required.")
		return
	}
	orgID := authUser.OrgID

	// Fetch per-org password policy (fall back to defaults if settings not found).
	passwordPolicy := auth.DefaultPasswordPolicy()
	if settings, sErr := h.store.GetOrgSettings(ctx, orgID); sErr != nil {
		h.logger.Warn("failed to fetch org settings, using defaults", "error", sErr)
	} else if settings != nil {
		passwordPolicy = auth.PasswordPolicy{
			MinLength:        settings.PasswordMinLength,
			RequireUppercase: settings.PasswordRequireUppercase,
			RequireLowercase: settings.PasswordRequireLowercase,
			RequireNumbers:   settings.PasswordRequireNumbers,
			RequireSymbols:   settings.PasswordRequireSymbols,
		}
	}

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
	if len(fieldErrors) > 0 {
		apiFieldErrors := make([]apierror.FieldError, len(fieldErrors))
		for i, fe := range fieldErrors {
			apiFieldErrors[i] = apierror.FieldError{Field: fe.Field, Message: fe.Message}
		}
		apierror.WriteValidation(w, apiFieldErrors)
		return
	}

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

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		apierror.InternalError(w)
		return
	}

	user := &model.User{
		OrgID:        orgID,
		Username:     req.Username,
		Email:        req.Email,
		GivenName:    req.GivenName,
		FamilyName:   req.FamilyName,
		PasswordHash: []byte(hash),
		Enabled:      req.Enabled,
	}

	created, err := h.store.CreateUser(ctx, user)
	if err != nil {
		h.logger.Error("failed to create user", "error", err)
		apierror.InternalError(w)
		return
	}

	writeJSON(w, http.StatusCreated, created.ToAdminResponse(0), h.logger)
}

// GetUser handles GET /api/v1/admin/users/{id}.
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	ctx := r.Context()

	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get user", "error", err)
		apierror.InternalError(w)
		return
	}
	if user == nil {
		apierror.NotFound(w)
		return
	}

	sessionCount, err := h.sessions.CountByUserID(ctx, userID)
	if err != nil {
		h.logger.Warn("failed to count sessions", "user_id", userID, "error", err)
	}

	writeJSON(w, http.StatusOK, user.ToAdminResponse(sessionCount), h.logger)
}

// UpdateUser handles PUT /api/v1/admin/users/{id}.
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid or malformed JSON request body.")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.GivenName = strings.TrimSpace(req.GivenName)
	req.FamilyName = strings.TrimSpace(req.FamilyName)

	ctx := r.Context()

	updated, err := h.store.UpdateUser(ctx, userID, &req)
	if err != nil {
		h.logger.Error("failed to update user", "error", err)
		apierror.InternalError(w)
		return
	}
	if updated == nil {
		apierror.NotFound(w)
		return
	}

	sessionCount, err := h.sessions.CountByUserID(ctx, userID)
	if err != nil {
		h.logger.Warn("failed to count sessions", "user_id", userID, "error", err)
	}

	writeJSON(w, http.StatusOK, updated.ToAdminResponse(sessionCount), h.logger)
}

// DeleteUser handles DELETE /api/v1/admin/users/{id}.
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	// Prevent self-deletion.
	authUser := middleware.GetAuthenticatedUser(r.Context())
	if authUser != nil && authUser.UserID == userID {
		apierror.BadRequest(w, "You cannot delete your own account.")
		return
	}

	ctx := r.Context()

	// Delete sessions first.
	if err := h.sessions.DeleteByUserID(ctx, userID); err != nil {
		h.logger.Error("failed to delete user sessions", "error", err)
		apierror.InternalError(w)
		return
	}

	if err := h.store.DeleteUser(ctx, userID); err != nil {
		h.logger.Error("failed to delete user", "error", err)
		apierror.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ResetPassword handles POST /api/v1/admin/users/{id}/reset-password.
func (h *AdminHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req model.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "Invalid or malformed JSON request body.")
		return
	}

	if fe := auth.ValidatePassword(req.Password); fe != nil {
		apierror.WriteValidation(w, []apierror.FieldError{{Field: fe.Field, Message: fe.Message}})
		return
	}

	ctx := r.Context()

	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get user", "error", err)
		apierror.InternalError(w)
		return
	}
	if user == nil {
		apierror.NotFound(w)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		apierror.InternalError(w)
		return
	}

	if err := h.store.UpdatePassword(ctx, userID, []byte(hash)); err != nil {
		h.logger.Error("failed to update password", "error", err)
		apierror.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListSessions handles GET /api/v1/admin/users/{id}/sessions.
func (h *AdminHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	sessions, err := h.sessions.ListByUserID(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list sessions", "error", err)
		apierror.InternalError(w)
		return
	}

	resp := make([]*model.SessionResponse, len(sessions))
	for i, s := range sessions {
		resp[i] = &model.SessionResponse{
			ID:        s.ID,
			CreatedAt: s.CreatedAt,
			ExpiresAt: s.ExpiresAt,
		}
	}

	writeJSON(w, http.StatusOK, resp, h.logger)
}

// RevokeSessions handles DELETE /api/v1/admin/users/{id}/sessions.
func (h *AdminHandler) RevokeSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	if err := h.sessions.DeleteByUserID(r.Context(), userID); err != nil {
		h.logger.Error("failed to revoke sessions", "error", err)
		apierror.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---

func parseUUIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := chi.URLParam(r, "id")
	id, err := uuid.Parse(raw)
	if err != nil {
		apierror.BadRequest(w, "Invalid UUID parameter.")
		return uuid.UUID{}, false
	}
	return id, true
}

func queryInt(r *http.Request, key string, fallback int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, v any, logger *slog.Logger) {
	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Error("failed to encode response", "error", err)
	}
}
