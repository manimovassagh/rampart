package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// cssColorHexPattern matches 3, 4, 6, or 8-digit hex colors like #fff, #FFFFFF, #ff000080.
var cssColorHexPattern = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

// cssColorRGBPattern matches rgb(r,g,b) and rgba(r,g,b,a) with integers/percentages.
var cssColorRGBPattern = regexp.MustCompile(`^rgba?\(\s*\d{1,3}%?\s*,\s*\d{1,3}%?\s*,\s*\d{1,3}%?\s*(,\s*(0|1|0?\.\d+)\s*)?\)$`)

// cssColorHSLPattern matches hsl(h,s%,l%) and hsla(h,s%,l%,a).
var cssColorHSLPattern = regexp.MustCompile(`^hsla?\(\s*\d{1,3}\s*,\s*\d{1,3}%\s*,\s*\d{1,3}%\s*(,\s*(0|1|0?\.\d+)\s*)?\)$`)

// cssNamedColors is the set of standard CSS named colors.
var cssNamedColors = map[string]bool{
	"aliceblue": true, "antiquewhite": true, "aqua": true, "aquamarine": true,
	"azure": true, "beige": true, "bisque": true, "black": true,
	"blanchedalmond": true, "blue": true, "blueviolet": true, "brown": true,
	"burlywood": true, "cadetblue": true, "chartreuse": true, "chocolate": true,
	"coral": true, "cornflowerblue": true, "cornsilk": true, "crimson": true,
	"cyan": true, "darkblue": true, "darkcyan": true, "darkgoldenrod": true,
	"darkgray": true, "darkgreen": true, "darkgrey": true, "darkkhaki": true,
	"darkmagenta": true, "darkolivegreen": true, "darkorange": true, "darkorchid": true,
	"darkred": true, "darksalmon": true, "darkseagreen": true, "darkslateblue": true,
	"darkslategray": true, "darkslategrey": true, "darkturquoise": true, "darkviolet": true,
	"deeppink": true, "deepskyblue": true, "dimgray": true, "dimgrey": true,
	"dodgerblue": true, "firebrick": true, "floralwhite": true, "forestgreen": true,
	"fuchsia": true, "gainsboro": true, "ghostwhite": true, "gold": true,
	"goldenrod": true, "gray": true, "green": true, "greenyellow": true,
	"grey": true, "honeydew": true, "hotpink": true, "indianred": true,
	"indigo": true, "ivory": true, "khaki": true, "lavender": true,
	"lavenderblush": true, "lawngreen": true, "lemonchiffon": true, "lightblue": true,
	"lightcoral": true, "lightcyan": true, "lightgoldenrodyellow": true, "lightgray": true,
	"lightgreen": true, "lightgrey": true, "lightpink": true, "lightsalmon": true,
	"lightseagreen": true, "lightskyblue": true, "lightslategray": true, "lightslategrey": true,
	"lightsteelblue": true, "lightyellow": true, "lime": true, "limegreen": true,
	"linen": true, "magenta": true, "maroon": true, "mediumaquamarine": true,
	"mediumblue": true, "mediumorchid": true, "mediumpurple": true, "mediumseagreen": true,
	"mediumslateblue": true, "mediumspringgreen": true, "mediumturquoise": true, "mediumvioletred": true,
	"midnightblue": true, "mintcream": true, "mistyrose": true, "moccasin": true,
	"navajowhite": true, "navy": true, "oldlace": true, "olive": true,
	"olivedrab": true, "orange": true, "orangered": true, "orchid": true,
	"palegoldenrod": true, "palegreen": true, "paleturquoise": true, "palevioletred": true,
	"papayawhip": true, "peachpuff": true, "peru": true, "pink": true,
	"plum": true, "powderblue": true, "purple": true, "rebeccapurple": true,
	"red": true, "rosybrown": true, "royalblue": true, "saddlebrown": true,
	"salmon": true, "sandybrown": true, "seagreen": true, "seashell": true,
	"sienna": true, "silver": true, "skyblue": true, "slateblue": true,
	"slategray": true, "slategrey": true, "snow": true, "springgreen": true,
	"steelblue": true, "tan": true, "teal": true, "thistle": true,
	"tomato": true, "transparent": true, "turquoise": true, "violet": true,
	"wheat": true, "white": true, "whitesmoke": true, "yellow": true,
	"yellowgreen": true,
}

// ValidateCSSColor checks that a string is a safe CSS color value.
// It accepts empty strings (no color set), hex colors, rgb/rgba, hsl/hsla,
// and standard CSS named colors. It rejects anything that could be a CSS
// injection vector (semicolons, braces, url(), expression(), @import, etc.).
func ValidateCSSColor(value string) error {
	if value == "" {
		return nil
	}

	// First, reject any dangerous characters/patterns regardless of format.
	lower := strings.ToLower(value)
	dangerousPatterns := []string{";", "{", "}", "url(", "expression(", "@import", "javascript:", "\\"}
	for _, p := range dangerousPatterns {
		if strings.Contains(lower, p) {
			return fmt.Errorf("color value contains forbidden pattern %q", p)
		}
	}

	// Check against allowed formats.
	if cssColorHexPattern.MatchString(value) {
		return nil
	}
	if cssColorRGBPattern.MatchString(lower) {
		return nil
	}
	if cssColorHSLPattern.MatchString(lower) {
		return nil
	}
	if cssNamedColors[lower] {
		return nil
	}

	return fmt.Errorf("invalid CSS color value: must be a hex color (#RGB, #RRGGBB), rgb(), hsl(), or a named CSS color")
}

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
	MaxFailedLoginAttempts    int           `json:"max_failed_login_attempts"`
	LockoutDuration           time.Duration `json:"lockout_duration"`
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
