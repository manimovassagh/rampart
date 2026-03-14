package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
)

func TestEmailVerificationTokenCreateAndConsume(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "evtoken-" + uniqueSlug(""), Email: uniqueSlug("ev") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token := "verify-token-" + uuid.New().String()

	// Create token
	err = db.CreateEmailVerificationToken(ctx, user.ID, token, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("CreateEmailVerificationToken: %v", err)
	}

	// Consume token
	userID, err := db.ConsumeEmailVerificationToken(ctx, token)
	if err != nil {
		t.Fatalf("ConsumeEmailVerificationToken: %v", err)
	}
	if userID != user.ID {
		t.Errorf("user_id: got %v, want %v", userID, user.ID)
	}

	// Consuming again should fail (already used)
	_, err = db.ConsumeEmailVerificationToken(ctx, token)
	if err == nil {
		t.Fatal("expected error consuming already-used token")
	}
}

func TestEmailVerificationTokenExpired(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "evexpired-" + uniqueSlug(""), Email: uniqueSlug("eve") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token := "expired-verify-" + uuid.New().String()

	// Create expired token
	err = db.CreateEmailVerificationToken(ctx, user.ID, token, time.Now().Add(-1*time.Minute))
	if err != nil {
		t.Fatalf("CreateEmailVerificationToken: %v", err)
	}

	// Should fail
	_, err = db.ConsumeEmailVerificationToken(ctx, token)
	if err == nil {
		t.Fatal("expected error consuming expired token")
	}
}

func TestConsumeNonexistentEmailVerificationToken(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	_, err := db.ConsumeEmailVerificationToken(ctx, "totally-fake-token")
	if err == nil {
		t.Fatal("expected error consuming nonexistent token")
	}
}

func TestCreateTokenInvalidatesPrevious(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "evinvalidate-" + uniqueSlug(""), Email: uniqueSlug("evi") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token1 := "first-verify-" + uuid.New().String()
	token2 := "second-verify-" + uuid.New().String()

	// Create first token
	err = db.CreateEmailVerificationToken(ctx, user.ID, token1, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("CreateEmailVerificationToken (1): %v", err)
	}

	// Create second token (should invalidate first)
	err = db.CreateEmailVerificationToken(ctx, user.ID, token2, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("CreateEmailVerificationToken (2): %v", err)
	}

	// First token should be invalidated
	_, err = db.ConsumeEmailVerificationToken(ctx, token1)
	if err == nil {
		t.Error("expected error consuming invalidated token")
	}

	// Second token should work
	userID, err := db.ConsumeEmailVerificationToken(ctx, token2)
	if err != nil {
		t.Fatalf("ConsumeEmailVerificationToken (2): %v", err)
	}
	if userID != user.ID {
		t.Errorf("user_id: got %v, want %v", userID, user.ID)
	}
}

func TestMarkEmailVerified(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "markverified-" + uniqueSlug(""), Email: uniqueSlug("mv") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Initially not verified
	got, _ := db.GetUserByID(ctx, user.ID)
	if got.EmailVerified {
		t.Error("expected email_verified=false initially")
	}

	// Mark verified
	err = db.MarkEmailVerified(ctx, user.ID)
	if err != nil {
		t.Fatalf("MarkEmailVerified: %v", err)
	}

	// Verify
	got, _ = db.GetUserByID(ctx, user.ID)
	if !got.EmailVerified {
		t.Error("expected email_verified=true after MarkEmailVerified")
	}
}

func TestDeleteExpiredEmailVerificationTokens(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	// Just verify it doesn't error; actual cleanup depends on existing data
	_, err := db.DeleteExpiredEmailVerificationTokens(ctx)
	if err != nil {
		t.Fatalf("DeleteExpiredEmailVerificationTokens: %v", err)
	}
}
