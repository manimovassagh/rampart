package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
)

func testClient(t *testing.T, db *DB, orgID uuid.UUID) *model.OAuthClient {
	t.Helper()
	ctx := context.Background()
	client, err := db.CreateOAuthClient(ctx, &model.OAuthClient{
		OrgID:        orgID,
		Name:         "consent-client-" + uniqueSlug(""),
		ClientType:   "public",
		RedirectURIs: []string{"http://localhost/cb"},
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("creating test client: %v", err)
	}
	return client
}

func TestConsentGrantAndCheck(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)
	user := testUser(t, db, org.ID)
	client := testClient(t, db, org.ID)

	// Initially no consent
	has, err := db.HasConsent(ctx, user.ID, client.ID, "openid profile")
	if err != nil {
		t.Fatalf("HasConsent: %v", err)
	}
	if has {
		t.Error("expected no consent initially")
	}

	// Grant consent
	err = db.GrantConsent(ctx, user.ID, client.ID, "openid profile")
	if err != nil {
		t.Fatalf("GrantConsent: %v", err)
	}

	// Now has consent
	has, err = db.HasConsent(ctx, user.ID, client.ID, "openid profile")
	if err != nil {
		t.Fatalf("HasConsent after grant: %v", err)
	}
	if !has {
		t.Error("expected consent after grant")
	}

	// Different scopes should not match
	has, err = db.HasConsent(ctx, user.ID, client.ID, "openid")
	if err != nil {
		t.Fatalf("HasConsent different scopes: %v", err)
	}
	if has {
		t.Error("expected no consent for different scopes")
	}
}

func TestConsentGrantIdempotent(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)
	user := testUser(t, db, org.ID)
	client := testClient(t, db, org.ID)

	// Grant same consent twice (ON CONFLICT DO UPDATE)
	err := db.GrantConsent(ctx, user.ID, client.ID, "openid")
	if err != nil {
		t.Fatalf("GrantConsent: %v", err)
	}
	err = db.GrantConsent(ctx, user.ID, client.ID, "openid profile")
	if err != nil {
		t.Fatalf("GrantConsent (update): %v", err)
	}

	// Should now have the updated scopes
	has, err := db.HasConsent(ctx, user.ID, client.ID, "openid profile")
	if err != nil {
		t.Fatalf("HasConsent: %v", err)
	}
	if !has {
		t.Error("expected updated consent")
	}
}

func TestConsentGetAndList(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)
	user := testUser(t, db, org.ID)
	client := testClient(t, db, org.ID)

	err := db.GrantConsent(ctx, user.ID, client.ID, "openid email")
	if err != nil {
		t.Fatalf("GrantConsent: %v", err)
	}

	// GetConsent
	consent, err := db.GetConsent(ctx, user.ID, client.ID)
	if err != nil {
		t.Fatalf("GetConsent: %v", err)
	}
	if consent == nil {
		t.Fatal("expected consent, got nil")
	}
	if consent.Scopes != "openid email" {
		t.Errorf("scopes: got %q, want %q", consent.Scopes, "openid email")
	}
	if consent.UserID != user.ID {
		t.Errorf("user_id: got %v, want %v", consent.UserID, user.ID)
	}
	if consent.ClientID != client.ID {
		t.Errorf("client_id: got %v, want %v", consent.ClientID, client.ID)
	}

	// GetConsent not found
	missing, err := db.GetConsent(ctx, user.ID, "nonexistent-client")
	if err != nil {
		t.Fatalf("GetConsent (miss): %v", err)
	}
	if missing != nil {
		t.Error("expected nil for nonexistent consent")
	}

	// ListUserConsents
	consents, err := db.ListUserConsents(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListUserConsents: %v", err)
	}
	if len(consents) < 1 {
		t.Error("expected at least 1 consent")
	}

	// Empty list for unknown user
	empty, err := db.ListUserConsents(ctx, uuid.New())
	if err != nil {
		t.Fatalf("ListUserConsents (empty): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 consents for unknown user, got %d", len(empty))
	}
}

func TestConsentRevoke(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)
	user := testUser(t, db, org.ID)
	client := testClient(t, db, org.ID)

	err := db.GrantConsent(ctx, user.ID, client.ID, "openid")
	if err != nil {
		t.Fatalf("GrantConsent: %v", err)
	}

	// Revoke
	err = db.RevokeConsent(ctx, user.ID, client.ID)
	if err != nil {
		t.Fatalf("RevokeConsent: %v", err)
	}

	// Verify revoked
	has, err := db.HasConsent(ctx, user.ID, client.ID, "openid")
	if err != nil {
		t.Fatalf("HasConsent after revoke: %v", err)
	}
	if has {
		t.Error("expected no consent after revoke")
	}

	// Revoking nonexistent should not error
	err = db.RevokeConsent(ctx, user.ID, "nonexistent-client")
	if err != nil {
		t.Fatalf("RevokeConsent (nonexistent): %v", err)
	}
}
