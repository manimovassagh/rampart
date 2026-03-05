package database

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
)

// testDB returns a connected DB for integration tests.
// It skips the test when RAMPART_DB_URL is not set (local dev without PostgreSQL).
func testDB(t *testing.T) *DB {
	t.Helper()
	dbURL := os.Getenv("RAMPART_DB_URL")
	if dbURL == "" {
		t.Skip("RAMPART_DB_URL not set — skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Run migrations so tables exist.
	migrationsPath := "../../migrations"
	if _, err := os.Stat(migrationsPath); err != nil {
		t.Skipf("migrations directory not found at %s: %v", migrationsPath, err)
	}
	if err := RunMigrations(dbURL, migrationsPath, testLogger()); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	return db
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

// uniqueSlug returns a slug unlikely to collide across parallel test runs.
func uniqueSlug(prefix string) string {
	return prefix + "-" + uuid.New().String()[:8]
}

// --- Organization CRUD ---

func TestOrganizationCRUD(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	tests := []struct {
		name        string
		slug        string
		displayName string
	}{
		{"Org Alpha", uniqueSlug("alpha"), "Alpha Inc."},
		{"Org Beta", uniqueSlug("beta"), "Beta Corp."},
		{"Minimal Org", uniqueSlug("min"), ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create
			created, err := db.CreateOrganization(ctx, &model.CreateOrgRequest{
				Name:        tc.name,
				Slug:        tc.slug,
				DisplayName: tc.displayName,
			})
			if err != nil {
				t.Fatalf("CreateOrganization: %v", err)
			}
			if created.ID == uuid.Nil {
				t.Fatal("expected non-nil UUID")
			}
			if created.Name != tc.name {
				t.Errorf("name: got %q, want %q", created.Name, tc.name)
			}
			if created.Slug != tc.slug {
				t.Errorf("slug: got %q, want %q", created.Slug, tc.slug)
			}
			if !created.Enabled {
				t.Error("expected new org to be enabled by default")
			}

			// Get by ID
			got, err := db.GetOrganizationByID(ctx, created.ID)
			if err != nil {
				t.Fatalf("GetOrganizationByID: %v", err)
			}
			if got == nil {
				t.Fatal("expected org, got nil")
			}
			if got.Slug != tc.slug {
				t.Errorf("slug: got %q, want %q", got.Slug, tc.slug)
			}

			// Get by slug
			idBySlug, err := db.GetOrganizationIDBySlug(ctx, tc.slug)
			if err != nil {
				t.Fatalf("GetOrganizationIDBySlug: %v", err)
			}
			if idBySlug != created.ID {
				t.Errorf("id by slug: got %v, want %v", idBySlug, created.ID)
			}

			// Update
			updated, err := db.UpdateOrganization(ctx, created.ID, &model.UpdateOrgRequest{
				Name:        tc.name + " Updated",
				DisplayName: "Updated Display",
				Enabled:     true,
			})
			if err != nil {
				t.Fatalf("UpdateOrganization: %v", err)
			}
			if updated.Name != tc.name+" Updated" {
				t.Errorf("updated name: got %q, want %q", updated.Name, tc.name+" Updated")
			}

			// List
			orgs, total, err := db.ListOrganizations(ctx, "", 100, 0)
			if err != nil {
				t.Fatalf("ListOrganizations: %v", err)
			}
			if total < 1 {
				t.Error("expected at least 1 org in list")
			}
			found := false
			for _, o := range orgs {
				if o.ID == created.ID {
					found = true
				}
			}
			if !found {
				t.Error("created org not found in list")
			}

			// Count
			count, err := db.CountOrganizations(ctx)
			if err != nil {
				t.Fatalf("CountOrganizations: %v", err)
			}
			if count < 1 {
				t.Error("expected count >= 1")
			}

			// Delete
			err = db.DeleteOrganization(ctx, created.ID)
			if err != nil {
				t.Fatalf("DeleteOrganization: %v", err)
			}

			// Verify deleted
			gone, err := db.GetOrganizationByID(ctx, created.ID)
			if err != nil {
				t.Fatalf("GetOrganizationByID after delete: %v", err)
			}
			if gone != nil {
				t.Error("expected nil after delete")
			}
		})
	}
}

func TestOrganizationListSearch(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	slug := uniqueSlug("search")
	org, err := db.CreateOrganization(ctx, &model.CreateOrgRequest{
		Name: "Searchable Org", Slug: slug, DisplayName: "Findme Corp",
	})
	if err != nil {
		t.Fatalf("CreateOrganization: %v", err)
	}
	t.Cleanup(func() { _ = db.DeleteOrganization(ctx, org.ID) })

	// Search by name
	orgs, total, err := db.ListOrganizations(ctx, "Searchable", 100, 0)
	if err != nil {
		t.Fatalf("ListOrganizations with search: %v", err)
	}
	if total < 1 {
		t.Error("expected at least 1 result for search")
	}
	found := false
	for _, o := range orgs {
		if o.ID == org.ID {
			found = true
		}
	}
	if !found {
		t.Error("expected to find org by name search")
	}

	// Search miss
	_, total, err = db.ListOrganizations(ctx, "zzz-nonexistent-zzz", 100, 0)
	if err != nil {
		t.Fatalf("ListOrganizations miss: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 results, got %d", total)
	}
}

func TestDeleteDefaultOrganizationRejected(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	defaultID, err := db.GetDefaultOrganizationID(ctx)
	if err != nil {
		t.Skipf("default org not found (may not be seeded): %v", err)
	}

	err = db.DeleteOrganization(ctx, defaultID)
	if err == nil {
		t.Fatal("expected error deleting default org")
	}
}

func TestUpdateNonexistentOrganization(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	got, err := db.UpdateOrganization(ctx, uuid.New(), &model.UpdateOrgRequest{
		Name: "ghost", DisplayName: "Ghost", Enabled: true,
	})
	if err != nil {
		t.Fatalf("UpdateOrganization: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent org")
	}
}

// --- User CRUD ---

func testOrg(t *testing.T, db *DB) *model.Organization {
	t.Helper()
	ctx := context.Background()
	org, err := db.CreateOrganization(ctx, &model.CreateOrgRequest{
		Name: "Test Org " + uniqueSlug("u"), Slug: uniqueSlug("testorg"), DisplayName: "Test",
	})
	if err != nil {
		t.Fatalf("creating test org: %v", err)
	}
	t.Cleanup(func() { _ = db.DeleteOrganization(ctx, org.ID) })
	return org
}

func TestUserCRUD(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	tests := []struct {
		username string
		email    string
		given    string
		family   string
	}{
		{"alice", "alice-" + uniqueSlug("u") + "@example.com", "Alice", "Smith"},
		{"bob", "bob-" + uniqueSlug("u") + "@example.com", "Bob", "Jones"},
		{"minimal", "min-" + uniqueSlug("u") + "@example.com", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.username, func(t *testing.T) {
			// Create
			created, err := db.CreateUser(ctx, &model.User{
				OrgID:        org.ID,
				Username:     tc.username + "-" + uniqueSlug(""),
				Email:        tc.email,
				GivenName:    tc.given,
				FamilyName:   tc.family,
				PasswordHash: []byte("$2a$10$fakehash"),
			})
			if err != nil {
				t.Fatalf("CreateUser: %v", err)
			}
			if created.ID == uuid.Nil {
				t.Fatal("expected non-nil user UUID")
			}
			if !created.Enabled {
				t.Error("expected new user to be enabled by default")
			}

			// Get by ID
			got, err := db.GetUserByID(ctx, created.ID)
			if err != nil {
				t.Fatalf("GetUserByID: %v", err)
			}
			if got == nil {
				t.Fatal("expected user, got nil")
			}
			if got.Email != tc.email {
				t.Errorf("email: got %q, want %q", got.Email, tc.email)
			}

			// Get by email
			byEmail, err := db.GetUserByEmail(ctx, tc.email, org.ID)
			if err != nil {
				t.Fatalf("GetUserByEmail: %v", err)
			}
			if byEmail == nil || byEmail.ID != created.ID {
				t.Error("GetUserByEmail returned wrong user")
			}

			// Get by username
			byName, err := db.GetUserByUsername(ctx, created.Username, org.ID)
			if err != nil {
				t.Fatalf("GetUserByUsername: %v", err)
			}
			if byName == nil || byName.ID != created.ID {
				t.Error("GetUserByUsername returned wrong user")
			}

			// Update
			updated, err := db.UpdateUser(ctx, created.ID, &model.UpdateUserRequest{
				Username:      created.Username,
				Email:         created.Email,
				GivenName:     "Updated",
				FamilyName:    "Name",
				Enabled:       true,
				EmailVerified: true,
			})
			if err != nil {
				t.Fatalf("UpdateUser: %v", err)
			}
			if updated.GivenName != "Updated" {
				t.Errorf("given_name: got %q, want %q", updated.GivenName, "Updated")
			}
			if !updated.EmailVerified {
				t.Error("expected email_verified to be true")
			}

			// Update password
			err = db.UpdatePassword(ctx, created.ID, []byte("$2a$10$newhash"))
			if err != nil {
				t.Fatalf("UpdatePassword: %v", err)
			}

			// Update last login
			err = db.UpdateLastLoginAt(ctx, created.ID)
			if err != nil {
				t.Fatalf("UpdateLastLoginAt: %v", err)
			}
			afterLogin, _ := db.GetUserByID(ctx, created.ID)
			if afterLogin.LastLoginAt == nil {
				t.Error("expected last_login_at to be set")
			}

			// Delete
			err = db.DeleteUser(ctx, created.ID)
			if err != nil {
				t.Fatalf("DeleteUser: %v", err)
			}

			// Verify deleted
			gone, err := db.GetUserByID(ctx, created.ID)
			if err != nil {
				t.Fatalf("GetUserByID after delete: %v", err)
			}
			if gone != nil {
				t.Error("expected nil after delete")
			}
		})
	}
}

func TestUserListAndCount(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	// Create a couple of users
	for i := 0; i < 3; i++ {
		slug := uniqueSlug("lu")
		_, err := db.CreateUser(ctx, &model.User{
			OrgID:        org.ID,
			Username:     "listuser-" + slug,
			Email:        slug + "@example.com",
			PasswordHash: []byte("$2a$10$fakehash"),
		})
		if err != nil {
			t.Fatalf("CreateUser[%d]: %v", i, err)
		}
	}

	// List all
	users, total, err := db.ListUsers(ctx, org.ID, "", "", 10, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if total < 3 {
		t.Errorf("expected at least 3 users, got %d", total)
	}
	if len(users) < 3 {
		t.Errorf("expected at least 3 users in page, got %d", len(users))
	}

	// List with search
	_, total, err = db.ListUsers(ctx, org.ID, "listuser", "", 10, 0)
	if err != nil {
		t.Fatalf("ListUsers with search: %v", err)
	}
	if total < 3 {
		t.Errorf("expected at least 3 matching users, got %d", total)
	}

	// List with status filter
	users, _, err = db.ListUsers(ctx, org.ID, "", "enabled", 10, 0)
	if err != nil {
		t.Fatalf("ListUsers with status: %v", err)
	}
	for _, u := range users {
		if !u.Enabled {
			t.Error("expected all returned users to be enabled")
		}
	}

	// Count
	count, err := db.CountUsers(ctx, org.ID)
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if count < 3 {
		t.Errorf("expected count >= 3, got %d", count)
	}

	// Count recent
	recent, err := db.CountRecentUsers(ctx, org.ID, 1)
	if err != nil {
		t.Fatalf("CountRecentUsers: %v", err)
	}
	if recent < 3 {
		t.Errorf("expected recent count >= 3, got %d", recent)
	}
}

func TestGetUserByEmailNotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	u, err := db.GetUserByEmail(ctx, "nonexistent@example.com", uuid.New())
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if u != nil {
		t.Error("expected nil for nonexistent user")
	}
}

func TestDeleteNonexistentUser(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	err := db.DeleteUser(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error deleting nonexistent user")
	}
}

// --- Role CRUD ---

func TestRoleCRUD(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	tests := []struct {
		name string
		desc string
	}{
		{"editor-" + uniqueSlug(""), "Can edit content"},
		{"viewer-" + uniqueSlug(""), "Read-only access"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create
			created, err := db.CreateRole(ctx, &model.Role{
				OrgID: org.ID, Name: tc.name, Description: tc.desc,
			})
			if err != nil {
				t.Fatalf("CreateRole: %v", err)
			}
			if created.ID == uuid.Nil {
				t.Fatal("expected non-nil role UUID")
			}
			if created.Builtin {
				t.Error("expected non-builtin role")
			}

			// Get by ID
			got, err := db.GetRoleByID(ctx, created.ID)
			if err != nil {
				t.Fatalf("GetRoleByID: %v", err)
			}
			if got == nil || got.Name != tc.name {
				t.Errorf("got %v, want name=%q", got, tc.name)
			}

			// Update
			updated, err := db.UpdateRole(ctx, created.ID, &model.UpdateRoleRequest{
				Name: tc.name, Description: "Updated desc",
			})
			if err != nil {
				t.Fatalf("UpdateRole: %v", err)
			}
			if updated.Description != "Updated desc" {
				t.Errorf("description: got %q, want %q", updated.Description, "Updated desc")
			}

			// List
			roles, total, err := db.ListRoles(ctx, org.ID, "", 100, 0)
			if err != nil {
				t.Fatalf("ListRoles: %v", err)
			}
			if total < 1 {
				t.Error("expected at least 1 role")
			}
			found := false
			for _, r := range roles {
				if r.ID == created.ID {
					found = true
				}
			}
			if !found {
				t.Error("created role not found in list")
			}

			// Count
			count, err := db.CountRoles(ctx, org.ID)
			if err != nil {
				t.Fatalf("CountRoles: %v", err)
			}
			if count < 1 {
				t.Error("expected count >= 1")
			}

			// Delete
			err = db.DeleteRole(ctx, created.ID)
			if err != nil {
				t.Fatalf("DeleteRole: %v", err)
			}

			gone, err := db.GetRoleByID(ctx, created.ID)
			if err != nil {
				t.Fatalf("GetRoleByID after delete: %v", err)
			}
			if gone != nil {
				t.Error("expected nil after delete")
			}
		})
	}
}

func TestRoleListSearch(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	name := "searchrole-" + uniqueSlug("")
	role, err := db.CreateRole(ctx, &model.Role{
		OrgID: org.ID, Name: name, Description: "findable",
	})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	t.Cleanup(func() { _ = db.DeleteRole(ctx, role.ID) })

	roles, total, err := db.ListRoles(ctx, org.ID, "searchrole", 100, 0)
	if err != nil {
		t.Fatalf("ListRoles search: %v", err)
	}
	if total < 1 {
		t.Error("expected at least 1 result")
	}
	found := false
	for _, r := range roles {
		if r.ID == role.ID {
			found = true
		}
	}
	if !found {
		t.Error("role not found by search")
	}
}

func TestRoleUserAssignment(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	role, err := db.CreateRole(ctx, &model.Role{
		OrgID: org.ID, Name: "assign-test-" + uniqueSlug(""), Description: "test",
	})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "assign-" + uniqueSlug(""), Email: uniqueSlug("a") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Assign
	err = db.AssignRole(ctx, user.ID, role.ID)
	if err != nil {
		t.Fatalf("AssignRole: %v", err)
	}

	// Idempotent assign
	err = db.AssignRole(ctx, user.ID, role.ID)
	if err != nil {
		t.Fatalf("AssignRole (idempotent): %v", err)
	}

	// Get user roles
	roles, err := db.GetUserRoles(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserRoles: %v", err)
	}
	if len(roles) != 1 || roles[0].ID != role.ID {
		t.Errorf("expected 1 role, got %d", len(roles))
	}

	// Get role names
	names, err := db.GetUserRoleNames(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserRoleNames: %v", err)
	}
	if len(names) != 1 {
		t.Errorf("expected 1 name, got %d", len(names))
	}

	// Count role users
	count, err := db.CountRoleUsers(ctx, role.ID)
	if err != nil {
		t.Fatalf("CountRoleUsers: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 role user, got %d", count)
	}

	// Get role users
	assignments, err := db.GetRoleUsers(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetRoleUsers: %v", err)
	}
	if len(assignments) != 1 || assignments[0].UserID != user.ID {
		t.Error("expected user in role assignments")
	}

	// Unassign
	err = db.UnassignRole(ctx, user.ID, role.ID)
	if err != nil {
		t.Fatalf("UnassignRole: %v", err)
	}

	roles, err = db.GetUserRoles(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserRoles after unassign: %v", err)
	}
	if len(roles) != 0 {
		t.Errorf("expected 0 roles after unassign, got %d", len(roles))
	}
}

// --- Group CRUD ---

func TestGroupCRUD(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	tests := []struct {
		name string
		desc string
	}{
		{"engineering-" + uniqueSlug(""), "Engineering team"},
		{"support-" + uniqueSlug(""), "Support team"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			created, err := db.CreateGroup(ctx, &model.Group{
				OrgID: org.ID, Name: tc.name, Description: tc.desc,
			})
			if err != nil {
				t.Fatalf("CreateGroup: %v", err)
			}
			if created.ID == uuid.Nil {
				t.Fatal("expected non-nil group UUID")
			}

			got, err := db.GetGroupByID(ctx, created.ID)
			if err != nil {
				t.Fatalf("GetGroupByID: %v", err)
			}
			if got == nil || got.Name != tc.name {
				t.Errorf("got %v, want name=%q", got, tc.name)
			}

			updated, err := db.UpdateGroup(ctx, created.ID, &model.UpdateGroupRequest{
				Name: tc.name, Description: "Updated",
			})
			if err != nil {
				t.Fatalf("UpdateGroup: %v", err)
			}
			if updated.Description != "Updated" {
				t.Errorf("description: got %q, want %q", updated.Description, "Updated")
			}

			groups, total, err := db.ListGroups(ctx, org.ID, "", 100, 0)
			if err != nil {
				t.Fatalf("ListGroups: %v", err)
			}
			if total < 1 {
				t.Error("expected at least 1 group")
			}
			found := false
			for _, g := range groups {
				if g.ID == created.ID {
					found = true
				}
			}
			if !found {
				t.Error("created group not in list")
			}

			count, err := db.CountGroups(ctx, org.ID)
			if err != nil {
				t.Fatalf("CountGroups: %v", err)
			}
			if count < 1 {
				t.Error("expected count >= 1")
			}

			err = db.DeleteGroup(ctx, created.ID)
			if err != nil {
				t.Fatalf("DeleteGroup: %v", err)
			}

			gone, err := db.GetGroupByID(ctx, created.ID)
			if err != nil {
				t.Fatalf("GetGroupByID after delete: %v", err)
			}
			if gone != nil {
				t.Error("expected nil after delete")
			}
		})
	}
}

func TestGroupMembership(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	group, err := db.CreateGroup(ctx, &model.Group{
		OrgID: org.ID, Name: "membership-" + uniqueSlug(""), Description: "test",
	})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "member-" + uniqueSlug(""), Email: uniqueSlug("m") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Add
	err = db.AddUserToGroup(ctx, user.ID, group.ID)
	if err != nil {
		t.Fatalf("AddUserToGroup: %v", err)
	}

	// Idempotent add
	err = db.AddUserToGroup(ctx, user.ID, group.ID)
	if err != nil {
		t.Fatalf("AddUserToGroup (idempotent): %v", err)
	}

	// Get members
	members, err := db.GetGroupMembers(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetGroupMembers: %v", err)
	}
	if len(members) != 1 || members[0].UserID != user.ID {
		t.Error("expected user in group members")
	}

	// Count members
	count, err := db.CountGroupMembers(ctx, group.ID)
	if err != nil {
		t.Fatalf("CountGroupMembers: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 member, got %d", count)
	}

	// Get user groups
	userGroups, err := db.GetUserGroups(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserGroups: %v", err)
	}
	if len(userGroups) != 1 || userGroups[0].ID != group.ID {
		t.Error("expected group in user groups")
	}

	// Remove
	err = db.RemoveUserFromGroup(ctx, user.ID, group.ID)
	if err != nil {
		t.Fatalf("RemoveUserFromGroup: %v", err)
	}

	members, err = db.GetGroupMembers(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetGroupMembers after remove: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("expected 0 members after remove, got %d", len(members))
	}
}

func TestGroupRoleAssignment(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	group, err := db.CreateGroup(ctx, &model.Group{
		OrgID: org.ID, Name: "grole-" + uniqueSlug(""), Description: "test",
	})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	role, err := db.CreateRole(ctx, &model.Role{
		OrgID: org.ID, Name: "grole-role-" + uniqueSlug(""), Description: "test",
	})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	// Assign role to group
	err = db.AssignRoleToGroup(ctx, group.ID, role.ID)
	if err != nil {
		t.Fatalf("AssignRoleToGroup: %v", err)
	}

	// Idempotent
	err = db.AssignRoleToGroup(ctx, group.ID, role.ID)
	if err != nil {
		t.Fatalf("AssignRoleToGroup (idempotent): %v", err)
	}

	// Get group roles
	groupRoles, err := db.GetGroupRoles(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetGroupRoles: %v", err)
	}
	if len(groupRoles) != 1 || groupRoles[0].RoleID != role.ID {
		t.Error("expected role in group roles")
	}

	// Count group roles
	count, err := db.CountGroupRoles(ctx, group.ID)
	if err != nil {
		t.Fatalf("CountGroupRoles: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 group role, got %d", count)
	}

	// Unassign
	err = db.UnassignRoleFromGroup(ctx, group.ID, role.ID)
	if err != nil {
		t.Fatalf("UnassignRoleFromGroup: %v", err)
	}

	groupRoles, err = db.GetGroupRoles(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetGroupRoles after unassign: %v", err)
	}
	if len(groupRoles) != 0 {
		t.Errorf("expected 0 group roles, got %d", len(groupRoles))
	}
}

func TestEffectiveUserRoles(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	// Create a user, two roles, a group
	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "effective-" + uniqueSlug(""), Email: uniqueSlug("e") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	directRole, err := db.CreateRole(ctx, &model.Role{
		OrgID: org.ID, Name: "direct-" + uniqueSlug(""), Description: "direct",
	})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	groupRole, err := db.CreateRole(ctx, &model.Role{
		OrgID: org.ID, Name: "group-" + uniqueSlug(""), Description: "via group",
	})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	group, err := db.CreateGroup(ctx, &model.Group{
		OrgID: org.ID, Name: "effective-grp-" + uniqueSlug(""), Description: "test",
	})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	// Assign direct role to user
	if err := db.AssignRole(ctx, user.ID, directRole.ID); err != nil {
		t.Fatalf("AssignRole: %v", err)
	}

	// Assign role to group, add user to group
	if err := db.AssignRoleToGroup(ctx, group.ID, groupRole.ID); err != nil {
		t.Fatalf("AssignRoleToGroup: %v", err)
	}
	if err := db.AddUserToGroup(ctx, user.ID, group.ID); err != nil {
		t.Fatalf("AddUserToGroup: %v", err)
	}

	// Get effective roles
	effective, err := db.GetEffectiveUserRoles(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetEffectiveUserRoles: %v", err)
	}
	if len(effective) < 2 {
		t.Errorf("expected at least 2 effective roles, got %d: %v", len(effective), effective)
	}

	foundDirect, foundGroup := false, false
	for _, name := range effective {
		if name == directRole.Name {
			foundDirect = true
		}
		if name == groupRole.Name {
			foundGroup = true
		}
	}
	if !foundDirect {
		t.Error("direct role not in effective roles")
	}
	if !foundGroup {
		t.Error("group-inherited role not in effective roles")
	}
}

// --- OAuth Client CRUD ---

func TestOAuthClientCRUD(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	tests := []struct {
		name       string
		clientType string
		uris       []string
	}{
		{"Public App", "public", []string{"http://localhost:3000/callback"}},
		{"Confidential App", "confidential", []string{"https://app.example.com/callback", "https://app.example.com/silent"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			created, err := db.CreateOAuthClient(ctx, &model.OAuthClient{
				OrgID:        org.ID,
				Name:         tc.name + "-" + uniqueSlug(""),
				ClientType:   tc.clientType,
				RedirectURIs: tc.uris,
				Description:  "test client",
				Enabled:      true,
			})
			if err != nil {
				t.Fatalf("CreateOAuthClient: %v", err)
			}
			if created.ID == "" {
				t.Fatal("expected non-empty client ID")
			}

			// Get
			got, err := db.GetOAuthClient(ctx, created.ID)
			if err != nil {
				t.Fatalf("GetOAuthClient: %v", err)
			}
			if got == nil {
				t.Fatal("expected client, got nil")
			}
			if got.ClientType != tc.clientType {
				t.Errorf("client_type: got %q, want %q", got.ClientType, tc.clientType)
			}
			if len(got.RedirectURIs) != len(tc.uris) {
				t.Errorf("redirect_uris: got %d, want %d", len(got.RedirectURIs), len(tc.uris))
			}

			// Update
			updated, err := db.UpdateOAuthClient(ctx, created.ID, &model.UpdateClientRequest{
				Name:         created.Name,
				Description:  "updated desc",
				RedirectURIs: "https://new.example.com/cb",
				Enabled:      true,
			})
			if err != nil {
				t.Fatalf("UpdateOAuthClient: %v", err)
			}
			if updated.Description != "updated desc" {
				t.Errorf("description: got %q, want %q", updated.Description, "updated desc")
			}

			// List
			clients, total, err := db.ListOAuthClients(ctx, org.ID, "", 100, 0)
			if err != nil {
				t.Fatalf("ListOAuthClients: %v", err)
			}
			if total < 1 {
				t.Error("expected at least 1 client")
			}
			found := false
			for _, c := range clients {
				if c.ID == created.ID {
					found = true
				}
			}
			if !found {
				t.Error("created client not in list")
			}

			// Count
			count, err := db.CountOAuthClients(ctx, org.ID)
			if err != nil {
				t.Fatalf("CountOAuthClients: %v", err)
			}
			if count < 1 {
				t.Error("expected count >= 1")
			}

			// Update secret
			err = db.UpdateClientSecret(ctx, created.ID, []byte("$2a$10$fakesecret"))
			if err != nil {
				t.Fatalf("UpdateClientSecret: %v", err)
			}

			// Delete
			err = db.DeleteOAuthClient(ctx, created.ID)
			if err != nil {
				t.Fatalf("DeleteOAuthClient: %v", err)
			}

			gone, err := db.GetOAuthClient(ctx, created.ID)
			if err != nil {
				t.Fatalf("GetOAuthClient after delete: %v", err)
			}
			if gone != nil {
				t.Error("expected nil after delete")
			}
		})
	}
}

func TestOAuthClientListSearch(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	name := "searchclient-" + uniqueSlug("")
	client, err := db.CreateOAuthClient(ctx, &model.OAuthClient{
		OrgID: org.ID, Name: name, ClientType: "public",
		RedirectURIs: []string{"http://localhost/cb"}, Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreateOAuthClient: %v", err)
	}
	t.Cleanup(func() { _ = db.DeleteOAuthClient(ctx, client.ID) })

	clients, total, err := db.ListOAuthClients(ctx, org.ID, "searchclient", 100, 0)
	if err != nil {
		t.Fatalf("ListOAuthClients search: %v", err)
	}
	if total < 1 {
		t.Error("expected at least 1 result")
	}
	found := false
	for _, c := range clients {
		if c.ID == client.ID {
			found = true
		}
	}
	if !found {
		t.Error("client not found by search")
	}
}

func TestDeleteNonexistentOAuthClient(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	err := db.DeleteOAuthClient(ctx, "nonexistent-client-id-12345")
	if err == nil {
		t.Fatal("expected error deleting nonexistent client")
	}
}

// --- Authorization Code ---

func TestAuthorizationCodeStoreAndConsume(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	// Need a user and client
	user, err := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "authcode-" + uniqueSlug(""), Email: uniqueSlug("ac") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	client, err := db.CreateOAuthClient(ctx, &model.OAuthClient{
		OrgID: org.ID, Name: "authcode-client-" + uniqueSlug(""), ClientType: "public",
		RedirectURIs: []string{"http://localhost/cb"}, Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreateOAuthClient: %v", err)
	}

	code := "test-code-" + uuid.New().String()

	// Store
	err = db.StoreAuthorizationCode(ctx, code, client.ID, user.ID, org.ID,
		"http://localhost/cb", "challenge123", "openid", time.Now().Add(5*time.Minute))
	if err != nil {
		t.Fatalf("StoreAuthorizationCode: %v", err)
	}

	// Consume
	ac, err := db.ConsumeAuthorizationCode(ctx, code)
	if err != nil {
		t.Fatalf("ConsumeAuthorizationCode: %v", err)
	}
	if ac == nil {
		t.Fatal("expected authorization code, got nil")
	}
	if ac.ClientID != client.ID {
		t.Errorf("client_id: got %q, want %q", ac.ClientID, client.ID)
	}
	if ac.Scope != "openid" {
		t.Errorf("scope: got %q, want %q", ac.Scope, "openid")
	}
	if !ac.Used {
		t.Error("expected used=true after consume")
	}

	// Consume again should return nil (already used)
	ac2, err := db.ConsumeAuthorizationCode(ctx, code)
	if err != nil {
		t.Fatalf("ConsumeAuthorizationCode (second): %v", err)
	}
	if ac2 != nil {
		t.Error("expected nil for already-consumed code")
	}
}

func TestConsumeExpiredCode(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	user, _ := db.CreateUser(ctx, &model.User{
		OrgID: org.ID, Username: "expired-" + uniqueSlug(""), Email: uniqueSlug("ex") + "@example.com",
		PasswordHash: []byte("$2a$10$fakehash"),
	})
	client, _ := db.CreateOAuthClient(ctx, &model.OAuthClient{
		OrgID: org.ID, Name: "expired-client-" + uniqueSlug(""), ClientType: "public",
		RedirectURIs: []string{"http://localhost/cb"}, Enabled: true,
	})

	code := "expired-code-" + uuid.New().String()
	err := db.StoreAuthorizationCode(ctx, code, client.ID, user.ID, org.ID,
		"http://localhost/cb", "", "openid", time.Now().Add(-1*time.Minute))
	if err != nil {
		t.Fatalf("StoreAuthorizationCode: %v", err)
	}

	ac, err := db.ConsumeAuthorizationCode(ctx, code)
	if err != nil {
		t.Fatalf("ConsumeAuthorizationCode: %v", err)
	}
	if ac != nil {
		t.Error("expected nil for expired code")
	}
}

func TestConsumeNonexistentCode(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	ac, err := db.ConsumeAuthorizationCode(ctx, "totally-fake-code")
	if err != nil {
		t.Fatalf("ConsumeAuthorizationCode: %v", err)
	}
	if ac != nil {
		t.Error("expected nil for nonexistent code")
	}
}

// --- Audit Events ---

func TestAuditEventCreateAndList(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	actorID := uuid.New()
	event := &model.AuditEvent{
		OrgID:      org.ID,
		EventType:  "user.login",
		ActorID:    &actorID,
		ActorName:  "testuser",
		TargetType: "user",
		TargetID:   uuid.New().String(),
		TargetName: "testuser",
		IPAddress:  "127.0.0.1",
		UserAgent:  "test-agent",
		Details:    map[string]any{"method": "password"},
	}

	err := db.CreateAuditEvent(ctx, event)
	if err != nil {
		t.Fatalf("CreateAuditEvent: %v", err)
	}

	// List
	events, total, err := db.ListAuditEvents(ctx, org.ID, "", "", 10, 0)
	if err != nil {
		t.Fatalf("ListAuditEvents: %v", err)
	}
	if total < 1 {
		t.Error("expected at least 1 event")
	}
	if len(events) < 1 {
		t.Fatal("expected events in page")
	}

	// Filter by type
	_, total, err = db.ListAuditEvents(ctx, org.ID, "user.login", "", 10, 0)
	if err != nil {
		t.Fatalf("ListAuditEvents filter: %v", err)
	}
	if total < 1 {
		t.Error("expected at least 1 login event")
	}

	// Search
	_, total, err = db.ListAuditEvents(ctx, org.ID, "", "testuser", 10, 0)
	if err != nil {
		t.Fatalf("ListAuditEvents search: %v", err)
	}
	if total < 1 {
		t.Error("expected at least 1 result for search")
	}

	// Count recent
	count, err := db.CountRecentEvents(ctx, org.ID, 1)
	if err != nil {
		t.Fatalf("CountRecentEvents: %v", err)
	}
	if count < 1 {
		t.Error("expected at least 1 recent event")
	}
}

// --- Organization Settings ---

func TestOrgSettingsCRUD(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db) // CreateOrganization also inserts default settings

	// Get default settings
	settings, err := db.GetOrgSettings(ctx, org.ID)
	if err != nil {
		t.Fatalf("GetOrgSettings: %v", err)
	}
	if settings == nil {
		t.Fatal("expected default settings, got nil")
	}
	if settings.OrgID != org.ID {
		t.Errorf("org_id: got %v, want %v", settings.OrgID, org.ID)
	}

	// Update
	updated, err := db.UpdateOrgSettings(ctx, org.ID, &model.UpdateOrgSettingsRequest{
		PasswordMinLength:         16,
		PasswordRequireUppercase:  true,
		PasswordRequireLowercase:  true,
		PasswordRequireNumbers:    true,
		PasswordRequireSymbols:    true,
		MFAEnforcement:            "required",
		AccessTokenTTLSeconds:     1800,
		RefreshTokenTTLSeconds:    43200,
		SelfRegistrationEnabled:   false,
		EmailVerificationRequired: true,
		ForgotPasswordEnabled:     true,
		RememberMeEnabled:         true,
		LoginPageTitle:            "Secure Login",
		LoginPageMessage:          "Enter credentials",
		LoginTheme:                "midnight",
	})
	if err != nil {
		t.Fatalf("UpdateOrgSettings: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated settings, got nil")
	}
	if updated.PasswordMinLength != 16 {
		t.Errorf("password_min_length: got %d, want 16", updated.PasswordMinLength)
	}
	if updated.MFAEnforcement != "required" {
		t.Errorf("mfa_enforcement: got %q, want %q", updated.MFAEnforcement, "required")
	}
	if updated.LoginTheme != "midnight" {
		t.Errorf("login_theme: got %q, want %q", updated.LoginTheme, "midnight")
	}

	// Read back
	reread, err := db.GetOrgSettings(ctx, org.ID)
	if err != nil {
		t.Fatalf("GetOrgSettings after update: %v", err)
	}
	if reread.PasswordMinLength != 16 {
		t.Errorf("re-read password_min_length: got %d, want 16", reread.PasswordMinLength)
	}
	if int(reread.AccessTokenTTL.Seconds()) != 1800 {
		t.Errorf("access_token_ttl: got %v, want 1800s", reread.AccessTokenTTL)
	}
}

func TestGetOrgSettingsNotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	settings, err := db.GetOrgSettings(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetOrgSettings: %v", err)
	}
	if settings != nil {
		t.Error("expected nil for nonexistent org settings")
	}
}

// --- Export/Import roundtrip (integration) ---

func TestExportImportRoundtrip(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	// Create roles, groups, clients
	role, err := db.CreateRole(ctx, &model.Role{
		OrgID: org.ID, Name: "export-role-" + uniqueSlug(""), Description: "export test",
	})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	group, err := db.CreateGroup(ctx, &model.Group{
		OrgID: org.ID, Name: "export-group-" + uniqueSlug(""), Description: "export test",
	})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	if err := db.AssignRoleToGroup(ctx, group.ID, role.ID); err != nil {
		t.Fatalf("AssignRoleToGroup: %v", err)
	}

	_, err = db.CreateOAuthClient(ctx, &model.OAuthClient{
		OrgID: org.ID, Name: "export-client-" + uniqueSlug(""), ClientType: "public",
		RedirectURIs: []string{"http://localhost/cb"}, Description: "test", Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreateOAuthClient: %v", err)
	}

	// Export
	export, err := db.ExportOrganization(ctx, org.ID)
	if err != nil {
		t.Fatalf("ExportOrganization: %v", err)
	}
	if export == nil {
		t.Fatal("expected export, got nil")
	}
	if export.Organization.Slug != org.Slug {
		t.Errorf("export slug: got %q, want %q", export.Organization.Slug, org.Slug)
	}
	if len(export.Roles) < 1 {
		t.Error("expected at least 1 role in export")
	}
	if len(export.Groups) < 1 {
		t.Error("expected at least 1 group in export")
	}
	if len(export.Clients) < 1 {
		t.Error("expected at least 1 client in export")
	}
	if export.Settings == nil {
		t.Error("expected settings in export")
	}

	// Verify group has role reference
	foundGroupWithRole := false
	for _, g := range export.Groups {
		if g.Name == group.Name && len(g.Roles) > 0 {
			foundGroupWithRole = true
			if g.Roles[0] != role.Name {
				t.Errorf("group role: got %q, want %q", g.Roles[0], role.Name)
			}
		}
	}
	if !foundGroupWithRole {
		t.Error("expected group with role in export")
	}

	// Import into a new slug
	export.Organization.Slug = uniqueSlug("imported")
	export.Organization.Name = "Imported Org"
	err = db.ImportOrganization(ctx, export)
	if err != nil {
		t.Fatalf("ImportOrganization: %v", err)
	}

	// Verify imported org exists
	importedID, err := db.GetOrganizationIDBySlug(ctx, export.Organization.Slug)
	if err != nil {
		t.Fatalf("GetOrganizationIDBySlug for imported: %v", err)
	}
	if importedID == uuid.Nil {
		t.Fatal("expected non-nil imported org ID")
	}

	// Cleanup imported org
	_ = db.DeleteOrganization(ctx, importedID)
}

func TestExportNonexistentOrganization(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	_, err := db.ExportOrganization(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error exporting nonexistent org")
	}
}
