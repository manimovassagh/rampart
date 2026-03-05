package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
)

func TestMeSuccess(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	authUser := &middleware.AuthenticatedUser{
		UserID:            userID,
		OrgID:             orgID,
		PreferredUsername: "admin",
		Email:             "admin@test.com",
		EmailVerified:     true,
		GivenName:         "Admin",
		FamilyName:        "User",
	}

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	Me(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp MeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID != userID.String() {
		t.Errorf("id = %q, want %q", resp.ID, userID.String())
	}
	if resp.PreferredUsername != "admin" {
		t.Errorf("preferred_username = %q, want admin", resp.PreferredUsername)
	}
	if resp.Email != "admin@test.com" {
		t.Errorf("email = %q, want admin@test.com", resp.Email)
	}
	if !resp.EmailVerified {
		t.Error("email_verified = false, want true")
	}
}

func TestMeUnauthenticated(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	w := httptest.NewRecorder()

	Me(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMeNilUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()

	Me(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMeResponseContentType(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	authUser := &middleware.AuthenticatedUser{
		UserID:            userID,
		OrgID:             orgID,
		PreferredUsername: "testuser",
		Email:             "test@example.com",
	}

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	Me(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("content-type = %q, want application/json", contentType)
	}
}

func TestMeOptionalFields(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	authUser := &middleware.AuthenticatedUser{
		UserID:            userID,
		OrgID:             orgID,
		PreferredUsername: "minimal",
		Email:             "minimal@test.com",
		EmailVerified:     false,
		GivenName:         "",
		FamilyName:        "",
	}

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	Me(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp MeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.OrgID != orgID.String() {
		t.Errorf("org_id = %q, want %q", resp.OrgID, orgID.String())
	}
	if resp.GivenName != "" {
		t.Errorf("given_name = %q, want empty", resp.GivenName)
	}
	if resp.FamilyName != "" {
		t.Errorf("family_name = %q, want empty", resp.FamilyName)
	}
	if resp.EmailVerified {
		t.Error("email_verified = true, want false")
	}
}
