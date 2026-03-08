// admin_console_orgs.go contains admin console handlers for organization management:
// ListOrgsPage, CreateOrgPage, CreateOrgAction, OrgDetailPage,
// UpdateOrgAction, UpdateOrgSettingsAction, DeleteOrgAction.
package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
)

// ListOrgsPage handles GET /admin/organizations
func (h *AdminConsoleHandler) ListOrgsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	search := r.URL.Query().Get("search")
	page := queryInt(r, "page", 1)
	limit := 20
	offset := (page - 1) * limit

	orgs, total, err := h.store.ListOrganizations(ctx, search, limit, offset)
	if err != nil {
		h.logger.Error("failed to list organizations", "error", err)
		h.render(w, r, "orgs_list", &pageData{Title: "Organizations", ActiveNav: navOrganizations, Error: "Failed to load organizations."})
		return
	}

	orgResponses := make([]*model.OrgResponse, len(orgs))
	for i, o := range orgs {
		count, _ := h.store.CountUsers(ctx, o.ID)
		orgResponses[i] = o.ToOrgResponse(count)
	}

	pg := buildPagination(page, limit, total, pathAdminOrgs, search)

	if r.Header.Get(headerHXRequest) == formValueTrue {
		h.renderPartial(w, r, "orgs_list", "orgs_table", &pageData{Orgs: orgResponses, Search: search, Pagination: pg})
		return
	}

	h.render(w, r, "orgs_list", &pageData{
		Title:      "Organizations",
		ActiveNav:  "organizations",
		Orgs:       orgResponses,
		Search:     search,
		Pagination: pg,
	})
}

// CreateOrgPage handles GET /admin/organizations/new
func (h *AdminConsoleHandler) CreateOrgPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, tmplOrgCreate, &pageData{Title: titleCreateOrg, ActiveNav: navOrganizations})
}

// CreateOrgAction handles POST /admin/organizations
func (h *AdminConsoleHandler) CreateOrgAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, tmplOrgCreate, &pageData{Title: titleCreateOrg, ActiveNav: navOrganizations, Error: msgInvalidForm})
		return
	}

	req := &model.CreateOrgRequest{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Slug:        strings.ToLower(strings.TrimSpace(r.FormValue("slug"))),
		DisplayName: strings.TrimSpace(r.FormValue("display_name")),
	}

	if req.Name == "" || req.Slug == "" {
		h.render(w, r, tmplOrgCreate, &pageData{Title: titleCreateOrg, ActiveNav: navOrganizations, Error: "Name and slug are required."})
		return
	}

	newOrg, err := h.store.CreateOrganization(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), msgDuplicateKey) || strings.Contains(err.Error(), "unique") {
			h.render(w, r, tmplOrgCreate, &pageData{Title: titleCreateOrg, ActiveNav: navOrganizations, Error: "An organization with this slug already exists."})
			return
		}
		h.logger.Error("failed to create organization", "error", err)
		h.render(w, r, tmplOrgCreate, &pageData{Title: titleCreateOrg, ActiveNav: navOrganizations, Error: "Failed to create organization."})
		return
	}

	orgAuthUser := middleware.GetAuthenticatedUser(r.Context())
	h.auditLog(r, orgAuthUser.OrgID, model.EventOrgCreated, "organization", newOrg.ID.String(), req.Name)
	middleware.SetFlash(w, "Organization created successfully.")
	http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
}

// OrgDetailPage handles GET /admin/organizations/{id}
func (h *AdminConsoleHandler) OrgDetailPage(w http.ResponseWriter, r *http.Request) {
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

	var settingsResp *model.OrgSettingsResponse
	if settings, sErr := h.store.GetOrgSettings(ctx, orgID); sErr == nil && settings != nil {
		settingsResp = settings.ToResponse()
	}

	h.render(w, r, "org_detail", &pageData{
		Title:       fmt.Sprintf("Organization: %s", org.Name),
		ActiveNav:   "organizations",
		OrgDetail:   org,
		OrgSettings: settingsResp,
	})
}

// UpdateOrgAction handles POST /admin/organizations/{id}
func (h *AdminConsoleHandler) UpdateOrgAction(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
		return
	}

	req := &model.UpdateOrgRequest{
		Name:        strings.TrimSpace(r.FormValue("name")),
		DisplayName: strings.TrimSpace(r.FormValue("display_name")),
		Enabled:     r.FormValue("enabled") == formValueTrue,
	}

	if _, err := h.store.UpdateOrganization(r.Context(), orgID, req); err != nil {
		h.logger.Error("failed to update organization", "error", err)
		middleware.SetFlash(w, "Failed to update organization.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Organization updated successfully.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
}

// UpdateOrgSettingsAction handles POST /admin/organizations/{id}/settings
func (h *AdminConsoleHandler) UpdateOrgSettingsAction(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "Invalid form data.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
		return
	}

	accessTTL, _ := strconv.Atoi(r.FormValue("access_token_ttl_seconds"))
	refreshTTL, _ := strconv.Atoi(r.FormValue("refresh_token_ttl_seconds"))
	minLen, _ := strconv.Atoi(r.FormValue("password_min_length"))

	if minLen < 1 {
		minLen = 1
	}
	if accessTTL < 60 {
		accessTTL = 60
	}
	if refreshTTL < 60 {
		refreshTTL = 60
	}

	mfa := r.FormValue("mfa_enforcement")
	if mfa != mfaOff && mfa != mfaOptional && mfa != mfaRequired {
		mfa = mfaOff
	}

	req := &model.UpdateOrgSettingsRequest{
		PasswordMinLength:         minLen,
		PasswordRequireUppercase:  r.FormValue("password_require_uppercase") == formValueTrue,
		PasswordRequireLowercase:  r.FormValue("password_require_lowercase") == formValueTrue,
		PasswordRequireNumbers:    r.FormValue("password_require_numbers") == formValueTrue,
		PasswordRequireSymbols:    r.FormValue("password_require_symbols") == formValueTrue,
		MFAEnforcement:            mfa,
		AccessTokenTTLSeconds:     accessTTL,
		RefreshTokenTTLSeconds:    refreshTTL,
		SelfRegistrationEnabled:   r.FormValue("self_registration_enabled") == formValueTrue,
		EmailVerificationRequired: r.FormValue("email_verification_required") == formValueTrue,
		ForgotPasswordEnabled:     r.FormValue("forgot_password_enabled") == formValueTrue,
		RememberMeEnabled:         r.FormValue("remember_me_enabled") == formValueTrue,
		LoginPageTitle:            strings.TrimSpace(r.FormValue("login_page_title")),
		LoginPageMessage:          strings.TrimSpace(r.FormValue("login_page_message")),
		LoginTheme:                strings.TrimSpace(r.FormValue("login_theme")),
	}

	if _, err := h.store.UpdateOrgSettings(r.Context(), orgID, req); err != nil {
		h.logger.Error("failed to update org settings", "error", err)
		middleware.SetFlash(w, "Failed to update settings.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Settings updated successfully.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
}

// DeleteOrgAction handles POST /admin/organizations/{id}/delete
func (h *AdminConsoleHandler) DeleteOrgAction(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
		return
	}

	if err := h.store.DeleteOrganization(r.Context(), orgID); err != nil {
		if strings.Contains(err.Error(), "default") {
			middleware.SetFlash(w, "Cannot delete the default organization.")
		} else {
			h.logger.Error("failed to delete organization", "error", err)
			middleware.SetFlash(w, "Failed to delete organization.")
		}
		http.Redirect(w, r, fmt.Sprintf(pathAdminOrgFmt, orgID), http.StatusFound)
		return
	}

	middleware.SetFlash(w, "Organization deleted.")
	http.Redirect(w, r, pathAdminOrgs, http.StatusFound)
}
