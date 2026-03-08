// admin_console_saml.go contains admin console handlers for SAML provider management.
package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
)

// ListSAMLProvidersPage handles GET /admin/saml-providers
func (h *AdminConsoleHandler) ListSAMLProvidersPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)

	providers, err := h.store.ListSAMLProviders(ctx, authUser.OrgID)
	if err != nil {
		h.logger.Error("failed to list SAML providers", "error", err)
		h.render(w, r, "saml_providers_list", &pageData{Title: "SAML Providers", ActiveNav: navSAML, Error: "Failed to load SAML providers."})
		return
	}

	h.render(w, r, "saml_providers_list", &pageData{
		Title:         "SAML Providers",
		ActiveNav:     navSAML,
		SAMLProviders: providers,
	})
}

// CreateSAMLProviderPage handles GET /admin/saml-providers/new
func (h *AdminConsoleHandler) CreateSAMLProviderPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "saml_provider_create", &pageData{
		Title:     "Add SAML Provider",
		ActiveNav: navSAML,
	})
}

// CreateSAMLProviderAction handles POST /admin/saml-providers
func (h *AdminConsoleHandler) CreateSAMLProviderAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)

	if err := r.ParseForm(); err != nil {
		h.render(w, r, "saml_provider_create", &pageData{Title: "Add SAML Provider", ActiveNav: navSAML, Error: msgInvalidForm})
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	entityID := strings.TrimSpace(r.FormValue("entity_id"))
	ssoURL := strings.TrimSpace(r.FormValue("sso_url"))
	certificate := strings.TrimSpace(r.FormValue("certificate"))

	formErrors := map[string]string{}
	if name == "" {
		formErrors["name"] = "Name is required."
	}
	if entityID == "" {
		formErrors["entity_id"] = "Entity ID is required."
	}
	if ssoURL == "" {
		formErrors["sso_url"] = "SSO URL is required."
	}
	if certificate == "" {
		formErrors["certificate"] = "IdP Certificate is required."
	}
	if len(formErrors) > 0 {
		h.render(w, r, "saml_provider_create", &pageData{
			Title:      "Add SAML Provider",
			ActiveNav:  navSAML,
			FormErrors: formErrors,
			FormValues: map[string]string{
				"name": name, "entity_id": entityID, "sso_url": ssoURL,
				"certificate": certificate,
				"metadata_url": r.FormValue("metadata_url"),
				"slo_url":      r.FormValue("slo_url"),
			},
		})
		return
	}

	p, err := h.store.CreateSAMLProvider(ctx, &model.SAMLProvider{
		OrgID:        authUser.OrgID,
		Name:         name,
		EntityID:     entityID,
		MetadataURL:  strings.TrimSpace(r.FormValue("metadata_url")),
		SSOURL:       ssoURL,
		SLOURL:       strings.TrimSpace(r.FormValue("slo_url")),
		Certificate:  certificate,
		NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
		AttributeMapping: map[string]string{
			"email":       strings.TrimSpace(r.FormValue("attr_email")),
			"given_name":  strings.TrimSpace(r.FormValue("attr_given_name")),
			"family_name": strings.TrimSpace(r.FormValue("attr_family_name")),
			"username":    strings.TrimSpace(r.FormValue("attr_username")),
		},
		Enabled: true,
	})
	if err != nil {
		h.logger.Error("failed to create SAML provider", "error", err)
		h.render(w, r, "saml_provider_create", &pageData{Title: "Add SAML Provider", ActiveNav: navSAML, Error: "Failed to create SAML provider."})
		return
	}

	h.auditLog(r, authUser.OrgID, "saml_provider.created", "saml_provider", p.ID.String(), p.Name)
	middleware.SetFlash(w, "SAML provider created.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminSAMLFmt, p.ID), http.StatusSeeOther)
}

// SAMLProviderDetailPage handles GET /admin/saml-providers/{id}
func (h *AdminConsoleHandler) SAMLProviderDetailPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminSAMLProviders, http.StatusSeeOther)
		return
	}

	p, err := h.store.GetSAMLProviderByID(ctx, id)
	if err != nil || p == nil {
		middleware.SetFlash(w, "SAML provider not found.")
		http.Redirect(w, r, pathAdminSAMLProviders, http.StatusSeeOther)
		return
	}

	h.render(w, r, "saml_provider_detail", &pageData{
		Title:     "SAML: " + p.Name,
		ActiveNav: navSAML,
		SAMLDetail: p,
	})
}

// UpdateSAMLProviderAction handles POST /admin/saml-providers/{id}
func (h *AdminConsoleHandler) UpdateSAMLProviderAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminSAMLProviders, http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, msgInvalidForm)
		http.Redirect(w, r, fmt.Sprintf(pathAdminSAMLFmt, id), http.StatusSeeOther)
		return
	}

	enabled := r.FormValue("enabled") == formValueTrue

	_, err = h.store.UpdateSAMLProvider(ctx, id, &model.UpdateSAMLProviderRequest{
		Name:         strings.TrimSpace(r.FormValue("name")),
		EntityID:     strings.TrimSpace(r.FormValue("entity_id")),
		MetadataURL:  strings.TrimSpace(r.FormValue("metadata_url")),
		SSOURL:       strings.TrimSpace(r.FormValue("sso_url")),
		SLOURL:       strings.TrimSpace(r.FormValue("slo_url")),
		Certificate:  strings.TrimSpace(r.FormValue("certificate")),
		NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
		AttributeMapping: map[string]string{
			"email":       strings.TrimSpace(r.FormValue("attr_email")),
			"given_name":  strings.TrimSpace(r.FormValue("attr_given_name")),
			"family_name": strings.TrimSpace(r.FormValue("attr_family_name")),
			"username":    strings.TrimSpace(r.FormValue("attr_username")),
		},
		Enabled: enabled,
	})
	if err != nil {
		h.logger.Error("failed to update SAML provider", "error", err)
		middleware.SetFlash(w, "Failed to update SAML provider.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminSAMLFmt, id), http.StatusSeeOther)
		return
	}

	h.auditLog(r, authUser.OrgID, "saml_provider.updated", "saml_provider", id.String(), "")
	middleware.SetFlash(w, "SAML provider updated.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminSAMLFmt, id), http.StatusSeeOther)
}

// DeleteSAMLProviderAction handles POST /admin/saml-providers/{id}/delete
func (h *AdminConsoleHandler) DeleteSAMLProviderAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminSAMLProviders, http.StatusSeeOther)
		return
	}

	if err := h.store.DeleteSAMLProvider(ctx, id); err != nil {
		h.logger.Error("failed to delete SAML provider", "error", err)
		middleware.SetFlash(w, "Failed to delete SAML provider.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminSAMLFmt, id), http.StatusSeeOther)
		return
	}

	h.auditLog(r, authUser.OrgID, "saml_provider.deleted", "saml_provider", id.String(), "")
	middleware.SetFlash(w, "SAML provider deleted.")
	http.Redirect(w, r, pathAdminSAMLProviders, http.StatusSeeOther)
}
