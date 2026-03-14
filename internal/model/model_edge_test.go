package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// 1. User model edge cases
// ---------------------------------------------------------------------------

func TestUserToResponse_EmptyUsername(t *testing.T) {
	user := &User{ID: uuid.New(), OrgID: uuid.New(), Username: ""}
	resp := user.ToResponse()
	if resp.Username != "" {
		t.Errorf("Username = %q, want empty", resp.Username)
	}
}

func TestUserToResponse_VeryLongUsername(t *testing.T) {
	long := strings.Repeat("a", 1000)
	user := &User{ID: uuid.New(), OrgID: uuid.New(), Username: long}
	resp := user.ToResponse()
	if resp.Username != long {
		t.Errorf("Username length = %d, want 1000", len(resp.Username))
	}
}

func TestUserToResponse_UnicodeUsername(t *testing.T) {
	unicodeNames := []string{
		"\u4e16\u754c",                               // Chinese characters
		"\u00e9\u00e0\u00fc",                         // accented Latin
		"\U0001f600\U0001f680",                       // emoji
		"\u0627\u0644\u0639\u0631\u0628\u064a\u0629", // Arabic
	}
	for _, name := range unicodeNames {
		user := &User{ID: uuid.New(), Username: name}
		resp := user.ToResponse()
		if resp.Username != name {
			t.Errorf("Username = %q, want %q", resp.Username, name)
		}
	}
}

func TestUserToResponse_SQLInjectionUsername(t *testing.T) {
	injections := []string{
		"'; DROP TABLE users; --",
		"1' OR '1'='1",
		"admin'--",
		"' UNION SELECT * FROM users --",
	}
	for _, inj := range injections {
		user := &User{ID: uuid.New(), Username: inj}
		resp := user.ToResponse()
		if resp.Username != inj {
			t.Errorf("Username = %q, want %q (SQL injection preserved as-is)", resp.Username, inj)
		}
	}
}

func TestUserToResponse_XSSInEmail(t *testing.T) {
	xssEmails := []string{
		`<script>alert('xss')</script>@evil.com`,
		`"><img src=x onerror=alert(1)>@evil.com`,
		`javascript:alert(1)@evil.com`,
		`user+<svg/onload=alert(1)>@evil.com`,
	}
	for _, email := range xssEmails {
		user := &User{ID: uuid.New(), Email: email}
		resp := user.ToResponse()
		// The model layer stores data as-is; sanitization is at the handler layer.
		if resp.Email != email {
			t.Errorf("Email = %q, want %q", resp.Email, email)
		}
	}
}

func TestUserToResponse_PasswordHashStripped(t *testing.T) {
	user := &User{
		ID:           uuid.New(),
		Username:     "testuser",
		PasswordHash: []byte("super-secret-hash-value"),
	}
	resp := user.ToResponse()

	// Verify the response type has no PasswordHash field by marshaling to JSON.
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if strings.Contains(string(data), "super-secret-hash-value") {
		t.Error("UserResponse JSON contains password hash — sensitive data leak")
	}
	if strings.Contains(string(data), "password_hash") {
		t.Error("UserResponse JSON contains password_hash key")
	}
}

// ---------------------------------------------------------------------------
// 2. OAuthClient model edge cases
// ---------------------------------------------------------------------------

func TestOAuthClient_EmptyRedirectURIs(t *testing.T) {
	client := &OAuthClient{
		ID:           "client-1",
		RedirectURIs: []string{},
	}
	resp := client.ToAdminResponse()
	if len(resp.RedirectURIs) != 0 {
		t.Errorf("RedirectURIs length = %d, want 0", len(resp.RedirectURIs))
	}
}

func TestOAuthClient_NilRedirectURIs(t *testing.T) {
	client := &OAuthClient{
		ID:           "client-2",
		RedirectURIs: nil,
	}
	resp := client.ToAdminResponse()
	if resp.RedirectURIs != nil {
		t.Errorf("RedirectURIs = %v, want nil", resp.RedirectURIs)
	}
}

func TestOAuthClient_DangerousRedirectURIs(t *testing.T) {
	// The model stores URIs as-is; validation should happen at handler level.
	// These tests verify data integrity through the model layer.
	dangerousURIs := []string{
		"javascript:alert(document.cookie)",
		"data:text/html,<script>alert(1)</script>",
		"https://*.example.com/callback",
		"http://evil.com/../../../etc/passwd",
		"",
	}
	client := &OAuthClient{
		ID:           "client-3",
		RedirectURIs: dangerousURIs,
	}
	resp := client.ToAdminResponse()
	if len(resp.RedirectURIs) != len(dangerousURIs) {
		t.Fatalf("RedirectURIs length = %d, want %d", len(resp.RedirectURIs), len(dangerousURIs))
	}
	for i, uri := range resp.RedirectURIs {
		if uri != dangerousURIs[i] {
			t.Errorf("RedirectURIs[%d] = %q, want %q", i, uri, dangerousURIs[i])
		}
	}
}

func TestOAuthClient_SecretHashNotInAdminResponse(t *testing.T) {
	client := &OAuthClient{
		ID:               "client-4",
		ClientSecretHash: []byte("very-secret-hash"),
		Name:             "Test",
	}
	resp := client.ToAdminResponse()
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if strings.Contains(string(data), "very-secret-hash") {
		t.Error("AdminClientResponse JSON contains client secret hash")
	}
}

func TestOAuthClient_ManyRedirectURIs(t *testing.T) {
	uris := make([]string, 100)
	for i := range uris {
		uris[i] = "https://example.com/callback/" + strings.Repeat("x", i)
	}
	client := &OAuthClient{ID: "client-5", RedirectURIs: uris}
	resp := client.ToAdminResponse()
	if len(resp.RedirectURIs) != 100 {
		t.Errorf("RedirectURIs length = %d, want 100", len(resp.RedirectURIs))
	}
}

// ---------------------------------------------------------------------------
// 3. Organization / ValidateCSSColor edge cases
// ---------------------------------------------------------------------------

func TestValidateCSSColor_CSSInjection(t *testing.T) {
	injections := []struct {
		name  string
		value string
	}{
		{"behavior property", "red; behavior: url(xss.htc)"},
		{"data URI in url()", "url(data:text/css,body{background:red})"},
		{"expression with whitespace", "expression (alert(1))"},
		{"nested expression", "expression(expression(1))"},
		{"import with data URI", "@import 'data:text/css,*{color:red}'"},
		{"backslash unicode escape", `\0075rl(evil)`},
		{"null byte injection", "red\x00; background:url(evil)"},
		{"multiline injection", "red;\nbackground: url(evil)"},
		{"tab injection", "red;\tbackground: url(evil)"},
	}
	for _, tt := range injections {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCSSColor(tt.value)
			if err == nil {
				t.Errorf("ValidateCSSColor(%q) = nil, want error for CSS injection", tt.value)
			}
		})
	}
}

func TestValidateCSSColor_ValidHexEdgeCases(t *testing.T) {
	valid := []string{"#000", "#FFF", "#000000", "#FFFFFF", "#00000000", "#FFFFFFFF"}
	for _, v := range valid {
		if err := ValidateCSSColor(v); err != nil {
			t.Errorf("ValidateCSSColor(%q) = %v, want nil", v, err)
		}
	}
}

func TestValidateCSSColor_InvalidFormats(t *testing.T) {
	invalid := []struct {
		name  string
		value string
	}{
		{"hash only", "#"},
		{"hash with one char", "#f"},
		{"hash with two chars", "#ff"},
		{"hash with five chars", "#ff550"},
		{"hash with seven chars", "#ff5500f"},
		{"hash with nine chars", "#ff5500ff0"},
		{"rgb missing paren", "rgb255,0,0)"},
		{"rgb extra args", "rgb(255,0,0,0,0)"},
		{"hsl missing percent", "hsl(120,50,50)"},
		{"named with space", "dark blue"},
		{"CSS var()", "var(--primary)"},
		{"calc()", "calc(100% - 10px)"},
		{"inherit keyword", "inherit"},
		{"initial keyword", "initial"},
		{"unset keyword", "unset"},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCSSColor(tt.value)
			if err == nil {
				t.Errorf("ValidateCSSColor(%q) = nil, want error", tt.value)
			}
		})
	}
}

func TestOrganization_EmptySlug(t *testing.T) {
	org := &Organization{ID: uuid.New(), Name: "Test", Slug: ""}
	resp := org.ToOrgResponse(0)
	if resp.Slug != "" {
		t.Errorf("Slug = %q, want empty", resp.Slug)
	}
}

func TestOrganization_SpecialCharsInSlug(t *testing.T) {
	specialSlugs := []string{
		"my-org",
		"my_org",
		"org.name",
		"org/name",
		"org with spaces",
		"org@special#chars",
		"<script>alert(1)</script>",
		"'; DROP TABLE orgs; --",
	}
	for _, slug := range specialSlugs {
		org := &Organization{ID: uuid.New(), Name: "Test", Slug: slug}
		resp := org.ToOrgResponse(0)
		if resp.Slug != slug {
			t.Errorf("Slug = %q, want %q", resp.Slug, slug)
		}
	}
}

// ---------------------------------------------------------------------------
// 4. Audit event constants uniqueness and non-empty
// ---------------------------------------------------------------------------

func TestAuditEventConstants_Unique(t *testing.T) {
	events := []string{
		EventUserLogin,
		EventUserLoginFailed,
		EventUserCreated,
		EventUserUpdated,
		EventUserDeleted,
		EventUserPasswordReset,
		EventClientCreated,
		EventOrgCreated,
		EventSessionRevoked,
		EventSessionsRevokedAll,
		EventSocialLogin,
		EventSocialLoginFailed,
		EventSocialAccountLinked,
		EventMFAEnrolled,
		EventMFADisabled,
		EventMFAVerified,
		EventMFAFailed,
		EventPasswordResetRequested,
		EventPasswordResetCompleted,
		EventRoleAssigned,
		EventRoleUnassigned,
	}

	seen := make(map[string]bool, len(events))
	for _, e := range events {
		if e == "" {
			t.Error("audit event constant is empty string")
			continue
		}
		if seen[e] {
			t.Errorf("duplicate audit event constant: %q", e)
		}
		seen[e] = true
	}
}

func TestAuditEventConstants_NoEmpty(t *testing.T) {
	events := []string{
		EventUserLogin, EventUserLoginFailed, EventUserCreated,
		EventUserUpdated, EventUserDeleted, EventUserPasswordReset,
		EventClientCreated, EventOrgCreated, EventSessionRevoked,
		EventSessionsRevokedAll, EventSocialLogin, EventSocialLoginFailed,
		EventSocialAccountLinked, EventMFAEnrolled, EventMFADisabled,
		EventMFAVerified, EventMFAFailed, EventPasswordResetRequested,
		EventPasswordResetCompleted, EventRoleAssigned, EventRoleUnassigned,
	}
	for _, e := range events {
		if strings.TrimSpace(e) == "" {
			t.Errorf("audit event constant is empty or whitespace-only: %q", e)
		}
	}
}

func TestAuditEventConstants_Format(t *testing.T) {
	// All event constants should follow "resource.action" format.
	events := map[string]string{
		"EventUserLogin":              EventUserLogin,
		"EventUserLoginFailed":        EventUserLoginFailed,
		"EventUserCreated":            EventUserCreated,
		"EventUserUpdated":            EventUserUpdated,
		"EventUserDeleted":            EventUserDeleted,
		"EventUserPasswordReset":      EventUserPasswordReset,
		"EventClientCreated":          EventClientCreated,
		"EventOrgCreated":             EventOrgCreated,
		"EventSessionRevoked":         EventSessionRevoked,
		"EventSessionsRevokedAll":     EventSessionsRevokedAll,
		"EventMFAEnrolled":            EventMFAEnrolled,
		"EventMFADisabled":            EventMFADisabled,
		"EventMFAVerified":            EventMFAVerified,
		"EventMFAFailed":              EventMFAFailed,
		"EventPasswordResetRequested": EventPasswordResetRequested,
		"EventPasswordResetCompleted": EventPasswordResetCompleted,
		"EventRoleAssigned":           EventRoleAssigned,
		"EventRoleUnassigned":         EventRoleUnassigned,
	}
	for name, val := range events {
		// social_login and social_login_failed don't follow dot format, skip them.
		if val == EventSocialLogin || val == EventSocialLoginFailed || val == EventSocialAccountLinked {
			continue
		}
		if !strings.Contains(val, ".") {
			t.Errorf("%s = %q, expected resource.action format with a dot", name, val)
		}
	}
}

// ---------------------------------------------------------------------------
// 5. Password-related fields edge cases (RegistrationRequest, ResetPasswordRequest)
// ---------------------------------------------------------------------------

func TestRegistrationRequest_EmptyPassword(t *testing.T) {
	req := RegistrationRequest{Username: "user", Email: "u@e.com", Password: ""}
	if req.Password != "" {
		t.Error("Password should be empty")
	}
}

func TestRegistrationRequest_VeryLongPassword(t *testing.T) {
	long := strings.Repeat("P", 10000)
	req := RegistrationRequest{Password: long}
	if len(req.Password) != 10000 {
		t.Errorf("Password length = %d, want 10000", len(req.Password))
	}
}

func TestRegistrationRequest_UnicodePassword(t *testing.T) {
	passwords := []string{
		"\u00e9\u00e0\u00fc\u00f1\u00f6\u00e4",
		"\U0001f512\U0001f511\U0001f510",
		"\u0410\u0411\u0412\u0413\u0414",
		"\u4e16\u754c\u4f60\u597d",
	}
	for _, pw := range passwords {
		req := RegistrationRequest{Password: pw}
		if req.Password != pw {
			t.Errorf("Password = %q, want %q", req.Password, pw)
		}
	}
}

func TestRegistrationRequest_NullBytesInPassword(t *testing.T) {
	pw := "pass\x00word"
	req := RegistrationRequest{Password: pw}
	if req.Password != pw {
		t.Errorf("Password = %q, want %q", req.Password, pw)
	}
	if len(req.Password) != 9 {
		t.Errorf("Password length = %d, want 9 (includes null byte)", len(req.Password))
	}
}

func TestResetPasswordRequest_EmptyPassword(t *testing.T) {
	req := ResetPasswordRequest{Password: ""}
	if req.Password != "" {
		t.Error("Password should be empty")
	}
}

func TestRegistrationRequest_JSONRoundTrip(t *testing.T) {
	original := RegistrationRequest{
		Username:   "testuser",
		Email:      "test@example.com",
		Password:   "MyP@ssw0rd!",
		GivenName:  "Test",
		FamilyName: "User",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var decoded RegistrationRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if decoded != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// ---------------------------------------------------------------------------
// 6. UUID fields edge cases
// ---------------------------------------------------------------------------

func TestUser_NilUUID(t *testing.T) {
	var zeroID uuid.UUID
	user := &User{ID: zeroID, OrgID: zeroID}
	resp := user.ToResponse()
	if resp.ID != uuid.Nil {
		t.Errorf("ID = %v, want uuid.Nil", resp.ID)
	}
	if resp.OrgID != uuid.Nil {
		t.Errorf("OrgID = %v, want uuid.Nil", resp.OrgID)
	}
}

func TestUser_ZeroUUID(t *testing.T) {
	user := &User{ID: uuid.Nil, OrgID: uuid.Nil}
	resp := user.ToResponse()
	if resp.ID.String() != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("ID = %q, want zero UUID", resp.ID.String())
	}
}

func TestAuditEvent_NilActorID(t *testing.T) {
	event := AuditEvent{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		EventType: EventUserLogin,
		ActorID:   nil,
	}
	if event.ActorID != nil {
		t.Error("ActorID should be nil")
	}
}

func TestAuditEvent_WithActorID(t *testing.T) {
	actorID := uuid.New()
	event := AuditEvent{
		ID:        uuid.New(),
		EventType: EventUserLogin,
		ActorID:   &actorID,
	}
	if event.ActorID == nil {
		t.Fatal("ActorID should not be nil")
	}
	if *event.ActorID != actorID {
		t.Errorf("ActorID = %v, want %v", *event.ActorID, actorID)
	}
}

func TestOAuthClient_EmptyStringID(t *testing.T) {
	client := &OAuthClient{ID: ""}
	resp := client.ToAdminResponse()
	if resp.ID != "" {
		t.Errorf("ID = %q, want empty", resp.ID)
	}
}

// ---------------------------------------------------------------------------
// 7. Timestamp fields edge cases
// ---------------------------------------------------------------------------

func TestUser_ZeroTime(t *testing.T) {
	var zero time.Time
	user := &User{ID: uuid.New(), CreatedAt: zero, UpdatedAt: zero}
	resp := user.ToResponse()
	if !resp.CreatedAt.IsZero() {
		t.Errorf("CreatedAt = %v, want zero time", resp.CreatedAt)
	}
	if !resp.UpdatedAt.IsZero() {
		t.Errorf("UpdatedAt = %v, want zero time", resp.UpdatedAt)
	}
}

func TestUser_FarFutureTime(t *testing.T) {
	future := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	user := &User{ID: uuid.New(), CreatedAt: future, UpdatedAt: future}
	resp := user.ToResponse()
	if !resp.CreatedAt.Equal(future) {
		t.Errorf("CreatedAt = %v, want %v", resp.CreatedAt, future)
	}
}

func TestUser_FarPastTime(t *testing.T) {
	past := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	user := &User{ID: uuid.New(), CreatedAt: past, UpdatedAt: past}
	resp := user.ToResponse()
	if !resp.CreatedAt.Equal(past) {
		t.Errorf("CreatedAt = %v, want %v", resp.CreatedAt, past)
	}
}

func TestUser_IsLocked_FutureLockedUntil(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	user := &User{LockedUntil: &future}
	if !user.IsLocked() {
		t.Error("IsLocked() = false, want true for future LockedUntil")
	}
}

func TestUser_IsLocked_PastLockedUntil(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	user := &User{LockedUntil: &past}
	if user.IsLocked() {
		t.Error("IsLocked() = true, want false for past LockedUntil")
	}
}

func TestUser_IsLocked_NilLockedUntil(t *testing.T) {
	user := &User{LockedUntil: nil}
	if user.IsLocked() {
		t.Error("IsLocked() = true, want false for nil LockedUntil")
	}
}

func TestUser_IsLocked_ZeroTime(t *testing.T) {
	zero := time.Time{}
	user := &User{LockedUntil: &zero}
	if user.IsLocked() {
		t.Error("IsLocked() = true, want false for zero-time LockedUntil")
	}
}

func TestOrgSettings_ZeroDurations(t *testing.T) {
	settings := &OrgSettings{
		AccessTokenTTL:  0,
		RefreshTokenTTL: 0,
	}
	resp := settings.ToResponse()
	if resp.AccessTokenTTLSeconds != 0 {
		t.Errorf("AccessTokenTTLSeconds = %d, want 0", resp.AccessTokenTTLSeconds)
	}
	if resp.RefreshTokenTTLSeconds != 0 {
		t.Errorf("RefreshTokenTTLSeconds = %d, want 0", resp.RefreshTokenTTLSeconds)
	}
}

func TestOrgSettings_NegativeDuration(t *testing.T) {
	settings := &OrgSettings{
		AccessTokenTTL: -5 * time.Minute,
	}
	resp := settings.ToResponse()
	if resp.AccessTokenTTLSeconds >= 0 {
		t.Errorf("AccessTokenTTLSeconds = %d, expected negative for negative duration", resp.AccessTokenTTLSeconds)
	}
}

func TestSessionResponse_ZeroExpiry(t *testing.T) {
	s := SessionResponse{ID: uuid.New(), CreatedAt: time.Now(), ExpiresAt: time.Time{}}
	if !s.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be zero time")
	}
}

// ---------------------------------------------------------------------------
// 8. Role edge cases
// ---------------------------------------------------------------------------

func TestRole_EmptyName(t *testing.T) {
	role := &Role{ID: uuid.New(), Name: ""}
	resp := role.ToRoleResponse(0)
	if resp.Name != "" {
		t.Errorf("Name = %q, want empty", resp.Name)
	}
}

func TestRole_SpecialCharNames(t *testing.T) {
	names := []string{
		"super-admin",
		"role_with_underscore",
		"role.with.dots",
		"ROLE WITH SPACES",
		"<script>alert(1)</script>",
		"role/with/slashes",
		"role\twith\ttabs",
	}
	for _, name := range names {
		role := &Role{ID: uuid.New(), Name: name}
		resp := role.ToRoleResponse(0)
		if resp.Name != name {
			t.Errorf("Name = %q, want %q", resp.Name, name)
		}
	}
}

func TestRole_ZeroUserCount(t *testing.T) {
	role := &Role{ID: uuid.New(), Name: "empty-role"}
	resp := role.ToRoleResponse(0)
	if resp.UserCount != 0 {
		t.Errorf("UserCount = %d, want 0", resp.UserCount)
	}
}

func TestRole_NegativeUserCount(t *testing.T) {
	role := &Role{ID: uuid.New(), Name: "bug-role"}
	resp := role.ToRoleResponse(-1)
	if resp.UserCount != -1 {
		t.Errorf("UserCount = %d, want -1", resp.UserCount)
	}
}

func TestRole_LargeUserCount(t *testing.T) {
	role := &Role{ID: uuid.New(), Name: "popular-role"}
	resp := role.ToRoleResponse(999999999)
	if resp.UserCount != 999999999 {
		t.Errorf("UserCount = %d, want 999999999", resp.UserCount)
	}
}

// ---------------------------------------------------------------------------
// 9. Social account edge cases
// ---------------------------------------------------------------------------

func TestSocialAccount_EmptyProvider(t *testing.T) {
	sa := &SocialAccount{ID: uuid.New(), Provider: ""}
	resp := sa.ToResponse()
	if resp.Provider != "" {
		t.Errorf("Provider = %q, want empty", resp.Provider)
	}
}

func TestSocialAccount_UnknownProvider(t *testing.T) {
	unknownProviders := []string{
		"myspace",
		"friendster",
		"custom-idp",
		"oidc-provider-xyz",
		"<script>alert(1)</script>",
	}
	for _, prov := range unknownProviders {
		sa := &SocialAccount{ID: uuid.New(), Provider: prov, Email: "user@example.com"}
		resp := sa.ToResponse()
		if resp.Provider != prov {
			t.Errorf("Provider = %q, want %q", resp.Provider, prov)
		}
	}
}

func TestSocialAccount_SensitiveFieldsStripped(t *testing.T) {
	sa := &SocialAccount{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		Provider:     "google",
		Email:        "user@gmail.com",
		Name:         "Test User",
		AccessToken:  "ya29.secret-access-token",
		RefreshToken: "1//secret-refresh-token",
	}
	resp := sa.ToResponse()

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	jsonStr := string(data)
	if strings.Contains(jsonStr, "ya29.secret-access-token") {
		t.Error("SocialAccountResponse contains access token")
	}
	if strings.Contains(jsonStr, "1//secret-refresh-token") {
		t.Error("SocialAccountResponse contains refresh token")
	}
	if strings.Contains(jsonStr, "user_id") {
		t.Error("SocialAccountResponse contains user_id")
	}
}

func TestSocialAccount_ToResponse_PreservesFields(t *testing.T) {
	id := uuid.New()
	sa := &SocialAccount{
		ID:       id,
		Provider: "github",
		Email:    "dev@github.com",
		Name:     "Dev User",
	}
	resp := sa.ToResponse()
	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.Provider != "github" {
		t.Errorf("Provider = %q, want github", resp.Provider)
	}
	if resp.Email != "dev@github.com" {
		t.Errorf("Email = %q, want dev@github.com", resp.Email)
	}
	if resp.Name != "Dev User" {
		t.Errorf("Name = %q, want Dev User", resp.Name)
	}
}

// ---------------------------------------------------------------------------
// 10. SocialProviderConfig edge cases
// ---------------------------------------------------------------------------

func TestSocialProviderConfig_EmptySecret(t *testing.T) {
	config := &SocialProviderConfig{
		ID:           uuid.New(),
		Provider:     "google",
		ClientID:     "my-client-id",
		ClientSecret: "",
	}
	resp := config.ToResponse()
	if resp.ClientSecret != "" {
		t.Errorf("ClientSecret = %q, want empty (not masked) when secret is empty", resp.ClientSecret)
	}
}

func TestSocialProviderConfig_SecretMasked(t *testing.T) {
	config := &SocialProviderConfig{
		ID:           uuid.New(),
		Provider:     "google",
		ClientID:     "my-client-id",
		ClientSecret: "super-secret-value",
	}
	resp := config.ToResponse()
	if resp.ClientSecret != "********" {
		t.Errorf("ClientSecret = %q, want ********", resp.ClientSecret)
	}
}

func TestSocialProviderConfig_SecretNotInJSON(t *testing.T) {
	config := &SocialProviderConfig{
		ID:           uuid.New(),
		Provider:     "github",
		ClientSecret: "gh_secret_abc123",
	}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if strings.Contains(string(data), "gh_secret_abc123") {
		t.Error("SocialProviderConfig JSON contains raw client secret (json:\"-\" tag not working)")
	}
}

// ---------------------------------------------------------------------------
// Additional edge cases: JSON serialization, WebAuthn, Webhook
// ---------------------------------------------------------------------------

func TestDashboardStats_QueryErrorsOmittedWhenZero(t *testing.T) {
	stats := DashboardStats{TotalUsers: 10, QueryErrors: 0}
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if strings.Contains(string(data), "query_errors") {
		t.Error("QueryErrors should be omitted from JSON when zero (omitempty)")
	}
}

func TestDashboardStats_QueryErrorsIncludedWhenNonZero(t *testing.T) {
	stats := DashboardStats{TotalUsers: 10, QueryErrors: 3}
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if !strings.Contains(string(data), `"query_errors":3`) {
		t.Errorf("QueryErrors should be in JSON when non-zero, got %s", string(data))
	}
}

func TestWebAuthnUser_DisplayName_GivenNameOnly(t *testing.T) {
	wau := &WebAuthnUser{User: &User{GivenName: "Alice", FamilyName: ""}}
	if wau.WebAuthnDisplayName() != "Alice" {
		t.Errorf("WebAuthnDisplayName() = %q, want Alice", wau.WebAuthnDisplayName())
	}
}

func TestWebAuthnUser_DisplayName_FullName(t *testing.T) {
	wau := &WebAuthnUser{User: &User{GivenName: "Alice", FamilyName: "Smith"}}
	if wau.WebAuthnDisplayName() != "Alice Smith" {
		t.Errorf("WebAuthnDisplayName() = %q, want 'Alice Smith'", wau.WebAuthnDisplayName())
	}
}

func TestWebAuthnUser_DisplayName_FallbackToUsername(t *testing.T) {
	wau := &WebAuthnUser{User: &User{Username: "alice123", GivenName: "", FamilyName: ""}}
	if wau.WebAuthnDisplayName() != "alice123" {
		t.Errorf("WebAuthnDisplayName() = %q, want alice123", wau.WebAuthnDisplayName())
	}
}

func TestWebAuthnUser_DisplayName_EmptyEverything(t *testing.T) {
	wau := &WebAuthnUser{User: &User{}}
	// Should return empty username when nothing is set.
	if wau.WebAuthnDisplayName() != "" {
		t.Errorf("WebAuthnDisplayName() = %q, want empty", wau.WebAuthnDisplayName())
	}
}

func TestWebAuthnUser_WebAuthnName(t *testing.T) {
	wau := &WebAuthnUser{User: &User{Username: "webauthn-user"}}
	if wau.WebAuthnName() != "webauthn-user" {
		t.Errorf("WebAuthnName() = %q, want webauthn-user", wau.WebAuthnName())
	}
}

func TestWebAuthnUser_WebAuthnID(t *testing.T) {
	id := uuid.New()
	wau := &WebAuthnUser{User: &User{ID: id}}
	idBytes := wau.WebAuthnID()
	if len(idBytes) == 0 {
		t.Error("WebAuthnID() returned empty bytes")
	}
	// Parse it back to verify round-trip.
	parsed, err := uuid.FromBytes(idBytes)
	if err != nil {
		t.Fatalf("uuid.FromBytes failed: %v", err)
	}
	if parsed != id {
		t.Errorf("WebAuthnID round-trip: got %v, want %v", parsed, id)
	}
}

func TestWebAuthnUser_EmptyCredentials(t *testing.T) {
	wau := &WebAuthnUser{User: &User{ID: uuid.New()}}
	creds := wau.WebAuthnCredentials()
	if creds != nil {
		t.Errorf("WebAuthnCredentials() = %v, want nil", creds)
	}
}

func TestAuditEvent_DetailsNil(t *testing.T) {
	event := AuditEvent{ID: uuid.New(), Details: nil}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	// nil map should serialize as null in JSON.
	if !strings.Contains(string(data), `"details":null`) {
		t.Errorf("nil Details should serialize as null, got %s", string(data))
	}
}

func TestAuditEvent_DetailsWithNestedData(t *testing.T) {
	event := AuditEvent{
		ID:        uuid.New(),
		EventType: EventUserLogin,
		Details: map[string]any{
			"ip":         "192.168.1.1",
			"user_agent": "Mozilla/5.0",
			"nested": map[string]any{
				"key": "value",
			},
		},
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if !strings.Contains(string(data), `"ip":"192.168.1.1"`) {
		t.Error("Details missing expected ip field")
	}
}

func TestMFADevice_ZeroValues(t *testing.T) {
	device := MFADevice{}
	if device.ID != uuid.Nil {
		t.Errorf("zero MFADevice ID = %v, want uuid.Nil", device.ID)
	}
	if device.Verified {
		t.Error("zero MFADevice Verified = true, want false")
	}
	if device.DeviceType != "" {
		t.Errorf("zero MFADevice DeviceType = %q, want empty", device.DeviceType)
	}
}

func TestGroup_ZeroCounts(t *testing.T) {
	group := &Group{ID: uuid.New(), Name: "empty"}
	resp := group.ToGroupResponse(0, 0)
	if resp.MemberCount != 0 {
		t.Errorf("MemberCount = %d, want 0", resp.MemberCount)
	}
	if resp.RoleCount != 0 {
		t.Errorf("RoleCount = %d, want 0", resp.RoleCount)
	}
}

func TestGroup_NegativeCounts(t *testing.T) {
	group := &Group{ID: uuid.New(), Name: "buggy"}
	resp := group.ToGroupResponse(-1, -1)
	if resp.MemberCount != -1 {
		t.Errorf("MemberCount = %d, want -1", resp.MemberCount)
	}
	if resp.RoleCount != -1 {
		t.Errorf("RoleCount = %d, want -1", resp.RoleCount)
	}
}

func TestWebhookPayload_EmptyDetails(t *testing.T) {
	payload := WebhookPayload{
		ID:   "evt-123",
		Type: EventUserLogin,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	// nil map fields with omitempty should be omitted.
	if strings.Contains(string(data), `"details":{`) {
		t.Error("empty Details should not appear as non-null object")
	}
}

func TestListUsersResponse_EmptyList(t *testing.T) {
	resp := ListUsersResponse{Users: []*AdminUserResponse{}, Total: 0, Page: 1, Limit: 10}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if !strings.Contains(string(data), `"users":[]`) {
		t.Errorf("empty users list should serialize as [], got %s", string(data))
	}
}

func TestCreateClientRequest_JSONDecode(t *testing.T) {
	input := `{"name":"App","description":"desc","client_type":"public","redirect_uris":"https://app.com/cb"}`
	var req CreateClientRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if req.Name != "App" {
		t.Errorf("Name = %q, want App", req.Name)
	}
	if req.ClientType != "public" {
		t.Errorf("ClientType = %q, want public", req.ClientType)
	}
	// Note: RedirectURIs is a single string in CreateClientRequest, not a slice.
	if req.RedirectURIs != "https://app.com/cb" {
		t.Errorf("RedirectURIs = %q, want https://app.com/cb", req.RedirectURIs)
	}
}

func TestAuthorizationCode_ZeroValues(t *testing.T) {
	code := AuthorizationCode{}
	if code.ID != uuid.Nil {
		t.Errorf("zero AuthorizationCode ID = %v, want uuid.Nil", code.ID)
	}
	if code.Used {
		t.Error("zero AuthorizationCode Used = true, want false")
	}
	if !code.ExpiresAt.IsZero() {
		t.Error("zero AuthorizationCode ExpiresAt should be zero time")
	}
}
