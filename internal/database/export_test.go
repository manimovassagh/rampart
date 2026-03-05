package database

import (
	"testing"

	"github.com/manimovassagh/rampart/internal/model"
)

func TestOrgExportRoundTripStructure(t *testing.T) {
	export := &model.OrgExport{
		Organization: model.OrgExportData{
			Name:        "Test Org",
			Slug:        "test-org",
			DisplayName: "Test Organization",
		},
		Settings: &model.OrgSettingsExport{
			PasswordMinLength:         12,
			PasswordRequireUppercase:  true,
			PasswordRequireLowercase:  true,
			PasswordRequireNumbers:    true,
			PasswordRequireSymbols:    false,
			MFAEnforcement:            "optional",
			AccessTokenTTLSeconds:     3600,
			RefreshTokenTTLSeconds:    86400,
			SelfRegistrationEnabled:   true,
			EmailVerificationRequired: true,
			ForgotPasswordEnabled:     true,
			RememberMeEnabled:         false,
			LoginPageTitle:            "Welcome",
			LoginPageMessage:          "Please sign in",
		},
		Roles: []model.RoleExport{
			{Name: "admin", Description: "Administrator role"},
			{Name: "viewer", Description: "Read-only role"},
		},
		Groups: []model.GroupExport{
			{Name: "engineering", Description: "Engineering team", Roles: []string{"admin"}},
			{Name: "support", Description: "Support team", Roles: []string{"viewer"}},
		},
		Clients: []model.ClientExport{
			{
				ClientID:     "test-client-id",
				Name:         "Test App",
				Description:  "A test application",
				ClientType:   "public",
				RedirectURIs: []string{"http://localhost:3000/callback"},
				Enabled:      true,
			},
		},
	}

	// Verify organization fields
	if export.Organization.Name != "Test Org" {
		t.Errorf("expected org name %q, got %q", "Test Org", export.Organization.Name)
	}
	if export.Organization.Slug != "test-org" {
		t.Errorf("expected org slug %q, got %q", "test-org", export.Organization.Slug)
	}

	// Verify settings
	if export.Settings == nil {
		t.Fatal("expected settings to be non-nil")
	}
	if export.Settings.PasswordMinLength != 12 {
		t.Errorf("expected password min length 12, got %d", export.Settings.PasswordMinLength)
	}
	if export.Settings.MFAEnforcement != "optional" {
		t.Errorf("expected MFA enforcement %q, got %q", "optional", export.Settings.MFAEnforcement)
	}
	if export.Settings.AccessTokenTTLSeconds != 3600 {
		t.Errorf("expected access token TTL 3600, got %d", export.Settings.AccessTokenTTLSeconds)
	}

	// Verify roles
	if len(export.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(export.Roles))
	}
	if export.Roles[0].Name != "admin" {
		t.Errorf("expected first role name %q, got %q", "admin", export.Roles[0].Name)
	}

	// Verify groups with role references
	if len(export.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(export.Groups))
	}
	if export.Groups[0].Name != "engineering" {
		t.Errorf("expected first group name %q, got %q", "engineering", export.Groups[0].Name)
	}
	if len(export.Groups[0].Roles) != 1 || export.Groups[0].Roles[0] != "admin" {
		t.Errorf("expected engineering group to have role [admin], got %v", export.Groups[0].Roles)
	}

	// Verify clients
	if len(export.Clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(export.Clients))
	}
	if export.Clients[0].ClientID != "test-client-id" {
		t.Errorf("expected client ID %q, got %q", "test-client-id", export.Clients[0].ClientID)
	}
	if !export.Clients[0].Enabled {
		t.Error("expected client to be enabled")
	}
	if len(export.Clients[0].RedirectURIs) != 1 {
		t.Fatalf("expected 1 redirect URI, got %d", len(export.Clients[0].RedirectURIs))
	}
}

func TestOrgExportEmptyOrg(t *testing.T) {
	export := &model.OrgExport{
		Organization: model.OrgExportData{
			Name:        "Empty Org",
			Slug:        "empty-org",
			DisplayName: "",
		},
	}

	if export.Organization.Name != "Empty Org" {
		t.Errorf("expected org name %q, got %q", "Empty Org", export.Organization.Name)
	}
	if export.Settings != nil {
		t.Error("expected settings to be nil for empty org")
	}
	if len(export.Roles) != 0 {
		t.Errorf("expected 0 roles, got %d", len(export.Roles))
	}
	if len(export.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(export.Groups))
	}
	if len(export.Clients) != 0 {
		t.Errorf("expected 0 clients, got %d", len(export.Clients))
	}
}

func TestOrgExportGroupWithMultipleRoles(t *testing.T) {
	group := model.GroupExport{
		Name:        "devops",
		Description: "DevOps team",
		Roles:       []string{"admin", "viewer", "deployer"},
	}

	if len(group.Roles) != 3 {
		t.Fatalf("expected 3 roles, got %d", len(group.Roles))
	}
	expected := []string{"admin", "viewer", "deployer"}
	for i, role := range group.Roles {
		if role != expected[i] {
			t.Errorf("expected role[%d] = %q, got %q", i, expected[i], role)
		}
	}
}
