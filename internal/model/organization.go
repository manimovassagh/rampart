package model

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents a row in the organizations table.
type Organization struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"display_name"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OrgSettings represents a row in the organization_settings table.
type OrgSettings struct {
	ID                        uuid.UUID     `json:"id"`
	OrgID                     uuid.UUID     `json:"org_id"`
	PasswordMinLength         int           `json:"password_min_length"`
	PasswordRequireUppercase  bool          `json:"password_require_uppercase"`
	PasswordRequireLowercase  bool          `json:"password_require_lowercase"`
	PasswordRequireNumbers    bool          `json:"password_require_numbers"`
	PasswordRequireSymbols    bool          `json:"password_require_symbols"`
	MFAEnforcement            string        `json:"mfa_enforcement"`
	AccessTokenTTL            time.Duration `json:"access_token_ttl"`
	RefreshTokenTTL           time.Duration `json:"refresh_token_ttl"`
	LogoURL                   string        `json:"logo_url,omitempty"`
	PrimaryColor              string        `json:"primary_color,omitempty"`
	BackgroundColor           string        `json:"background_color,omitempty"`
	SelfRegistrationEnabled   bool          `json:"self_registration_enabled"`
	EmailVerificationRequired bool          `json:"email_verification_required"`
	ForgotPasswordEnabled     bool          `json:"forgot_password_enabled"`
	RememberMeEnabled         bool          `json:"remember_me_enabled"`
	LoginPageTitle            string        `json:"login_page_title"`
	LoginPageMessage          string        `json:"login_page_message"`
	LoginTheme                string        `json:"login_theme"`
	CreatedAt                 time.Time     `json:"created_at"`
	UpdatedAt                 time.Time     `json:"updated_at"`
}

// OrgResponse is returned from the admin API with an enriched user count.
type OrgResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"display_name"`
	Enabled     bool      `json:"enabled"`
	UserCount   int       `json:"user_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToOrgResponse converts an Organization to an OrgResponse with a user count.
func (o *Organization) ToOrgResponse(userCount int) *OrgResponse {
	return &OrgResponse{
		ID:          o.ID,
		Name:        o.Name,
		Slug:        o.Slug,
		DisplayName: o.DisplayName,
		Enabled:     o.Enabled,
		UserCount:   userCount,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	}
}

// ListOrgsResponse is a paginated list of organizations for the admin API.
type ListOrgsResponse struct {
	Organizations []*OrgResponse `json:"organizations"`
	Total         int            `json:"total"`
	Page          int            `json:"page"`
	Limit         int            `json:"limit"`
}

// CreateOrgRequest is the expected JSON body for creating an organization.
type CreateOrgRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
}

// UpdateOrgRequest is the expected JSON body for updating an organization.
type UpdateOrgRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Enabled     bool   `json:"enabled"`
}

// UpdateOrgSettingsRequest is the expected JSON body for updating org settings.
type UpdateOrgSettingsRequest struct {
	PasswordMinLength         int    `json:"password_min_length"`
	PasswordRequireUppercase  bool   `json:"password_require_uppercase"`
	PasswordRequireLowercase  bool   `json:"password_require_lowercase"`
	PasswordRequireNumbers    bool   `json:"password_require_numbers"`
	PasswordRequireSymbols    bool   `json:"password_require_symbols"`
	MFAEnforcement            string `json:"mfa_enforcement"`
	AccessTokenTTLSeconds     int    `json:"access_token_ttl_seconds"`
	RefreshTokenTTLSeconds    int    `json:"refresh_token_ttl_seconds"`
	LogoURL                   string `json:"logo_url"`
	PrimaryColor              string `json:"primary_color"`
	BackgroundColor           string `json:"background_color"`
	SelfRegistrationEnabled   bool   `json:"self_registration_enabled"`
	EmailVerificationRequired bool   `json:"email_verification_required"`
	ForgotPasswordEnabled     bool   `json:"forgot_password_enabled"`
	RememberMeEnabled         bool   `json:"remember_me_enabled"`
	LoginPageTitle            string `json:"login_page_title"`
	LoginPageMessage          string `json:"login_page_message"`
	LoginTheme                string `json:"login_theme"`
}

// OrgSettingsResponse is the API representation of organization settings.
type OrgSettingsResponse struct {
	ID                        uuid.UUID `json:"id"`
	OrgID                     uuid.UUID `json:"org_id"`
	PasswordMinLength         int       `json:"password_min_length"`
	PasswordRequireUppercase  bool      `json:"password_require_uppercase"`
	PasswordRequireLowercase  bool      `json:"password_require_lowercase"`
	PasswordRequireNumbers    bool      `json:"password_require_numbers"`
	PasswordRequireSymbols    bool      `json:"password_require_symbols"`
	MFAEnforcement            string    `json:"mfa_enforcement"`
	AccessTokenTTLSeconds     int       `json:"access_token_ttl_seconds"`
	RefreshTokenTTLSeconds    int       `json:"refresh_token_ttl_seconds"`
	LogoURL                   string    `json:"logo_url,omitempty"`
	PrimaryColor              string    `json:"primary_color,omitempty"`
	BackgroundColor           string    `json:"background_color,omitempty"`
	SelfRegistrationEnabled   bool      `json:"self_registration_enabled"`
	EmailVerificationRequired bool      `json:"email_verification_required"`
	ForgotPasswordEnabled     bool      `json:"forgot_password_enabled"`
	RememberMeEnabled         bool      `json:"remember_me_enabled"`
	LoginPageTitle            string    `json:"login_page_title"`
	LoginPageMessage          string    `json:"login_page_message"`
	LoginTheme                string    `json:"login_theme"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

// ToResponse converts OrgSettings to an API-friendly response with seconds instead of Duration.
func (s *OrgSettings) ToResponse() *OrgSettingsResponse {
	return &OrgSettingsResponse{
		ID:                        s.ID,
		OrgID:                     s.OrgID,
		PasswordMinLength:         s.PasswordMinLength,
		PasswordRequireUppercase:  s.PasswordRequireUppercase,
		PasswordRequireLowercase:  s.PasswordRequireLowercase,
		PasswordRequireNumbers:    s.PasswordRequireNumbers,
		PasswordRequireSymbols:    s.PasswordRequireSymbols,
		MFAEnforcement:            s.MFAEnforcement,
		AccessTokenTTLSeconds:     int(s.AccessTokenTTL.Seconds()),
		RefreshTokenTTLSeconds:    int(s.RefreshTokenTTL.Seconds()),
		LogoURL:                   s.LogoURL,
		PrimaryColor:              s.PrimaryColor,
		BackgroundColor:           s.BackgroundColor,
		SelfRegistrationEnabled:   s.SelfRegistrationEnabled,
		EmailVerificationRequired: s.EmailVerificationRequired,
		ForgotPasswordEnabled:     s.ForgotPasswordEnabled,
		RememberMeEnabled:         s.RememberMeEnabled,
		LoginPageTitle:            s.LoginPageTitle,
		LoginPageMessage:          s.LoginPageMessage,
		LoginTheme:                s.LoginTheme,
		CreatedAt:                 s.CreatedAt,
		UpdatedAt:                 s.UpdatedAt,
	}
}
