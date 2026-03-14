package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
)

func TestPasswordResetTokenCreateAndConsume(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "pwreset-" + uniqueSlug(""), Email: uniqueSlug("pr") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token := "reset-token-" + uuid.New().String()

	// Create token
	err = db.CreatePasswordResetToken(ctx, user.ID, token, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("CreatePasswordResetToken: %v", err)
	}

	// Consume token
	userID, err := db.ConsumePasswordResetToken(ctx, token)
	if err != nil {
		t.Fatalf("ConsumePasswordResetToken: %v", err)
	}
	if userID != user.ID {
		t.Errorf("user_id: got %v, want %v", userID, user.ID)
	}

	// Consuming again should fail (already used)
	_, err = db.ConsumePasswordResetToken(ctx, token)
	if err == nil {
		t.Fatal("expected error consuming already-used token")
	}
}

func TestPasswordResetTokenExpired(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "prexpired-" + uniqueSlug(""), Email: uniqueSlug("pre") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token := "expired-reset-" + uuid.New().String()

	// Create expired token
	err = db.CreatePasswordResetToken(ctx, user.ID, token, time.Now().Add(-1*time.Minute))
	if err != nil {
		t.Fatalf("CreatePasswordResetToken: %v", err)
	}

	_, err = db.ConsumePasswordResetToken(ctx, token)
	if err == nil {
		t.Fatal("expected error consuming expired token")
	}
}

func TestConsumeNonexistentPasswordResetToken(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	_, err := db.ConsumePasswordResetToken(ctx, "totally-fake-reset-token")
	if err == nil {
		t.Fatal("expected error consuming nonexistent token")
	}
}

func TestPasswordResetTokenInvalidatesPrevious(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "prinvalidate-" + uniqueSlug(""), Email: uniqueSlug("pri") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token1 := "first-reset-" + uuid.New().String()
	token2 := "second-reset-" + uuid.New().String()

	// Create first token
	err = db.CreatePasswordResetToken(ctx, user.ID, token1, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("CreatePasswordResetToken (1): %v", err)
	}

	// Create second token (should invalidate first)
	err = db.CreatePasswordResetToken(ctx, user.ID, token2, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("CreatePasswordResetToken (2): %v", err)
	}

	// First token should be invalidated
	_, err = db.ConsumePasswordResetToken(ctx, token1)
	if err == nil {
		t.Error("expected error consuming invalidated token")
	}

	// Second token should work
	userID, err := db.ConsumePasswordResetToken(ctx, token2)
	if err != nil {
		t.Fatalf("ConsumePasswordResetToken (2): %v", err)
	}
	if userID != user.ID {
		t.Errorf("user_id: got %v, want %v", userID, user.ID)
	}
}

func TestDeleteExpiredPasswordResetTokens(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	_, err := db.DeleteExpiredPasswordResetTokens(ctx)
	if err != nil {
		t.Fatalf("DeleteExpiredPasswordResetTokens: %v", err)
	}
}
