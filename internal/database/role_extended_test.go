package database

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/store"
)

func TestCreateRoleDuplicate(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	roleName := "dup-role-" + uniqueSlug("")

	_, err := db.CreateRole(ctx, &model.Role{
		OrgID: org.ID, Name: roleName, Description: "first",
	})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	// Duplicate should return ErrDuplicateKey
	_, err = db.CreateRole(ctx, &model.Role{
		OrgID: org.ID, Name: roleName, Description: "duplicate",
	})
	if err == nil {
		t.Fatal("expected error for duplicate role name")
	}
	if !containsError(err, store.ErrDuplicateKey) {
		t.Errorf("expected ErrDuplicateKey, got: %v", err)
	}
}

func TestGetRoleByIDNotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	got, err := db.GetRoleByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetRoleByID: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent role")
	}
}

func TestDeleteRoleNotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	err := db.DeleteRole(ctx, uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error deleting nonexistent role")
	}
	if !containsError(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestUpdateRoleNotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	got, err := db.UpdateRole(ctx, uuid.New(), uuid.New(), &model.UpdateRoleRequest{
		Name: "ghost", Description: "Ghost",
	})
	if err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent role")
	}
}

func TestUserCountsByRole(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	role, err := db.CreateRole(ctx, &model.Role{
		OrgID: org.ID, Name: "counttest-" + uniqueSlug(""), Description: "count test",
	})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "countuser-" + uniqueSlug(""), Email: uniqueSlug("cu") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	err = db.AssignRole(ctx, user.ID, role.ID)
	if err != nil {
		t.Fatalf("AssignRole: %v", err)
	}

	counts, err := db.UserCountsByRole(ctx, org.ID)
	if err != nil {
		t.Fatalf("UserCountsByRole: %v", err)
	}
	if len(counts) < 1 {
		t.Fatal("expected at least 1 role count")
	}

	found := false
	for _, rc := range counts {
		if rc.Role == role.Name && rc.Count >= 1 {
			found = true
		}
	}
	if !found {
		t.Error("expected role to have count >= 1")
	}
}

func TestCreateOrganizationDuplicate(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	slug := uniqueSlug("duporg")
	_, err := db.CreateOrganization(ctx, &model.CreateOrgRequest{
		Name: "Dup Org", Slug: slug, DisplayName: "Dup",
	})
	if err != nil {
		t.Fatalf("CreateOrganization: %v", err)
	}

	// Duplicate slug should return ErrDuplicateKey
	_, err = db.CreateOrganization(ctx, &model.CreateOrgRequest{
		Name: "Dup Org 2", Slug: slug, DisplayName: "Dup 2",
	})
	if err == nil {
		t.Fatal("expected error for duplicate org slug")
	}
	if !containsError(err, store.ErrDuplicateKey) {
		t.Errorf("expected ErrDuplicateKey, got: %v", err)
	}
}

func TestDeleteNonexistentOrganization(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	err := db.DeleteOrganization(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error deleting nonexistent org")
	}
	if !containsError(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetOrganizationByIDNotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	got, err := db.GetOrganizationByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetOrganizationByID: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent org")
	}
}

func TestCreateGroupDuplicate(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	groupName := "dup-group-" + uniqueSlug("")

	_, err := db.CreateGroup(ctx, &model.Group{
		OrgID: org.ID, Name: groupName, Description: "first",
	})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	_, err = db.CreateGroup(ctx, &model.Group{
		OrgID: org.ID, Name: groupName, Description: "duplicate",
	})
	if err == nil {
		t.Fatal("expected error for duplicate group name")
	}
	if !containsError(err, store.ErrDuplicateKey) {
		t.Errorf("expected ErrDuplicateKey, got: %v", err)
	}
}

func TestGetGroupByIDNotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	got, err := db.GetGroupByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetGroupByID: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent group")
	}
}

func TestDeleteNonexistentGroup(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	err := db.DeleteGroup(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error deleting nonexistent group")
	}
}

func TestUpdateGroupNotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	got, err := db.UpdateGroup(ctx, uuid.New(), &model.UpdateGroupRequest{
		Name: "ghost", Description: "Ghost",
	})
	if err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent group")
	}
}

// containsError checks if err wraps target using errors.Is-like string matching,
// since some errors are wrapped with fmt.Errorf.
func containsError(err, target error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, target)
}
