package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
)

// ExportOrganization assembles a full export snapshot for the given organization.
func (db *DB) ExportOrganization(ctx context.Context, orgID uuid.UUID) (*model.OrgExport, error) {
	org, err := db.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("exporting organization: %w", err)
	}
	if org == nil {
		return nil, fmt.Errorf("organization not found")
	}

	export := &model.OrgExport{
		Organization: model.OrgExportData{
			Name:        org.Name,
			Slug:        org.Slug,
			DisplayName: org.DisplayName,
		},
	}

	// Settings
	settings, err := db.GetOrgSettings(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("exporting organization settings: %w", err)
	}
	if settings != nil {
		export.Settings = &model.OrgSettingsExport{
			PasswordMinLength:         settings.PasswordMinLength,
			PasswordRequireUppercase:  settings.PasswordRequireUppercase,
			PasswordRequireLowercase:  settings.PasswordRequireLowercase,
			PasswordRequireNumbers:    settings.PasswordRequireNumbers,
			PasswordRequireSymbols:    settings.PasswordRequireSymbols,
			MFAEnforcement:            settings.MFAEnforcement,
			AccessTokenTTLSeconds:     int(settings.AccessTokenTTL.Seconds()),
			RefreshTokenTTLSeconds:    int(settings.RefreshTokenTTL.Seconds()),
			SelfRegistrationEnabled:   settings.SelfRegistrationEnabled,
			EmailVerificationRequired: settings.EmailVerificationRequired,
			ForgotPasswordEnabled:     settings.ForgotPasswordEnabled,
			RememberMeEnabled:         settings.RememberMeEnabled,
			LoginPageTitle:            settings.LoginPageTitle,
			LoginPageMessage:          settings.LoginPageMessage,
			LoginTheme:                settings.LoginTheme,
		}
	}

	// Roles
	roles, _, err := db.ListRoles(ctx, orgID, "", 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("exporting roles: %w", err)
	}
	for _, r := range roles {
		export.Roles = append(export.Roles, model.RoleExport{
			Name:        r.Name,
			Description: r.Description,
		})
	}

	// Groups (with their assigned role names)
	groups, _, err := db.ListGroups(ctx, orgID, "", 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("exporting groups: %w", err)
	}
	for _, g := range groups {
		ge := model.GroupExport{
			Name:        g.Name,
			Description: g.Description,
		}
		groupRoles, err := db.GetGroupRoles(ctx, g.ID)
		if err != nil {
			return nil, fmt.Errorf("exporting group roles for %q: %w", g.Name, err)
		}
		for _, gr := range groupRoles {
			ge.Roles = append(ge.Roles, gr.RoleName)
		}
		export.Groups = append(export.Groups, ge)
	}

	// OAuth clients (no secrets)
	clients, _, err := db.ListOAuthClients(ctx, orgID, "", 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("exporting oauth clients: %w", err)
	}
	for _, c := range clients {
		export.Clients = append(export.Clients, model.ClientExport{
			ClientID:     c.ID,
			Name:         c.Name,
			Description:  c.Description,
			ClientType:   c.ClientType,
			RedirectURIs: c.RedirectURIs,
			Enabled:      c.Enabled,
		})
	}

	return export, nil
}

// ImportOrganization imports an organization snapshot in a single transaction.
// It upserts the organization, settings, roles, groups (with role assignments), and clients.
func (db *DB) ImportOrganization(ctx context.Context, export *model.OrgExport) error {
	ctx, cancel := txCtx(ctx)
	defer cancel()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning import transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Upsert organization
	var orgID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO organizations (name, slug, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (slug) DO UPDATE SET
			name = EXCLUDED.name,
			display_name = EXCLUDED.display_name,
			updated_at = now()
		RETURNING id`,
		export.Organization.Name, export.Organization.Slug, export.Organization.DisplayName,
	).Scan(&orgID)
	if err != nil {
		return fmt.Errorf("upserting organization: %w", err)
	}

	// Upsert settings
	if export.Settings != nil {
		s := export.Settings
		accessTTL := fmt.Sprintf("%d seconds", s.AccessTokenTTLSeconds)
		refreshTTL := fmt.Sprintf("%d seconds", s.RefreshTokenTTLSeconds)

		_, err = tx.Exec(ctx, `
			INSERT INTO organization_settings (org_id,
				password_min_length, password_require_uppercase, password_require_lowercase,
				password_require_numbers, password_require_symbols,
				mfa_enforcement, access_token_ttl, refresh_token_ttl,
				self_registration_enabled, email_verification_required,
				forgot_password_enabled, remember_me_enabled,
				login_page_title, login_page_message)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::interval, $9::interval, $10, $11, $12, $13, NULLIF($14, ''), NULLIF($15, ''))
			ON CONFLICT (org_id) DO UPDATE SET
				password_min_length = EXCLUDED.password_min_length,
				password_require_uppercase = EXCLUDED.password_require_uppercase,
				password_require_lowercase = EXCLUDED.password_require_lowercase,
				password_require_numbers = EXCLUDED.password_require_numbers,
				password_require_symbols = EXCLUDED.password_require_symbols,
				mfa_enforcement = EXCLUDED.mfa_enforcement,
				access_token_ttl = EXCLUDED.access_token_ttl,
				refresh_token_ttl = EXCLUDED.refresh_token_ttl,
				self_registration_enabled = EXCLUDED.self_registration_enabled,
				email_verification_required = EXCLUDED.email_verification_required,
				forgot_password_enabled = EXCLUDED.forgot_password_enabled,
				remember_me_enabled = EXCLUDED.remember_me_enabled,
				login_page_title = EXCLUDED.login_page_title,
				login_page_message = EXCLUDED.login_page_message,
				updated_at = now()`,
			orgID,
			s.PasswordMinLength, s.PasswordRequireUppercase, s.PasswordRequireLowercase,
			s.PasswordRequireNumbers, s.PasswordRequireSymbols,
			s.MFAEnforcement, accessTTL, refreshTTL,
			s.SelfRegistrationEnabled, s.EmailVerificationRequired,
			s.ForgotPasswordEnabled, s.RememberMeEnabled,
			s.LoginPageTitle, s.LoginPageMessage,
		)
		if err != nil {
			return fmt.Errorf("upserting organization settings: %w", err)
		}
	}

	// Upsert roles and build name→id map for group role assignments
	roleIDs := make(map[string]uuid.UUID)
	for _, r := range export.Roles {
		var roleID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO roles (org_id, name, description)
			VALUES ($1, $2, $3)
			ON CONFLICT (org_id, name) DO UPDATE SET
				description = EXCLUDED.description,
				updated_at = now()
			RETURNING id`,
			orgID, r.Name, r.Description,
		).Scan(&roleID)
		if err != nil {
			return fmt.Errorf("upserting role %q: %w", r.Name, err)
		}
		roleIDs[r.Name] = roleID
	}

	// Upsert groups and assign roles
	for _, g := range export.Groups {
		var groupID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO groups (org_id, name, description)
			VALUES ($1, $2, $3)
			ON CONFLICT (org_id, name) DO UPDATE SET
				description = EXCLUDED.description,
				updated_at = now()
			RETURNING id`,
			orgID, g.Name, g.Description,
		).Scan(&groupID)
		if err != nil {
			return fmt.Errorf("upserting group %q: %w", g.Name, err)
		}

		for _, roleName := range g.Roles {
			roleID, ok := roleIDs[roleName]
			if !ok {
				return fmt.Errorf("group %q references unknown role %q", g.Name, roleName)
			}
			_, err = tx.Exec(ctx,
				"INSERT INTO group_roles (group_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
				groupID, roleID)
			if err != nil {
				return fmt.Errorf("assigning role %q to group %q: %w", roleName, g.Name, err)
			}
		}
	}

	// Upsert OAuth clients (no secrets — imported clients get no secret)
	for _, c := range export.Clients {
		clientID := c.ClientID
		if clientID == "" {
			clientID = generateClientID()
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO oauth_clients (id, org_id, name, description, client_type, redirect_uris, enabled)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				client_type = EXCLUDED.client_type,
				redirect_uris = EXCLUDED.redirect_uris,
				enabled = EXCLUDED.enabled,
				updated_at = now()`,
			clientID, orgID, c.Name, c.Description, c.ClientType, c.RedirectURIs, c.Enabled,
		)
		if err != nil {
			return fmt.Errorf("upserting oauth client %q: %w", c.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing import transaction: %w", err)
	}

	return nil
}
