package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
)

// mockMeStore is a test double for MeStore.
type mockMeStore struct {
	accounts []*model.SocialAccount
	err      error
}

func (m *mockMeStore) GetSocialAccountsByUserID(_ context.Context, _ uuid.UUID) ([]*model.SocialAccount, error) {
	return m.accounts, m.err
}

// ── stub methods to satisfy store.SocialAccountStore ──

func (m *mockMeStore) CreateSocialAccount(_ context.Context, _ *model.SocialAccount) (*model.SocialAccount, error) {
	return nil, nil
}
func (m *mockMeStore) GetSocialAccount(_ context.Context, _, _ string) (*model.SocialAccount, error) {
	return nil, nil
}
func (m *mockMeStore) UpdateSocialAccountTokens(_ context.Context, _ uuid.UUID, _, _ string, _ *time.Time) error {
	return nil
}
func (m *mockMeStore) DeleteSocialAccount(_ context.Context, _ uuid.UUID) error { return nil }

func newTestMeHandler(store MeStore) *MeHandler {
	return NewMeHandler(store)
}

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

	h := newTestMeHandler(&mockMeStore{})
	h.Me(w, req)

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

	h := newTestMeHandler(&mockMeStore{})
	h.Me(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMeNilUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()

	h := newTestMeHandler(&mockMeStore{})
	h.Me(w, req)

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

	h := newTestMeHandler(&mockMeStore{})
	h.Me(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != contentTypeJSON {
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

	h := newTestMeHandler(&mockMeStore{})
	h.Me(w, req)

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

func TestMeWithSocialAccounts(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	socialID := uuid.New()

	authUser := &middleware.AuthenticatedUser{
		UserID:            userID,
		OrgID:             orgID,
		PreferredUsername: "socialuser",
		Email:             "social@test.com",
		EmailVerified:     true,
	}

	store := &mockMeStore{
		accounts: []*model.SocialAccount{
			{
				ID:       socialID,
				UserID:   userID,
				Provider: "google",
				Email:    "social@gmail.com",
				Name:     "Social User",
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h := newTestMeHandler(store)
	h.Me(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp MeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.SocialAccounts) != 1 {
		t.Fatalf("social_accounts length = %d, want 1", len(resp.SocialAccounts))
	}
	if resp.SocialAccounts[0].Provider != "google" {
		t.Errorf("provider = %q, want google", resp.SocialAccounts[0].Provider)
	}
	if resp.SocialAccounts[0].Email != "social@gmail.com" {
		t.Errorf("email = %q, want social@gmail.com", resp.SocialAccounts[0].Email)
	}
	if resp.SocialAccounts[0].ID != socialID {
		t.Errorf("id = %q, want %q", resp.SocialAccounts[0].ID, socialID)
	}
}
