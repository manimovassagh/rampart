package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/model"
)

const (
	mfaOff      = "off"
	mfaOptional = "optional"
	mfaRequired = "required"
)

// OrgStore defines the database operations required by OrgHandler.
type OrgStore interface {
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*model.Organization, error)
	ListOrganizations(ctx context.Context, search string, limit, offset int) ([]*model.Organization, int, error)
	CreateOrganization(ctx context.Context, req *model.CreateOrgRequest) (*model.Organization, error)
	UpdateOrganization(ctx context.Context, id uuid.UUID, req *model.UpdateOrgRequest) (*model.Organization, error)
	DeleteOrganization(ctx context.Context, id uuid.UUID) error
	CountUsers(ctx context.Context, orgID uuid.UUID) (int, error)
}

// OrgSettingsStore defines the settings operations required by OrgHandler.
type OrgSettingsStore interface {
	GetOrgSettings(ctx context.Context, orgID uuid.UUID) (*model.OrgSettings, error)
	UpdateOrgSettings(ctx context.Context, orgID uuid.UUID, req *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error)
}

// OrgHandler handles organization management endpoints.
type OrgHandler struct {
	store    OrgStore
	settings OrgSettingsStore
	logger   *slog.Logger
}

// NewOrgHandler creates a handler with organization dependencies.
func NewOrgHandler(store OrgStore, settings OrgSettingsStore, logger *slog.Logger) *OrgHandler {
	return &OrgHandler{store: store, settings: settings, logger: logger}
}

// ListOrgs handles GET /api/v1/admin/organizations.
func (h *OrgHandler) ListOrgs(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", defaultPageLimit)

	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	ctx := r.Context()

	orgs, total, err := h.store.ListOrganizations(ctx, search, limit, offset)
	if err != nil {
		h.logger.Error("failed to list organizations", "error", err)
		apierror.InternalError(w)
		return
	}

	orgResponses := make([]*model.OrgResponse, len(orgs))
	for i, o := range orgs {
		count, err := h.store.CountUsers(ctx, o.ID)
		if err != nil {
			h.logger.Warn("failed to count users for org", "org_id", o.ID, "error", err)
		}
		orgResponses[i] = o.ToOrgResponse(count)
	}

	resp := model.ListOrgsResponse{
		Organizations: orgResponses,
		Total:         total,
		Page:          page,
		Limit:         limit,
	}

	writeJSON(w, http.StatusOK, resp, h.logger)
}

// CreateOrg handles POST /api/v1/admin/organizations.
func (h *OrgHandler) CreateOrg(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req model.CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if req.Name == "" || req.Slug == "" {
		apierror.BadRequest(w, "Name and slug are required.")
		return
	}

	ctx := r.Context()

	org, err := h.store.CreateOrganization(ctx, &req)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			apierror.Conflict(w, "An organization with this slug already exists.")
			return
		}
		h.logger.Error("failed to create organization", "error", err)
		apierror.InternalError(w)
		return
	}

	writeJSON(w, http.StatusCreated, org.ToOrgResponse(0), h.logger)
}

// GetOrg handles GET /api/v1/admin/organizations/{id}.
func (h *OrgHandler) GetOrg(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	ctx := r.Context()

	org, err := h.store.GetOrganizationByID(ctx, orgID)
	if err != nil {
		h.logger.Error("failed to get organization", "error", err)
		apierror.InternalError(w)
		return
	}
	if org == nil {
		apierror.NotFound(w)
		return
	}

	userCount, err := h.store.CountUsers(ctx, orgID)
	if err != nil {
		h.logger.Warn("failed to count users for org", "org_id", orgID, "error", err)
	}

	writeJSON(w, http.StatusOK, org.ToOrgResponse(userCount), h.logger)
}

// UpdateOrg handles PUT /api/v1/admin/organizations/{id}.
func (h *OrgHandler) UpdateOrg(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req model.UpdateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	ctx := r.Context()

	updated, err := h.store.UpdateOrganization(ctx, orgID, &req)
	if err != nil {
		h.logger.Error("failed to update organization", "error", err)
		apierror.InternalError(w)
		return
	}
	if updated == nil {
		apierror.NotFound(w)
		return
	}

	userCount, err := h.store.CountUsers(ctx, orgID)
	if err != nil {
		h.logger.Warn("failed to count users for org", "org_id", orgID, "error", err)
	}

	writeJSON(w, http.StatusOK, updated.ToOrgResponse(userCount), h.logger)
}

// DeleteOrg handles DELETE /api/v1/admin/organizations/{id}.
func (h *OrgHandler) DeleteOrg(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	if err := h.store.DeleteOrganization(r.Context(), orgID); err != nil {
		if strings.Contains(err.Error(), "default") {
			apierror.BadRequest(w, "Cannot delete the default organization.")
			return
		}
		if strings.Contains(err.Error(), "not found") {
			apierror.NotFound(w)
			return
		}
		h.logger.Error("failed to delete organization", "error", err)
		apierror.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetOrgSettings handles GET /api/v1/admin/organizations/{id}/settings.
func (h *OrgHandler) GetOrgSettings(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	settings, err := h.settings.GetOrgSettings(r.Context(), orgID)
	if err != nil {
		h.logger.Error("failed to get org settings", "error", err)
		apierror.InternalError(w)
		return
	}
	if settings == nil {
		apierror.NotFound(w)
		return
	}

	writeJSON(w, http.StatusOK, settings.ToResponse(), h.logger)
}

// UpdateOrgSettings handles PUT /api/v1/admin/organizations/{id}/settings.
func (h *OrgHandler) UpdateOrgSettings(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req model.UpdateOrgSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	if req.PasswordMinLength < 1 {
		apierror.BadRequest(w, "Password minimum length must be at least 1.")
		return
	}
	if req.MFAEnforcement != mfaOff && req.MFAEnforcement != mfaOptional && req.MFAEnforcement != mfaRequired {
		apierror.BadRequest(w, "MFA enforcement must be one of: off, optional, required.")
		return
	}
	if req.AccessTokenTTLSeconds < 60 {
		apierror.BadRequest(w, "Access token TTL must be at least 60 seconds.")
		return
	}
	if req.RefreshTokenTTLSeconds < 60 {
		apierror.BadRequest(w, "Refresh token TTL must be at least 60 seconds.")
		return
	}

	settings, err := h.settings.UpdateOrgSettings(r.Context(), orgID, &req)
	if err != nil {
		h.logger.Error("failed to update org settings", "error", err)
		apierror.InternalError(w)
		return
	}
	if settings == nil {
		apierror.NotFound(w)
		return
	}

	writeJSON(w, http.StatusOK, settings.ToResponse(), h.logger)
}
