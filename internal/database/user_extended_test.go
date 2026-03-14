package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
)

func TestFindUserByEmail(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	email := uniqueSlug("find") + "@example.com"
	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "finduser-" + uniqueSlug(""), Email: email,
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Found across all orgs
	found, err := db.FindUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if found == nil {
		t.Fatal("expected user, got nil")
	}
	if found.ID != user.ID {
		t.Errorf("id: got %v, want %v", found.ID, user.ID)
	}

	// Not found
	missing, err := db.FindUserByEmail(ctx, "nonexistent-"+uniqueSlug("")+"@example.com")
	if err != nil {
		t.Fatalf("FindUserByEmail (miss): %v", err)
	}
	if missing != nil {
		t.Error("expected nil for nonexistent email")
	}
}

func TestIncrementAndResetFailedLogins(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "failedlogin-" + uniqueSlug(""), Email: uniqueSlug("fl") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Increment below threshold
	err = db.IncrementFailedLogins(ctx, user.ID, 5, 15*time.Minute)
	if err != nil {
		t.Fatalf("IncrementFailedLogins: %v", err)
	}

	got, _ := db.GetUserByID(ctx, user.ID)
	if got.FailedLoginAttempts != 1 {
		t.Errorf("failed_login_attempts: got %d, want 1", got.FailedLoginAttempts)
	}
	if got.LockedUntil != nil {
		t.Error("expected user to not be locked yet")
	}

	// Increment more times to trigger lockout (threshold=3)
	for i := range 2 {
		err = db.IncrementFailedLogins(ctx, user.ID, 3, 15*time.Minute)
		if err != nil {
			t.Fatalf("IncrementFailedLogins[%d]: %v", i, err)
		}
	}

	got, _ = db.GetUserByID(ctx, user.ID)
	if got.FailedLoginAttempts != 3 {
		t.Errorf("failed_login_attempts: got %d, want 3", got.FailedLoginAttempts)
	}
	if got.LockedUntil == nil {
		t.Error("expected user to be locked after reaching threshold")
	}

	// Reset
	err = db.ResetFailedLogins(ctx, user.ID)
	if err != nil {
		t.Fatalf("ResetFailedLogins: %v", err)
	}

	got, _ = db.GetUserByID(ctx, user.ID)
	if got.FailedLoginAttempts != 0 {
		t.Errorf("failed_login_attempts after reset: got %d, want 0", got.FailedLoginAttempts)
	}
	if got.LockedUntil != nil {
		t.Error("expected locked_until to be nil after reset")
	}
}

func TestDuplicateUserEmail(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	email := uniqueSlug("dup") + "@example.com"
	_, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "dup1-" + uniqueSlug(""), Email: email,
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Duplicate email should fail
	_, err = db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "dup2-" + uniqueSlug(""), Email: email,
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestGetUserByIDNotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	got, err := db.GetUserByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent user")
	}
}

func TestGetUserByUsernameNotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	got, err := db.GetUserByUsername(ctx, "nonexistent-user-xyz", uuid.New())
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent user")
	}
}

func TestUpdateNonexistentUser(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	got, err := db.UpdateUser(ctx, uuid.New(), uuid.New(), &model.UpdateUserRequest{
		Username:  "ghost",
		Email:     "ghost@example.com",
		GivenName: "Ghost",
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent user")
	}
}

func TestListUsersDisabledFilter(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	// Create an enabled user
	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "disabled-" + uniqueSlug(""), Email: uniqueSlug("dis") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Disable the user
	_, err = db.UpdateUser(ctx, user.ID, org.ID, &model.UpdateUserRequest{
		Username: user.Username,
		Email:    user.Email,
		Enabled:  false,
	})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	// Filter by disabled status
	users, _, err := db.ListUsers(ctx, org.ID, "", "disabled", 100, 0)
	if err != nil {
		t.Fatalf("ListUsers disabled: %v", err)
	}
	for _, u := range users {
		if u.Enabled {
			t.Error("expected all returned users to be disabled")
		}
	}
}
