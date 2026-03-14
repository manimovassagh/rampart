// Package handler contains HTTP handlers for the Rampart server.
// scim.go implements SCIM 2.0 endpoints for user and group provisioning.
// See RFC 7643 (Core Schema) and RFC 7644 (Protocol) for the SCIM specification.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/store"
)

// SCIMStore defines the database operations required by SCIMHandler.
type SCIMStore interface {
	store.UserReader
	store.UserWriter
	store.UserLister
	store.GroupReader
	store.GroupWriter
	store.GroupLister
	store.OrgReader
}

// SCIMHandler implements SCIM 2.0 provisioning endpoints.
type SCIMHandler struct {
	store  SCIMStore
	logger *slog.Logger
}

// NewSCIMHandler creates a new SCIM 2.0 handler.
func NewSCIMHandler(s SCIMStore, logger *slog.Logger) *SCIMHandler {
	return &SCIMHandler{store: s, logger: logger}
}

const (
	scimMediaType = "application/scim+json"
	scimUserURN   = "urn:ietf:params:scim:schemas:core:2.0:User"
	scimGroupURN  = "urn:ietf:params:scim:schemas:core:2.0:Group"
	scimListURN   = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	scimErrorURN  = "urn:ietf:params:scim:api:messages:2.0:Error"
	scimSPCURN    = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
	scimRTURN     = "urn:ietf:params:scim:schemas:core:2.0:ResourceType"
	scimSchemaURN = "urn:ietf:params:scim:schemas:core:2.0:Schema"
)

// --- Discovery Endpoints ---

// ServiceProviderConfig handles GET /scim/v2/ServiceProviderConfig.
func (h *SCIMHandler) ServiceProviderConfig(w http.ResponseWriter, _ *http.Request) {
	resp := map[string]any{
		"schemas":          []string{scimSPCURN},
		"documentationUri": "https://tools.ietf.org/html/rfc7644",
		"patch":            map[string]bool{"supported": true},
		"bulk":             map[string]any{"supported": false, "maxOperations": 0, "maxPayloadSize": 0},
		"filter":           map[string]any{"supported": true, "maxResults": 100},
		"changePassword":   map[string]bool{"supported": false},
		"sort":             map[string]bool{"supported": false},
		"etag":             map[string]bool{"supported": false},
		"authenticationSchemes": []map[string]string{
			{
				"type":        "oauthbearertoken",
				"name":        "OAuth Bearer Token",
				"description": "Authentication scheme using the OAuth Bearer Token Standard",
			},
		},
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// ResourceTypes handles GET /scim/v2/ResourceTypes.
func (h *SCIMHandler) ResourceTypes(w http.ResponseWriter, _ *http.Request) {
	resp := []map[string]any{
		{
			"schemas":     []string{scimRTURN},
			"id":          "User",
			"name":        "User",
			"endpoint":    "/scim/v2/Users",
			"description": "User Account",
			"schema":      scimUserURN,
		},
		{
			"schemas":     []string{scimRTURN},
			"id":          "Group",
			"name":        "Group",
			"endpoint":    "/scim/v2/Groups",
			"description": "Group",
			"schema":      scimGroupURN,
		},
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// Schemas handles GET /scim/v2/Schemas.
func (h *SCIMHandler) Schemas(w http.ResponseWriter, _ *http.Request) {
	resp := []map[string]any{
		{
			"schemas": []string{scimSchemaURN},
			"id":      scimUserURN,
			"name":    "User",
			"attributes": []map[string]any{
				{"name": "userName", "type": "string", "required": true, "uniqueness": "server"},
				{"name": "name", "type": "complex", "subAttributes": []map[string]string{
					{"name": "givenName", "type": "string"},
					{"name": "familyName", "type": "string"},
				}},
				{"name": "emails", "type": "complex", "multiValued": true},
				{"name": "active", "type": "boolean"},
				{"name": "externalId", "type": "string"},
			},
		},
		{
			"schemas": []string{scimSchemaURN},
			"id":      scimGroupURN,
			"name":    "Group",
			"attributes": []map[string]any{
				{"name": "displayName", "type": "string", "required": true},
				{"name": "members", "type": "complex", "multiValued": true},
			},
		},
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// --- User Endpoints ---

// ListUsers handles GET /scim/v2/Users.
func (h *SCIMHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := scimOrgID(r)

	startIndex, count := parsePagination(r)
	filter := parseFilter(r.URL.Query().Get("filter"))

	users, total, err := h.store.ListUsers(ctx, orgID, filter, "", count, (startIndex - 1))
	if err != nil {
		h.scimError(w, http.StatusInternalServerError, "Failed to list users.")
		return
	}

	resources := make([]any, len(users))
	for i, u := range users {
		resources[i] = userToSCIM(u)
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"schemas":      []string{scimListURN},
		"totalResults": total,
		"startIndex":   startIndex,
		"itemsPerPage": count,
		"Resources":    resources,
	})
}

// GetUser handles GET /scim/v2/Users/{id}.
func (h *SCIMHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid user ID.")
		return
	}

	orgID := scimOrgID(r)
	user, err := h.store.GetUserByID(r.Context(), id)
	if err != nil || user == nil || user.OrgID != orgID {
		h.scimError(w, http.StatusNotFound, "User not found.")
		return
	}

	h.writeJSON(w, http.StatusOK, userToSCIM(user))
}

// CreateUser handles POST /scim/v2/Users.
func (h *SCIMHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := scimOrgID(r)

	var req scimUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}

	email := ""
	if len(req.Emails) > 0 {
		email = req.Emails[0].Value
	}

	user, err := h.store.CreateUser(ctx, &model.User{
		OrgID:         orgID,
		Username:      req.UserName,
		Email:         email,
		EmailVerified: true, // SCIM-provisioned users are pre-verified
		GivenName:     req.Name.GivenName,
		FamilyName:    req.Name.FamilyName,
		Enabled:       req.Active,
	})
	if err != nil {
		if errors.Is(err, store.ErrDuplicateKey) {
			h.scimError(w, http.StatusConflict, "User already exists.")
			return
		}
		h.logger.Error("SCIM: failed to create user", "error", err)
		h.scimError(w, http.StatusInternalServerError, "Failed to create user.")
		return
	}

	h.writeJSON(w, http.StatusCreated, userToSCIM(user))
}

// UpdateUser handles PUT /scim/v2/Users/{id}.
func (h *SCIMHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := scimOrgID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid user ID.")
		return
	}

	var req scimUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}

	email := ""
	if len(req.Emails) > 0 {
		email = req.Emails[0].Value
	}

	user, err := h.store.UpdateUser(ctx, id, orgID, &model.UpdateUserRequest{
		Username:   req.UserName,
		Email:      email,
		GivenName:  req.Name.GivenName,
		FamilyName: req.Name.FamilyName,
		Enabled:    req.Active,
	})
	if err != nil {
		h.logger.Error("SCIM: failed to update user", "error", err)
		h.scimError(w, http.StatusInternalServerError, "Failed to update user.")
		return
	}

	h.writeJSON(w, http.StatusOK, userToSCIM(user))
}

// PatchUser handles PATCH /scim/v2/Users/{id}.
func (h *SCIMHandler) PatchUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	patchOrgID := scimOrgID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid user ID.")
		return
	}

	var patch scimPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}

	// Build update request from patch operations
	updateReq := &model.UpdateUserRequest{}
	for _, op := range patch.Operations {
		switch strings.ToLower(op.Path) {
		case "active":
			if active, ok := op.Value.(bool); ok {
				updateReq.Enabled = active
			}
		case "username":
			if s, ok := op.Value.(string); ok {
				updateReq.Username = s
			}
		case "name.givenname":
			if s, ok := op.Value.(string); ok {
				updateReq.GivenName = s
			}
		case "name.familyname":
			if s, ok := op.Value.(string); ok {
				updateReq.FamilyName = s
			}
		}
	}

	user, err := h.store.UpdateUser(ctx, id, patchOrgID, updateReq)
	if err != nil {
		h.logger.Error("SCIM: failed to patch user", "error", err)
		h.scimError(w, http.StatusInternalServerError, "Failed to patch user.")
		return
	}

	h.writeJSON(w, http.StatusOK, userToSCIM(user))
}

// DeleteUser handles DELETE /scim/v2/Users/{id}.
func (h *SCIMHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid user ID.")
		return
	}

	delOrgID := scimOrgID(r)
	if err := h.store.DeleteUser(r.Context(), id, delOrgID); err != nil {
		h.logger.Error("SCIM: failed to delete user", "error", err)
		h.scimError(w, http.StatusInternalServerError, "Failed to delete user.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Group Endpoints ---

// ListGroups handles GET /scim/v2/Groups.
func (h *SCIMHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := scimOrgID(r)

	startIndex, count := parsePagination(r)
	filter := parseFilter(r.URL.Query().Get("filter"))

	groups, total, err := h.store.ListGroups(ctx, orgID, filter, count, (startIndex - 1))
	if err != nil {
		h.scimError(w, http.StatusInternalServerError, "Failed to list groups.")
		return
	}

	resources := make([]any, len(groups))
	for i, g := range groups {
		members, _ := h.store.GetGroupMembers(ctx, g.ID)
		resources[i] = groupToSCIM(g, members)
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"schemas":      []string{scimListURN},
		"totalResults": total,
		"startIndex":   startIndex,
		"itemsPerPage": count,
		"Resources":    resources,
	})
}

// GetGroup handles GET /scim/v2/Groups/{id}.
func (h *SCIMHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid group ID.")
		return
	}

	group, err := h.store.GetGroupByID(ctx, id)
	if err != nil || group == nil {
		h.scimError(w, http.StatusNotFound, "Group not found.")
		return
	}

	members, _ := h.store.GetGroupMembers(ctx, id)
	h.writeJSON(w, http.StatusOK, groupToSCIM(group, members))
}

// CreateGroup handles POST /scim/v2/Groups.
func (h *SCIMHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := scimOrgID(r)

	var req scimGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}

	group, err := h.store.CreateGroup(ctx, &model.Group{
		OrgID:       orgID,
		Name:        req.DisplayName,
		Description: req.DisplayName,
	})
	if err != nil {
		if errors.Is(err, store.ErrDuplicateKey) {
			h.scimError(w, http.StatusConflict, "Group already exists.")
			return
		}
		h.logger.Error("SCIM: failed to create group", "error", err)
		h.scimError(w, http.StatusInternalServerError, "Failed to create group.")
		return
	}

	// Add members if provided
	for _, m := range req.Members {
		memberID, err := uuid.Parse(m.Value)
		if err != nil {
			continue
		}
		_ = h.store.AddUserToGroup(ctx, memberID, group.ID)
	}

	members, _ := h.store.GetGroupMembers(ctx, group.ID)
	h.writeJSON(w, http.StatusCreated, groupToSCIM(group, members))
}

// UpdateGroup handles PUT /scim/v2/Groups/{id}.
func (h *SCIMHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid group ID.")
		return
	}

	var req scimGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}

	group, err := h.store.UpdateGroup(ctx, id, &model.UpdateGroupRequest{
		Name:        req.DisplayName,
		Description: req.DisplayName,
	})
	if err != nil {
		h.logger.Error("SCIM: failed to update group", "error", err)
		h.scimError(w, http.StatusInternalServerError, "Failed to update group.")
		return
	}

	members, _ := h.store.GetGroupMembers(ctx, id)
	h.writeJSON(w, http.StatusOK, groupToSCIM(group, members))
}

// PatchGroup handles PATCH /scim/v2/Groups/{id}.
func (h *SCIMHandler) PatchGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid group ID.")
		return
	}

	var patch scimPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}

	for _, op := range patch.Operations {
		switch strings.ToLower(op.Op) {
		case "add":
			if strings.EqualFold(op.Path, "members") {
				h.patchAddMembers(ctx, id, op.Value)
			}
		case "remove":
			if strings.HasPrefix(strings.ToLower(op.Path), "members[") {
				h.patchRemoveMember(ctx, id, op.Path)
			}
		case "replace":
			if strings.EqualFold(op.Path, "displayName") {
				if s, ok := op.Value.(string); ok {
					_, _ = h.store.UpdateGroup(ctx, id, &model.UpdateGroupRequest{Name: s, Description: s})
				}
			}
		}
	}

	group, err := h.store.GetGroupByID(ctx, id)
	if err != nil || group == nil {
		h.scimError(w, http.StatusNotFound, "Group not found.")
		return
	}

	members, _ := h.store.GetGroupMembers(ctx, id)
	h.writeJSON(w, http.StatusOK, groupToSCIM(group, members))
}

// DeleteGroup handles DELETE /scim/v2/Groups/{id}.
func (h *SCIMHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.scimError(w, http.StatusBadRequest, "Invalid group ID.")
		return
	}

	if err := h.store.DeleteGroup(r.Context(), id); err != nil {
		h.logger.Error("SCIM: failed to delete group", "error", err)
		h.scimError(w, http.StatusInternalServerError, "Failed to delete group.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---

func (h *SCIMHandler) patchAddMembers(ctx context.Context, groupID uuid.UUID, value any) {
	// Value can be a single member or an array of members
	switch v := value.(type) {
	case []any:
		for _, m := range v {
			if member, ok := m.(map[string]any); ok {
				if val, ok := member["value"].(string); ok {
					if memberID, err := uuid.Parse(val); err == nil {
						_ = h.store.AddUserToGroup(ctx, memberID, groupID)
					}
				}
			}
		}
	case map[string]any:
		if val, ok := v["value"].(string); ok {
			if memberID, err := uuid.Parse(val); err == nil {
				_ = h.store.AddUserToGroup(ctx, memberID, groupID)
			}
		}
	}
}

func (h *SCIMHandler) patchRemoveMember(ctx context.Context, groupID uuid.UUID, path string) {
	// Parse "members[value eq \"<uuid>\"]"
	start := strings.Index(path, "\"")
	end := strings.LastIndex(path, "\"")
	if start >= 0 && end > start {
		if memberID, err := uuid.Parse(path[start+1 : end]); err == nil {
			_ = h.store.RemoveUserFromGroup(ctx, memberID, groupID)
		}
	}
}

func (h *SCIMHandler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", scimMediaType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("SCIM: failed to encode response", "error", err)
	}
}

func (h *SCIMHandler) scimError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", scimMediaType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"schemas": []string{scimErrorURN},
		"status":  fmt.Sprintf("%d", status),
		"detail":  detail,
	})
}

// scimOrgID extracts the organization ID for this SCIM request.
// Only super_admin users may switch orgs via the X-Org-Context header.
func scimOrgID(r *http.Request) uuid.UUID {
	if orgStr := r.Header.Get("X-Org-Context"); orgStr != "" {
		if id, err := uuid.Parse(orgStr); err == nil {
			authUser := middleware.GetAuthenticatedUser(r.Context())
			if authUser != nil && authUser.HasRole("super_admin") {
				return id
			}
		}
	}
	// Fall back to org ID from authenticated context
	if orgID, ok := r.Context().Value(scimOrgContextKey).(uuid.UUID); ok {
		return orgID
	}
	return uuid.Nil
}

type contextKey string

const scimOrgContextKey contextKey = "scim_org_id"

// --- SCIM Data Types ---

type scimUserRequest struct {
	UserName   string      `json:"userName"`
	Name       scimName    `json:"name"`
	Emails     []scimEmail `json:"emails"`
	Active     bool        `json:"active"`
	ExternalID string      `json:"externalId"`
}

type scimName struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

type scimEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type"`
	Primary bool   `json:"primary"`
}

type scimGroupRequest struct {
	DisplayName string       `json:"displayName"`
	Members     []scimMember `json:"members"`
}

type scimMember struct {
	Value   string `json:"value"`
	Display string `json:"display"`
}

type scimPatchRequest struct {
	Schemas    []string      `json:"schemas"`
	Operations []scimPatchOp `json:"Operations"`
}

type scimPatchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value"`
}

func userToSCIM(u *model.User) map[string]any {
	emails := []map[string]any{}
	if u.Email != "" {
		emails = append(emails, map[string]any{
			"value":   u.Email,
			"type":    "work",
			"primary": true,
		})
	}

	return map[string]any{
		"schemas":  []string{scimUserURN},
		"id":       u.ID.String(),
		"userName": u.Username,
		"name": map[string]string{
			"givenName":  u.GivenName,
			"familyName": u.FamilyName,
		},
		"emails": emails,
		"active": u.Enabled,
		"meta": map[string]string{
			"resourceType": "User",
			"created":      u.CreatedAt.Format("2006-01-02T15:04:05Z"),
			"lastModified": u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		},
	}
}

func groupToSCIM(g *model.Group, members []*model.GroupMember) map[string]any {
	scimMembers := make([]map[string]string, len(members))
	for i, m := range members {
		scimMembers[i] = map[string]string{
			"value":   m.UserID.String(),
			"display": m.Username,
		}
	}

	return map[string]any{
		"schemas":     []string{scimGroupURN},
		"id":          g.ID.String(),
		"displayName": g.Name,
		"members":     scimMembers,
		"meta": map[string]string{
			"resourceType": "Group",
			"created":      g.CreatedAt.Format("2006-01-02T15:04:05Z"),
			"lastModified": g.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		},
	}
}

func parsePagination(r *http.Request) (startIndex, count int) {
	startIndex = 1
	count = 100
	if s := r.URL.Query().Get("startIndex"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			startIndex = v
		}
	}
	if c := r.URL.Query().Get("count"); c != "" {
		if v, err := strconv.Atoi(c); err == nil && v > 0 && v <= 100 {
			count = v
		}
	}
	return
}

// parseFilter extracts a simple filter value from SCIM filter syntax.
// Supports: userName eq "value", displayName eq "value".
func parseFilter(filter string) string {
	if filter == "" {
		return ""
	}
	// Simple extraction: look for the value between quotes
	start := strings.Index(filter, "\"")
	end := strings.LastIndex(filter, "\"")
	if start >= 0 && end > start {
		return filter[start+1 : end]
	}
	return ""
}
