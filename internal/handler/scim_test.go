package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
)

func TestScimOrgIDIgnoresHeaderWithoutSuperAdmin(t *testing.T) {
	userOrgID := uuid.New()
	headerOrgID := uuid.New()

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: userOrgID, Roles: []string{"admin"}}
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", http.NoBody)
	req.Header.Set("X-Org-Context", headerOrgID.String())
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	// Set the SCIM org context fallback
	ctx = context.WithValue(ctx, contextKey("scim_org_id"), userOrgID)
	req = req.WithContext(ctx)

	got := scimOrgID(req)
	if got != userOrgID {
		t.Errorf("scimOrgID() = %s, want %s (user's own org, not header)", got, userOrgID)
	}
}

func TestScimOrgIDAllowsSuperAdmin(t *testing.T) {
	userOrgID := uuid.New()
	headerOrgID := uuid.New()

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: userOrgID, Roles: []string{"super_admin"}}
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", http.NoBody)
	req.Header.Set("X-Org-Context", headerOrgID.String())
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	ctx = context.WithValue(ctx, contextKey("scim_org_id"), userOrgID)
	req = req.WithContext(ctx)

	got := scimOrgID(req)
	if got != headerOrgID {
		t.Errorf("scimOrgID() = %s, want %s (header org for super_admin)", got, headerOrgID)
	}
}

func TestScimOrgIDFallsBackWithNoHeader(t *testing.T) {
	userOrgID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", http.NoBody)
	ctx := context.WithValue(req.Context(), contextKey("scim_org_id"), userOrgID)
	req = req.WithContext(ctx)

	got := scimOrgID(req)
	if got != userOrgID {
		t.Errorf("scimOrgID() = %s, want %s (fallback from context)", got, userOrgID)
	}
}

func TestScimOrgIDInvalidUUIDFallsBack(t *testing.T) {
	userOrgID := uuid.New()

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: userOrgID, Roles: []string{"super_admin"}}
	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", http.NoBody)
	req.Header.Set("X-Org-Context", "not-a-uuid")
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	ctx = context.WithValue(ctx, contextKey("scim_org_id"), userOrgID)
	req = req.WithContext(ctx)

	got := scimOrgID(req)
	if got != userOrgID {
		t.Errorf("scimOrgID() = %s, want %s (fallback on invalid UUID)", got, userOrgID)
	}
}
