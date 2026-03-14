package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUserToResponse(t *testing.T) {
	now := time.Now()
	userID := uuid.New()
	orgID := uuid.New()

	user := &User{
		ID:            userID,
		OrgID:         orgID,
		Username:      "johndoe",
		Email:         "john@example.com",
		EmailVerified: true,
		GivenName:     "John",
		FamilyName:    "Doe",
		PasswordHash:  []byte("secret-hash"),
		Enabled:       true,
		MFAEnabled:    true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	resp := user.ToResponse()

	if resp.ID != userID {
		t.Errorf("ID = %v, want %v", resp.ID, userID)
	}
	if resp.OrgID != orgID {
		t.Errorf("OrgID = %v, want %v", resp.OrgID, orgID)
	}
	if resp.Username != "johndoe" {
		t.Errorf("Username = %q, want johndoe", resp.Username)
	}
	if resp.Email != "john@example.com" {
		t.Errorf("Email = %q, want john@example.com", resp.Email)
	}
	if !resp.EmailVerified {
		t.Error("EmailVerified = false, want true")
	}
	if resp.GivenName != "John" {
		t.Errorf("GivenName = %q, want John", resp.GivenName)
	}
	if resp.FamilyName != "Doe" {
		t.Errorf("FamilyName = %q, want Doe", resp.FamilyName)
	}
	if !resp.Enabled {
		t.Error("Enabled = false, want true")
	}
	if !resp.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", resp.CreatedAt, now)
	}
	if !resp.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", resp.UpdatedAt, now)
	}
}

func TestUserToAdminResponse(t *testing.T) {
	now := time.Now()
	lastLogin := now.Add(-1 * time.Hour)
	userID := uuid.New()

	user := &User{
		ID:          userID,
		Username:    "admin",
		Email:       "admin@example.com",
		Enabled:     true,
		MFAEnabled:  true,
		LastLoginAt: &lastLogin,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resp := user.ToAdminResponse(5)

	if resp.ID != userID {
		t.Errorf("ID = %v, want %v", resp.ID, userID)
	}
	if resp.Username != "admin" {
		t.Errorf("Username = %q, want admin", resp.Username)
	}
	if !resp.MFAEnabled {
		t.Error("MFAEnabled = false, want true")
	}
	if resp.SessionCount != 5 {
		t.Errorf("SessionCount = %d, want 5", resp.SessionCount)
	}
	if resp.LastLoginAt == nil || !resp.LastLoginAt.Equal(lastLogin) {
		t.Errorf("LastLoginAt = %v, want %v", resp.LastLoginAt, lastLogin)
	}
}

func TestUserToAdminResponseNilLastLogin(t *testing.T) {
	user := &User{
		ID:          uuid.New(),
		Username:    "newuser",
		LastLoginAt: nil,
	}

	resp := user.ToAdminResponse(0)

	if resp.LastLoginAt != nil {
		t.Errorf("LastLoginAt = %v, want nil", resp.LastLoginAt)
	}
	if resp.SessionCount != 0 {
		t.Errorf("SessionCount = %d, want 0", resp.SessionCount)
	}
}

func TestOrganizationToOrgResponse(t *testing.T) {
	now := time.Now()
	orgID := uuid.New()

	org := &Organization{
		ID:          orgID,
		Name:        "Acme Corp",
		Slug:        "acme",
		DisplayName: "Acme Corporation",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resp := org.ToOrgResponse(42)

	if resp.ID != orgID {
		t.Errorf("ID = %v, want %v", resp.ID, orgID)
	}
	if resp.Name != "Acme Corp" {
		t.Errorf("Name = %q, want Acme Corp", resp.Name)
	}
	if resp.Slug != "acme" {
		t.Errorf("Slug = %q, want acme", resp.Slug)
	}
	if resp.DisplayName != "Acme Corporation" {
		t.Errorf("DisplayName = %q, want Acme Corporation", resp.DisplayName)
	}
	if !resp.Enabled {
		t.Error("Enabled = false, want true")
	}
	if resp.UserCount != 42 {
		t.Errorf("UserCount = %d, want 42", resp.UserCount)
	}
	if !resp.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", resp.CreatedAt, now)
	}
}

func TestOrgSettingsToResponse(t *testing.T) {
	settingsID := uuid.New()
	orgID := uuid.New()
	now := time.Now()

	settings := &OrgSettings{
		ID:                        settingsID,
		OrgID:                     orgID,
		PasswordMinLength:         12,
		PasswordRequireUppercase:  true,
		PasswordRequireLowercase:  true,
		PasswordRequireNumbers:    true,
		PasswordRequireSymbols:    false,
		MFAEnforcement:            "optional",
		AccessTokenTTL:            15 * time.Minute,
		RefreshTokenTTL:           24 * time.Hour,
		LogoURL:                   "https://example.com/logo.png",
		PrimaryColor:              "#3b82f6",
		SelfRegistrationEnabled:   true,
		EmailVerificationRequired: false,
		LoginTheme:                "modern",
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	resp := settings.ToResponse()

	if resp.ID != settingsID {
		t.Errorf("ID = %v, want %v", resp.ID, settingsID)
	}
	if resp.OrgID != orgID {
		t.Errorf("OrgID = %v, want %v", resp.OrgID, orgID)
	}
	if resp.PasswordMinLength != 12 {
		t.Errorf("PasswordMinLength = %d, want 12", resp.PasswordMinLength)
	}
	if !resp.PasswordRequireUppercase {
		t.Error("PasswordRequireUppercase = false, want true")
	}
	if resp.MFAEnforcement != "optional" {
		t.Errorf("MFAEnforcement = %q, want optional", resp.MFAEnforcement)
	}
	if resp.AccessTokenTTLSeconds != 900 {
		t.Errorf("AccessTokenTTLSeconds = %d, want 900", resp.AccessTokenTTLSeconds)
	}
	if resp.RefreshTokenTTLSeconds != 86400 {
		t.Errorf("RefreshTokenTTLSeconds = %d, want 86400", resp.RefreshTokenTTLSeconds)
	}
	if resp.LogoURL != "https://example.com/logo.png" {
		t.Errorf("LogoURL = %q, want https://example.com/logo.png", resp.LogoURL)
	}
	if resp.PrimaryColor != "#3b82f6" {
		t.Errorf("PrimaryColor = %q, want #3b82f6", resp.PrimaryColor)
	}
	if !resp.SelfRegistrationEnabled {
		t.Error("SelfRegistrationEnabled = false, want true")
	}
	if resp.LoginTheme != "modern" {
		t.Errorf("LoginTheme = %q, want modern", resp.LoginTheme)
	}
}

func TestValidateCSSColor(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// Valid: empty (no color set)
		{"empty string", "", false},

		// Valid: hex colors
		{"hex 3-digit", "#fff", false},
		{"hex 3-digit uppercase", "#FFF", false},
		{"hex 4-digit with alpha", "#ff0a", false},
		{"hex 6-digit", "#ff5500", false},
		{"hex 6-digit uppercase", "#FF5500", false},
		{"hex 8-digit with alpha", "#ff550080", false},

		// Valid: rgb/rgba
		{"rgb", "rgb(255,0,0)", false},
		{"rgb with spaces", "rgb( 255 , 0 , 0 )", false},
		{"rgba", "rgba(255,0,0,0.5)", false},
		{"rgb percentages", "rgb(100%,0%,50%)", false},

		// Valid: hsl/hsla
		{"hsl", "hsl(120,50%,50%)", false},
		{"hsla", "hsla(120,50%,50%,0.5)", false},

		// Valid: named colors
		{"named red", "red", false},
		{"named blue", "blue", false},
		{"named transparent", "transparent", false},
		{"named rebeccapurple", "rebeccapurple", false},
		{"named mixed case", "DarkBlue", false},

		// Invalid: CSS injection vectors
		{"semicolon injection", "#fff; background: url(evil)", true},
		{"closing brace", "#fff} body{background:red", true},
		{"opening brace", "red{color:white", true},
		{"url() injection", "url(javascript:alert(1))", true},
		{"expression() injection", "expression(alert(1))", true},
		{"@import injection", "@import url(evil.css)", true},
		{"javascript: protocol", "javascript:alert(1)", true},
		{"backslash escape", `\000075rl(evil)`, true},

		// Invalid: not a color
		{"random string", "notacolor", true},
		{"number only", "12345", true},
		{"hex without hash", "ff5500", true},
		{"invalid hex length", "#ff55", false}, // 4-digit hex is valid (with alpha)
		{"hex too long", "#ff5500ff00", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCSSColor(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCSSColor(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestRoleToRoleResponse(t *testing.T) {
	now := time.Now()
	roleID := uuid.New()
	orgID := uuid.New()

	role := &Role{
		ID:          roleID,
		OrgID:       orgID,
		Name:        "admin",
		Description: "Administrator role",
		Builtin:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resp := role.ToRoleResponse(10)

	if resp.ID != roleID {
		t.Errorf("ID = %v, want %v", resp.ID, roleID)
	}
	if resp.Name != "admin" {
		t.Errorf("Name = %q, want admin", resp.Name)
	}
	if resp.Description != "Administrator role" {
		t.Errorf("Description = %q, want Administrator role", resp.Description)
	}
	if !resp.Builtin {
		t.Error("Builtin = false, want true")
	}
	if resp.UserCount != 10 {
		t.Errorf("UserCount = %d, want 10", resp.UserCount)
	}
}

func TestGroupToGroupResponse(t *testing.T) {
	now := time.Now()
	groupID := uuid.New()
	orgID := uuid.New()

	group := &Group{
		ID:          groupID,
		OrgID:       orgID,
		Name:        "engineering",
		Description: "Engineering team",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resp := group.ToGroupResponse(15, 3)

	if resp.ID != groupID {
		t.Errorf("ID = %v, want %v", resp.ID, groupID)
	}
	if resp.Name != "engineering" {
		t.Errorf("Name = %q, want engineering", resp.Name)
	}
	if resp.Description != "Engineering team" {
		t.Errorf("Description = %q, want Engineering team", resp.Description)
	}
	if resp.MemberCount != 15 {
		t.Errorf("MemberCount = %d, want 15", resp.MemberCount)
	}
	if resp.RoleCount != 3 {
		t.Errorf("RoleCount = %d, want 3", resp.RoleCount)
	}
}

func TestOAuthClientToAdminResponse(t *testing.T) {
	now := time.Now()
	orgID := uuid.New()

	client := &OAuthClient{
		ID:               "my-client-id",
		OrgID:            orgID,
		Name:             "My App",
		ClientType:       "confidential",
		RedirectURIs:     []string{"http://localhost:3000/callback"},
		ClientSecretHash: []byte("hashed-secret"),
		Description:      "Test app",
		Enabled:          true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	resp := client.ToAdminResponse()

	if resp.ID != "my-client-id" {
		t.Errorf("ID = %q, want my-client-id", resp.ID)
	}
	if resp.OrgID != orgID {
		t.Errorf("OrgID = %v, want %v", resp.OrgID, orgID)
	}
	if resp.Name != "My App" {
		t.Errorf("Name = %q, want My App", resp.Name)
	}
	if resp.ClientType != "confidential" {
		t.Errorf("ClientType = %q, want confidential", resp.ClientType)
	}
	if len(resp.RedirectURIs) != 1 || resp.RedirectURIs[0] != "http://localhost:3000/callback" {
		t.Errorf("RedirectURIs = %v, want [http://localhost:3000/callback]", resp.RedirectURIs)
	}
	if resp.Description != "Test app" {
		t.Errorf("Description = %q, want Test app", resp.Description)
	}
	if !resp.Enabled {
		t.Error("Enabled = false, want true")
	}
}
