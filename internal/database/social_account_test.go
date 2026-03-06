package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
)

const (
	providerGoogle = "google"
	providerGitHub = "github"
)

func testUser(t *testing.T, db *DB, orgID uuid.UUID) *model.User {
	t.Helper()
	ctx := context.Background()
	user, err := db.CreateUser(ctx, &model.User{
		OrgID:        orgID,
		Username:     "social-" + uniqueSlug(""),
		Email:        uniqueSlug("s") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}
	return user
}

func TestCreateSocialAccount(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)
	user := testUser(t, db, org.ID)

	tests := []struct {
		name           string
		provider       string
		providerUserID string
		email          string
		displayName    string
	}{
		{"google account", providerGoogle, "google-" + uniqueSlug(""), "alice@gmail.com", "Alice"},
		{"github account", providerGitHub, "github-" + uniqueSlug(""), "alice@github.com", "alice"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			created, err := db.CreateSocialAccount(ctx, &model.SocialAccount{
				UserID:         user.ID,
				Provider:       tc.provider,
				ProviderUserID: tc.providerUserID,
				Email:          tc.email,
				Name:           tc.displayName,
			})
			if err != nil {
				t.Fatalf("CreateSocialAccount: %v", err)
			}
			if created.ID == uuid.Nil {
				t.Fatal("expected non-nil UUID")
			}
			if created.Provider != tc.provider {
				t.Errorf("provider: got %q, want %q", created.Provider, tc.provider)
			}
			if created.ProviderUserID != tc.providerUserID {
				t.Errorf("provider_user_id: got %q, want %q", created.ProviderUserID, tc.providerUserID)
			}
			if created.Email != tc.email {
				t.Errorf("email: got %q, want %q", created.Email, tc.email)
			}
			if created.Name != tc.displayName {
				t.Errorf("name: got %q, want %q", created.Name, tc.displayName)
			}
			if created.CreatedAt.IsZero() {
				t.Error("expected created_at to be set")
			}
		})
	}
}

func TestGetSocialAccountByProvider(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)
	user := testUser(t, db, org.ID)

	providerUID := "get-test-" + uniqueSlug("")
	created, err := db.CreateSocialAccount(ctx, &model.SocialAccount{
		UserID:         user.ID,
		Provider:       providerGoogle,
		ProviderUserID: providerUID,
		Email:          "get@gmail.com",
		Name:           "Get Test",
	})
	if err != nil {
		t.Fatalf("CreateSocialAccount: %v", err)
	}

	// Found
	got, err := db.GetSocialAccount(ctx, providerGoogle, providerUID)
	if err != nil {
		t.Fatalf("GetSocialAccount: %v", err)
	}
	if got == nil {
		t.Fatal("expected social account, got nil")
	}
	if got.ID != created.ID {
		t.Errorf("id: got %v, want %v", got.ID, created.ID)
	}
	if got.UserID != user.ID {
		t.Errorf("user_id: got %v, want %v", got.UserID, user.ID)
	}

	// Not found
	missing, err := db.GetSocialAccount(ctx, providerGoogle, "nonexistent-provider-id")
	if err != nil {
		t.Fatalf("GetSocialAccount (miss): %v", err)
	}
	if missing != nil {
		t.Error("expected nil for nonexistent social account")
	}
}

func TestGetSocialAccountsByUserID(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)
	user := testUser(t, db, org.ID)

	// Create two social accounts for the same user
	for _, provider := range []string{providerGoogle, providerGitHub} {
		_, err := db.CreateSocialAccount(ctx, &model.SocialAccount{
			UserID:         user.ID,
			Provider:       provider,
			ProviderUserID: provider + "-" + uniqueSlug(""),
			Email:          uniqueSlug("") + "@example.com",
		})
		if err != nil {
			t.Fatalf("CreateSocialAccount(%s): %v", provider, err)
		}
	}

	accounts, err := db.GetSocialAccountsByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetSocialAccountsByUserID: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(accounts))
	}
	for _, a := range accounts {
		if a.UserID != user.ID {
			t.Errorf("user_id: got %v, want %v", a.UserID, user.ID)
		}
	}

	// Empty result for unknown user
	empty, err := db.GetSocialAccountsByUserID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetSocialAccountsByUserID (empty): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 accounts for unknown user, got %d", len(empty))
	}
}

func TestUpdateSocialAccountTokens(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)
	user := testUser(t, db, org.ID)

	created, err := db.CreateSocialAccount(ctx, &model.SocialAccount{
		UserID:         user.ID,
		Provider:       providerGoogle,
		ProviderUserID: "token-test-" + uniqueSlug(""),
		Email:          "token@gmail.com",
	})
	if err != nil {
		t.Fatalf("CreateSocialAccount: %v", err)
	}

	expires := time.Now().Add(1 * time.Hour).UTC().Truncate(time.Microsecond)
	err = db.UpdateSocialAccountTokens(ctx, created.ID, "new-access-token", "new-refresh-token", &expires)
	if err != nil {
		t.Fatalf("UpdateSocialAccountTokens: %v", err)
	}

	// Verify update
	got, err := db.GetSocialAccount(ctx, providerGoogle, created.ProviderUserID)
	if err != nil {
		t.Fatalf("GetSocialAccount: %v", err)
	}
	if got.AccessToken != "new-access-token" {
		t.Errorf("access_token: got %q, want %q", got.AccessToken, "new-access-token")
	}
	if got.RefreshToken != "new-refresh-token" {
		t.Errorf("refresh_token: got %q, want %q", got.RefreshToken, "new-refresh-token")
	}
	if got.TokenExpiresAt == nil {
		t.Fatal("expected token_expires_at to be set")
	}

	// Update nonexistent
	err = db.UpdateSocialAccountTokens(ctx, uuid.New(), "x", "y", nil)
	if err == nil {
		t.Fatal("expected error updating nonexistent social account")
	}
}

func TestDeleteSocialAccount(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)
	user := testUser(t, db, org.ID)

	created, err := db.CreateSocialAccount(ctx, &model.SocialAccount{
		UserID:         user.ID,
		Provider:       providerGitHub,
		ProviderUserID: "delete-test-" + uniqueSlug(""),
		Email:          "delete@github.com",
	})
	if err != nil {
		t.Fatalf("CreateSocialAccount: %v", err)
	}

	// Delete
	err = db.DeleteSocialAccount(ctx, created.ID)
	if err != nil {
		t.Fatalf("DeleteSocialAccount: %v", err)
	}

	// Verify deleted
	gone, err := db.GetSocialAccount(ctx, providerGitHub, created.ProviderUserID)
	if err != nil {
		t.Fatalf("GetSocialAccount after delete: %v", err)
	}
	if gone != nil {
		t.Error("expected nil after delete")
	}

	// Delete nonexistent
	err = db.DeleteSocialAccount(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error deleting nonexistent social account")
	}
}
