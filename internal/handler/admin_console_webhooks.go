// admin_console_webhooks.go contains admin console handlers for webhook management:
// ListWebhooksPage, CreateWebhookPage, CreateWebhookAction,
// WebhookDetailPage, UpdateWebhookAction, DeleteWebhookAction.
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
)

// ListWebhooksPage handles GET /admin/webhooks
func (h *AdminConsoleHandler) ListWebhooksPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	page := queryInt(r, "page", 1)
	limit := 50
	offset := (page - 1) * limit

	webhooks, total, err := h.store.ListWebhooks(ctx, orgID, limit, offset)
	if err != nil {
		h.logger.Error("failed to list webhooks", "error", err)
		h.render(w, r, "webhooks_list", &pageData{Title: "Webhooks", ActiveNav: navWebhooks, Error: "Failed to load webhooks."})
		return
	}

	pg := buildPagination(page, limit, total, pathAdminWebhooks, "")

	if r.Header.Get(headerHXRequest) == formValueTrue {
		h.renderPartial(w, r, "webhooks_list", "webhooks_table", &pageData{Webhooks: webhooks, Pagination: pg})
		return
	}

	h.render(w, r, "webhooks_list", &pageData{
		Title:      "Webhooks",
		ActiveNav:  navWebhooks,
		Webhooks:   webhooks,
		Pagination: pg,
	})
}

// CreateWebhookPage handles GET /admin/webhooks/new
func (h *AdminConsoleHandler) CreateWebhookPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "webhook_create", &pageData{
		Title:     "Create Webhook",
		ActiveNav: navWebhooks,
	})
}

// CreateWebhookAction handles POST /admin/webhooks
func (h *AdminConsoleHandler) CreateWebhookAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)

	if err := r.ParseForm(); err != nil {
		h.render(w, r, "webhook_create", &pageData{Title: "Create Webhook", ActiveNav: navWebhooks, Error: msgInvalidForm})
		return
	}

	whURL := strings.TrimSpace(r.FormValue("url"))
	description := strings.TrimSpace(r.FormValue("description"))
	eventTypesRaw := strings.TrimSpace(r.FormValue("event_types"))

	formErrors := map[string]string{}
	if whURL == "" {
		formErrors["url"] = "URL is required."
	}
	if eventTypesRaw == "" {
		formErrors["event_types"] = "At least one event type is required."
	}
	if len(formErrors) > 0 {
		h.render(w, r, "webhook_create", &pageData{
			Title:      "Create Webhook",
			ActiveNav:  navWebhooks,
			FormErrors: formErrors,
			FormValues: map[string]string{"url": whURL, "description": description, "event_types": eventTypesRaw},
		})
		return
	}

	// Parse comma-separated event types
	var eventTypes []string
	for _, et := range strings.Split(eventTypesRaw, ",") {
		et = strings.TrimSpace(et)
		if et != "" {
			eventTypes = append(eventTypes, et)
		}
	}

	// Generate a random secret
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		h.render(w, r, "webhook_create", &pageData{Title: "Create Webhook", ActiveNav: navWebhooks, Error: msgInternalErr})
		return
	}
	secret := hex.EncodeToString(secretBytes)

	wh, err := h.store.CreateWebhook(ctx, &model.Webhook{
		OrgID:       authUser.OrgID,
		URL:         whURL,
		Secret:      secret,
		Description: description,
		EventTypes:  eventTypes,
		Enabled:     true,
	})
	if err != nil {
		h.logger.Error("failed to create webhook", "error", err)
		h.render(w, r, "webhook_create", &pageData{Title: "Create Webhook", ActiveNav: navWebhooks, Error: "Failed to create webhook."})
		return
	}

	h.auditLog(r, authUser.OrgID, "webhook.created", "webhook", wh.ID.String(), wh.URL)

	middleware.SetFlash(w, "Webhook created. Copy the secret now — it will not be shown again.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminWebhookFmt, wh.ID)+"?show_secret=1", http.StatusSeeOther)
}

// WebhookDetailPage handles GET /admin/webhooks/{id}
func (h *AdminConsoleHandler) WebhookDetailPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	authUser := middleware.GetAuthenticatedUser(ctx)
	wh, err := h.store.GetWebhookByID(ctx, id)
	if err != nil || wh == nil || wh.OrgID != authUser.OrgID {
		middleware.SetFlash(w, "Webhook not found.")
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	// Fetch recent deliveries
	deliveries, _, err := h.store.ListWebhookDeliveries(ctx, id, 20, 0)
	if err != nil {
		h.logger.Error("failed to list deliveries", "error", err)
	}

	data := &pageData{
		Title:         "Webhook: " + wh.Description,
		ActiveNav:     navWebhooks,
		WebhookDetail: wh,
		Deliveries:    deliveries,
	}

	// Show secret only right after creation
	if r.URL.Query().Get("show_secret") == "1" {
		data.WebhookSecret = wh.Secret
	}

	h.render(w, r, "webhook_detail", data)
}

// UpdateWebhookAction handles POST /admin/webhooks/{id}
func (h *AdminConsoleHandler) UpdateWebhookAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	wh, err := h.store.GetWebhookByID(ctx, id)
	if err != nil || wh == nil || wh.OrgID != authUser.OrgID {
		middleware.SetFlash(w, "Webhook not found.")
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, msgInvalidForm)
		http.Redirect(w, r, fmt.Sprintf(pathAdminWebhookFmt, id), http.StatusSeeOther)
		return
	}

	whURL := strings.TrimSpace(r.FormValue("url"))
	description := strings.TrimSpace(r.FormValue("description"))
	eventTypesRaw := strings.TrimSpace(r.FormValue("event_types"))
	enabled := r.FormValue("enabled") == formValueTrue

	var eventTypes []string
	for _, et := range strings.Split(eventTypesRaw, ",") {
		et = strings.TrimSpace(et)
		if et != "" {
			eventTypes = append(eventTypes, et)
		}
	}

	_, err = h.store.UpdateWebhook(ctx, id, &model.UpdateWebhookRequest{
		URL:         whURL,
		Description: description,
		EventTypes:  eventTypes,
		Enabled:     enabled,
	})
	if err != nil {
		h.logger.Error("failed to update webhook", "error", err)
		middleware.SetFlash(w, "Failed to update webhook.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminWebhookFmt, id), http.StatusSeeOther)
		return
	}

	h.auditLog(r, authUser.OrgID, "webhook.updated", "webhook", id.String(), whURL)
	middleware.SetFlash(w, "Webhook updated.")
	http.Redirect(w, r, fmt.Sprintf(pathAdminWebhookFmt, id), http.StatusSeeOther)
}

// DeleteWebhookAction handles POST /admin/webhooks/{id}/delete
func (h *AdminConsoleHandler) DeleteWebhookAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	wh, err := h.store.GetWebhookByID(ctx, id)
	if err != nil || wh == nil || wh.OrgID != authUser.OrgID {
		middleware.SetFlash(w, "Webhook not found.")
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	if err := h.store.DeleteWebhook(ctx, id); err != nil {
		h.logger.Error("failed to delete webhook", "error", err)
		middleware.SetFlash(w, "Failed to delete webhook.")
		http.Redirect(w, r, fmt.Sprintf(pathAdminWebhookFmt, id), http.StatusSeeOther)
		return
	}

	h.auditLog(r, authUser.OrgID, "webhook.deleted", "webhook", id.String(), "")
	middleware.SetFlash(w, "Webhook deleted.")
	http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
}
