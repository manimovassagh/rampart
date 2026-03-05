package model

// OrgExport is the JSON structure for organization export/import.
type OrgExport struct {
	Organization OrgExportData      `json:"organization"`
	Settings     *OrgSettingsExport `json:"settings,omitempty"`
	Roles        []RoleExport       `json:"roles,omitempty"`
	Groups       []GroupExport      `json:"groups,omitempty"`
	Clients      []ClientExport     `json:"clients,omitempty"`
}

// OrgExportData is the organization info in an export.
type OrgExportData struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
}

// OrgSettingsExport captures all settings for export.
type OrgSettingsExport struct {
	PasswordMinLength         int    `json:"password_min_length"`
	PasswordRequireUppercase  bool   `json:"password_require_uppercase"`
	PasswordRequireLowercase  bool   `json:"password_require_lowercase"`
	PasswordRequireNumbers    bool   `json:"password_require_numbers"`
	PasswordRequireSymbols    bool   `json:"password_require_symbols"`
	MFAEnforcement            string `json:"mfa_enforcement"`
	AccessTokenTTLSeconds     int    `json:"access_token_ttl_seconds"`
	RefreshTokenTTLSeconds    int    `json:"refresh_token_ttl_seconds"`
	SelfRegistrationEnabled   bool   `json:"self_registration_enabled"`
	EmailVerificationRequired bool   `json:"email_verification_required"`
	ForgotPasswordEnabled     bool   `json:"forgot_password_enabled"`
	RememberMeEnabled         bool   `json:"remember_me_enabled"`
	LoginPageTitle            string `json:"login_page_title"`
	LoginPageMessage          string `json:"login_page_message"`
}

// RoleExport captures a role for export.
type RoleExport struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GroupExport captures a group and its role names for export.
type GroupExport struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Roles       []string `json:"roles,omitempty"`
}

// ClientExport captures an OAuth client for export (no secrets).
type ClientExport struct {
	ClientID     string   `json:"client_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	ClientType   string   `json:"client_type"`
	RedirectURIs []string `json:"redirect_uris"`
	Enabled      bool     `json:"enabled"`
}
