// admin_console_export.go contains admin console handlers for organization
// export and import: ExportOrgAction, ImportOrgPage, ImportOrgAction.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/store"
)

// ExportOrgAction handles GET /admin/organizations/{id}/export — downloads org config as JSON.
func (h *AdminConsoleHandler) ExportOrgAction(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
		return
	}

	ctx := r.Context()
	org, err := h.store.GetOrganizationByID(ctx, orgID)
	if err != nil || org == nil {
		middleware.SetFlash(w, "Organization not found.")
		http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
		return
	}

	export := model.OrgExport{
		Organization: model.OrgExportData{
			Name:        org.Name,
			Slug:        org.Slug,
			DisplayName: org.DisplayName,
		},
	}

	if settings, err := h.store.GetOrgSettings(ctx, orgID); err == nil && settings != nil {
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

	if roles, _, err := h.store.ListRoles(ctx, orgID, "", 1000, 0); err == nil {
		for _, role := range roles {
			export.Roles = append(export.Roles, model.RoleExport{
				Name:        role.Name,
				Description: role.Description,
			})
		}
	}

	if groups, _, err := h.store.ListGroups(ctx, orgID, "", 1000, 0); err == nil {
		for _, g := range groups {
			ge := model.GroupExport{Name: g.Name, Description: g.Description}
			if groupRoles, err := h.store.GetGroupRoles(ctx, g.ID); err == nil {
				for _, gr := range groupRoles {
					ge.Roles = append(ge.Roles, gr.RoleName)
				}
			}
			export.Groups = append(export.Groups, ge)
		}
	}

	if clients, _, err := h.store.ListOAuthClients(ctx, orgID, "", 1000, 0); err == nil {
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
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		h.logger.Error("failed to marshal export", "error", err)
		middleware.SetFlash(w, "Failed to export organization.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", apierror.ContentTypeJSON)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="org-%s.json"`, org.Slug))
	if _, err := w.Write(data); err != nil {
		h.logger.Error("failed to write export response", "error", err)
	}
}

// ImportOrgPage handles GET /admin/organizations/import
func (h *AdminConsoleHandler) ImportOrgPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, tmplOrgImport, &pageData{Title: titleImportOrg, ActiveNav: navOrganizations})
}

// ImportOrgAction handles POST /admin/organizations/import
func (h *AdminConsoleHandler) ImportOrgAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.render(w, r, tmplOrgImport, &pageData{
			Title: titleImportOrg, ActiveNav: navOrganizations,
			Error: "Invalid form data. Max file size is 10MB.",
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		h.render(w, r, tmplOrgImport, &pageData{
			Title: titleImportOrg, ActiveNav: navOrganizations,
			Error: "Please select a JSON file to import.",
		})
		return
	}
	defer func() { _ = file.Close() }()

	var export model.OrgExport
	if err := json.NewDecoder(file).Decode(&export); err != nil {
		h.render(w, r, tmplOrgImport, &pageData{
			Title: titleImportOrg, ActiveNav: navOrganizations,
			Error: "Invalid JSON file format.",
		})
		return
	}

	// Validate import size limits to prevent DoS
	const maxImportItems = 100
	if len(export.Roles) > maxImportItems || len(export.Groups) > maxImportItems || len(export.Clients) > maxImportItems {
		h.render(w, r, tmplOrgImport, &pageData{
			Title: titleImportOrg, ActiveNav: navOrganizations,
			Error: fmt.Sprintf("Import exceeds maximum of %d items per category.", maxImportItems),
		})
		return
	}

	ctx := r.Context()

	// Create organization
	org, err := h.store.CreateOrganization(ctx, &model.CreateOrgRequest{
		Name:        export.Organization.Name,
		Slug:        export.Organization.Slug,
		DisplayName: export.Organization.DisplayName,
	})
	if err != nil {
		h.logger.Error("failed to import organization", "error", err)
		msg := "Failed to create organization."
		if errors.Is(err, store.ErrDuplicateKey) {
			msg = "An organization with this slug already exists."
		}
		h.render(w, r, tmplOrgImport, &pageData{
			Title: titleImportOrg, ActiveNav: navOrganizations,
			Error: msg,
		})
		return
	}

	// Import settings
	if export.Settings != nil {
		_, _ = h.store.UpdateOrgSettings(ctx, org.ID, &model.UpdateOrgSettingsRequest{
			PasswordMinLength:         export.Settings.PasswordMinLength,
			PasswordRequireUppercase:  export.Settings.PasswordRequireUppercase,
			PasswordRequireLowercase:  export.Settings.PasswordRequireLowercase,
			PasswordRequireNumbers:    export.Settings.PasswordRequireNumbers,
			PasswordRequireSymbols:    export.Settings.PasswordRequireSymbols,
			MFAEnforcement:            export.Settings.MFAEnforcement,
			AccessTokenTTLSeconds:     export.Settings.AccessTokenTTLSeconds,
			RefreshTokenTTLSeconds:    export.Settings.RefreshTokenTTLSeconds,
			SelfRegistrationEnabled:   export.Settings.SelfRegistrationEnabled,
			EmailVerificationRequired: export.Settings.EmailVerificationRequired,
			ForgotPasswordEnabled:     export.Settings.ForgotPasswordEnabled,
			RememberMeEnabled:         export.Settings.RememberMeEnabled,
			LoginPageTitle:            export.Settings.LoginPageTitle,
			LoginPageMessage:          export.Settings.LoginPageMessage,
			LoginTheme:                export.Settings.LoginTheme,
		})
	}

	// Import roles
	roleMap := make(map[string]uuid.UUID)
	for _, re := range export.Roles {
		role, err := h.store.CreateRole(ctx, &model.Role{
			OrgID:       org.ID,
			Name:        re.Name,
			Description: re.Description,
		})
		if err == nil {
			roleMap[role.Name] = role.ID
		}
	}

	// Import groups with role assignments
	for _, ge := range export.Groups {
		group, err := h.store.CreateGroup(ctx, &model.Group{
			OrgID:       org.ID,
			Name:        ge.Name,
			Description: ge.Description,
		})
		if err != nil {
			continue
		}
		for _, roleName := range ge.Roles {
			if roleID, ok := roleMap[roleName]; ok {
				_ = h.store.AssignRoleToGroup(ctx, group.ID, roleID)
			}
		}
	}

	middleware.SetFlash(w, fmt.Sprintf("Organization '%s' imported successfully.", org.Name))
	http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, org.ID), http.StatusFound)
}
