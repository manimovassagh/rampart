package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/manimovassagh/rampart/internal/model"
)

// GetOrgSettings returns the settings for an organization.
func (db *DB) GetOrgSettings(ctx context.Context, orgID uuid.UUID) (*model.OrgSettings, error) {
	query := `
		SELECT id, org_id,
		       password_min_length, password_require_uppercase, password_require_lowercase,
		       password_require_numbers, password_require_symbols,
		       mfa_enforcement, access_token_ttl, refresh_token_ttl,
		       logo_url, primary_color, background_color,
		       self_registration_enabled, email_verification_required,
		       forgot_password_enabled, remember_me_enabled,
		       login_page_title, login_page_message, login_theme,
		       max_failed_login_attempts, lockout_duration_seconds,
		       created_at, updated_at
		FROM organization_settings
		WHERE org_id = $1`

	var s model.OrgSettings
	var logoURL, primaryColor, bgColor *string
	var loginTitle, loginMessage, loginTheme *string
	var accessTTL, refreshTTL time.Duration
	var lockoutDurationSecs int

	err := db.Pool.QueryRow(ctx, query, orgID).Scan(
		&s.ID, &s.OrgID,
		&s.PasswordMinLength, &s.PasswordRequireUppercase, &s.PasswordRequireLowercase,
		&s.PasswordRequireNumbers, &s.PasswordRequireSymbols,
		&s.MFAEnforcement, &accessTTL, &refreshTTL,
		&logoURL, &primaryColor, &bgColor,
		&s.SelfRegistrationEnabled, &s.EmailVerificationRequired,
		&s.ForgotPasswordEnabled, &s.RememberMeEnabled,
		&loginTitle, &loginMessage, &loginTheme,
		&s.MaxFailedLoginAttempts, &lockoutDurationSecs,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying organization settings: %w", err)
	}

	s.AccessTokenTTL = accessTTL
	s.RefreshTokenTTL = refreshTTL
	s.LockoutDuration = time.Duration(lockoutDurationSecs) * time.Second
	if logoURL != nil {
		s.LogoURL = *logoURL
	}
	if primaryColor != nil {
		s.PrimaryColor = *primaryColor
	}
	if bgColor != nil {
		s.BackgroundColor = *bgColor
	}
	if loginTitle != nil {
		s.LoginPageTitle = *loginTitle
	}
	if loginMessage != nil {
		s.LoginPageMessage = *loginMessage
	}
	if loginTheme != nil {
		s.LoginTheme = *loginTheme
	}

	return &s, nil
}

// UpdateOrgSettings updates the settings for an organization.
func (db *DB) UpdateOrgSettings(ctx context.Context, orgID uuid.UUID, req *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	accessTTL := fmt.Sprintf("%d seconds", req.AccessTokenTTLSeconds)
	refreshTTL := fmt.Sprintf("%d seconds", req.RefreshTokenTTLSeconds)

	query := `
		UPDATE organization_settings
		SET password_min_length = $2,
		    password_require_uppercase = $3,
		    password_require_lowercase = $4,
		    password_require_numbers = $5,
		    password_require_symbols = $6,
		    mfa_enforcement = $7,
		    access_token_ttl = $8::interval,
		    refresh_token_ttl = $9::interval,
		    logo_url = NULLIF($10, ''),
		    primary_color = NULLIF($11, ''),
		    background_color = NULLIF($12, ''),
		    self_registration_enabled = $13,
		    email_verification_required = $14,
		    forgot_password_enabled = $15,
		    remember_me_enabled = $16,
		    login_page_title = NULLIF($17, ''),
		    login_page_message = NULLIF($18, ''),
		    login_theme = COALESCE(NULLIF($19, ''), 'default'),
		    updated_at = now()
		WHERE org_id = $1
		RETURNING id, org_id,
		          password_min_length, password_require_uppercase, password_require_lowercase,
		          password_require_numbers, password_require_symbols,
		          mfa_enforcement, access_token_ttl, refresh_token_ttl,
		          logo_url, primary_color, background_color,
		          self_registration_enabled, email_verification_required,
		          forgot_password_enabled, remember_me_enabled,
		          login_page_title, login_page_message, login_theme,
		          created_at, updated_at`

	var s model.OrgSettings
	var logoURL, primaryColor, bgColor *string
	var loginTitle, loginMessage, loginTheme *string
	var aTTL, rTTL time.Duration

	err := db.Pool.QueryRow(ctx, query,
		orgID,
		req.PasswordMinLength,
		req.PasswordRequireUppercase,
		req.PasswordRequireLowercase,
		req.PasswordRequireNumbers,
		req.PasswordRequireSymbols,
		req.MFAEnforcement,
		accessTTL,
		refreshTTL,
		req.LogoURL,
		req.PrimaryColor,
		req.BackgroundColor,
		req.SelfRegistrationEnabled,
		req.EmailVerificationRequired,
		req.ForgotPasswordEnabled,
		req.RememberMeEnabled,
		req.LoginPageTitle,
		req.LoginPageMessage,
		req.LoginTheme,
	).Scan(
		&s.ID, &s.OrgID,
		&s.PasswordMinLength, &s.PasswordRequireUppercase, &s.PasswordRequireLowercase,
		&s.PasswordRequireNumbers, &s.PasswordRequireSymbols,
		&s.MFAEnforcement, &aTTL, &rTTL,
		&logoURL, &primaryColor, &bgColor,
		&s.SelfRegistrationEnabled, &s.EmailVerificationRequired,
		&s.ForgotPasswordEnabled, &s.RememberMeEnabled,
		&loginTitle, &loginMessage, &loginTheme,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("updating organization settings: %w", err)
	}

	s.AccessTokenTTL = aTTL
	s.RefreshTokenTTL = rTTL
	if logoURL != nil {
		s.LogoURL = *logoURL
	}
	if primaryColor != nil {
		s.PrimaryColor = *primaryColor
	}
	if bgColor != nil {
		s.BackgroundColor = *bgColor
	}
	if loginTitle != nil {
		s.LoginPageTitle = *loginTitle
	}
	if loginMessage != nil {
		s.LoginPageMessage = *loginMessage
	}
	if loginTheme != nil {
		s.LoginTheme = *loginTheme
	}

	return &s, nil
}
