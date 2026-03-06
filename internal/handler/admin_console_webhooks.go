package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
)

// Webhook nav and path constants.
const (
	navWebhooks        = "webhooks"
	pathAdminWebhooks  = "/admin/webhooks"
	titleWebhooks      = "Webhooks"
	titleCreateWebhook = "Create Webhook"
	tmplWebhooksList   = "webhooks_list"
	tmplWebhookCreate  = "webhook_create"
	tmplWebhookDetail  = "webhook_detail"
)

// WebhookStore defines the database operations required by webhook handlers.
type WebhookStore interface {
	ListWebhooks(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*model.Webhook, int, error)
	GetWebhookByID(ctx context.Context, id uuid.UUID) (*model.Webhook, error)
	CreateWebhook(ctx context.Context, webhook *model.Webhook) (*model.Webhook, error)
	DeleteWebhook(ctx context.Context, id uuid.UUID) error
	UpdateWebhookEnabled(ctx context.Context, id uuid.UUID, enabled bool) error
	ListWebhookDeliveries(ctx context.Context, webhookID uuid.UUID, limit, offset int) ([]*model.WebhookDelivery, int, error)
}

// webhookEvents lists the available event types for webhook subscriptions.
var webhookEvents = []string{
	"user.login",
	"user.created",
	"user.updated",
	"user.deleted",
	"user.login_failed",
	"role.assigned",
	"session.revoked",
	"client.created",
	"client.deleted",
	"social.login",
}

// WebhooksPage renders the webhooks management page.
func (h *AdminConsoleHandler) WebhooksPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	page := queryInt(r, "page", 1)
	limit := 20
	offset := (page - 1) * limit

	webhookStore := h.webhookStore
	if webhookStore == nil {
		h.render(w, r, tmplWebhooksList, &pageData{
			Title:     titleWebhooks,
			ActiveNav: navWebhooks,
			Error:     "Webhook storage not configured.",
		})
		return
	}

	webhooks, total, err := webhookStore.ListWebhooks(ctx, orgID, limit, offset)
	if err != nil {
		h.logger.Error("failed to list webhooks", "error", err)
		h.render(w, r, tmplWebhooksList, &pageData{
			Title:     titleWebhooks,
			ActiveNav: navWebhooks,
			Error:     "Failed to load webhooks.",
		})
		return
	}

	responses := make([]*model.WebhookResponse, len(webhooks))
	for i, wh := range webhooks {
		responses[i] = wh.ToResponse()
	}

	pg := buildPagination(page, limit, total, pathAdminWebhooks, "")

	if r.Header.Get(headerHXRequest) == formValueTrue {
		h.renderPartial(w, r, tmplWebhooksList, "webhooks_table", &pageData{
			Webhooks:   responses,
			Pagination: pg,
		})
		return
	}

	h.render(w, r, tmplWebhooksList, &pageData{
		Title:      titleWebhooks,
		ActiveNav:  navWebhooks,
		Webhooks:   responses,
		Pagination: pg,
	})
}

// WebhookDetailPage renders a single webhook detail page with delivery history.
func (h *AdminConsoleHandler) WebhookDetailPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	webhookID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	webhookStore := h.webhookStore
	if webhookStore == nil {
		h.render(w, r, tmplWebhookDetail, &pageData{
			Title:     titleWebhooks,
			ActiveNav: navWebhooks,
			Error:     "Webhook storage not configured.",
		})
		return
	}

	wh, err := webhookStore.GetWebhookByID(ctx, webhookID)
	if err != nil {
		h.logger.Error("failed to get webhook", "error", err)
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	deliveries, _, err := webhookStore.ListWebhookDeliveries(ctx, webhookID, 20, 0)
	if err != nil {
		h.logger.Error("failed to list webhook deliveries", "error", err)
		deliveries = nil
	}

	h.render(w, r, tmplWebhookDetail, &pageData{
		Title:              fmt.Sprintf("Webhook: %s", wh.URL),
		ActiveNav:          navWebhooks,
		WebhookDetail:      wh.ToResponse(),
		WebhookDeliveries:  deliveries,
		WebhookEvents:      wh.Events,
	})
}

// WebhookCreatePage renders the create webhook form.
func (h *AdminConsoleHandler) WebhookCreatePage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, tmplWebhookCreate, &pageData{
		Title:              titleCreateWebhook,
		ActiveNav:          navWebhooks,
		AvailableEvents:    webhookEvents,
	})
}

// WebhookCreate handles POST to create a webhook.
func (h *AdminConsoleHandler) WebhookCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, tmplWebhookCreate, &pageData{
			Title:           titleCreateWebhook,
			ActiveNav:       navWebhooks,
			AvailableEvents: webhookEvents,
			Error:           msgInvalidForm,
		})
		return
	}

	ctx := r.Context()
	authUser := middleware.GetAuthenticatedUser(ctx)
	orgID := authUser.OrgID

	webhookURL := strings.TrimSpace(r.FormValue("url"))
	description := strings.TrimSpace(r.FormValue("description"))
	selectedEvents := r.Form["events"]

	if webhookURL == "" {
		h.render(w, r, tmplWebhookCreate, &pageData{
			Title:           titleCreateWebhook,
			ActiveNav:       navWebhooks,
			AvailableEvents: webhookEvents,
			Error:           "Webhook URL is required.",
		})
		return
	}

	if len(selectedEvents) == 0 {
		h.render(w, r, tmplWebhookCreate, &pageData{
			Title:           titleCreateWebhook,
			ActiveNav:       navWebhooks,
			AvailableEvents: webhookEvents,
			Error:           "At least one event must be selected.",
		})
		return
	}

	// Generate a random secret for HMAC signing.
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		h.logger.Error("failed to generate webhook secret", "error", err)
		h.render(w, r, tmplWebhookCreate, &pageData{
			Title:           titleCreateWebhook,
			ActiveNav:       navWebhooks,
			AvailableEvents: webhookEvents,
			Error:           msgInternalErr,
		})
		return
	}
	secret := hex.EncodeToString(secretBytes)

	webhookStore := h.webhookStore
	if webhookStore == nil {
		h.render(w, r, tmplWebhookCreate, &pageData{
			Title:           titleCreateWebhook,
			ActiveNav:       navWebhooks,
			AvailableEvents: webhookEvents,
			Error:           "Webhook storage not configured.",
		})
		return
	}

	now := time.Now().UTC()
	wh := &model.Webhook{
		ID:          uuid.New(),
		OrgID:       orgID,
		URL:         webhookURL,
		Secret:      secret,
		Events:      selectedEvents,
		Enabled:     true,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := webhookStore.CreateWebhook(ctx, wh)
	if err != nil {
		h.logger.Error("failed to create webhook", "error", err)
		h.render(w, r, tmplWebhookCreate, &pageData{
			Title:           titleCreateWebhook,
			ActiveNav:       navWebhooks,
			AvailableEvents: webhookEvents,
			Error:           "Failed to create webhook.",
		})
		return
	}

	middleware.SetFlash(w, "Webhook created successfully. Secret: "+created.Secret)
	http.Redirect(w, r, fmt.Sprintf("/admin/webhooks/%s", created.ID), http.StatusSeeOther)
}

// WebhookDelete handles POST to delete a webhook.
func (h *AdminConsoleHandler) WebhookDelete(w http.ResponseWriter, r *http.Request) {
	webhookID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	webhookStore := h.webhookStore
	if webhookStore == nil {
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	if err := webhookStore.DeleteWebhook(r.Context(), webhookID); err != nil {
		h.logger.Error("failed to delete webhook", "error", err)
	}

	middleware.SetFlash(w, "Webhook deleted.")
	http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
}

// WebhookToggle handles POST to enable/disable a webhook.
func (h *AdminConsoleHandler) WebhookToggle(w http.ResponseWriter, r *http.Request) {
	webhookID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	webhookStore := h.webhookStore
	if webhookStore == nil {
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	ctx := r.Context()
	wh, err := webhookStore.GetWebhookByID(ctx, webhookID)
	if err != nil {
		h.logger.Error("failed to get webhook for toggle", "error", err)
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	newEnabled := !wh.Enabled
	if err := webhookStore.UpdateWebhookEnabled(ctx, webhookID, newEnabled); err != nil {
		h.logger.Error("failed to toggle webhook", "error", err)
	}

	status := "disabled"
	if newEnabled {
		status = "enabled"
	}
	middleware.SetFlash(w, fmt.Sprintf("Webhook %s.", status))
	http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
}

// WebhookTest handles POST to send a test event.
func (h *AdminConsoleHandler) WebhookTest(w http.ResponseWriter, r *http.Request) {
	webhookID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	webhookStore := h.webhookStore
	if webhookStore == nil {
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	wh, err := webhookStore.GetWebhookByID(r.Context(), webhookID)
	if err != nil {
		h.logger.Error("failed to get webhook for test", "error", err)
		http.Redirect(w, r, pathAdminWebhooks, http.StatusSeeOther)
		return
	}

	// For now, just flash a message. Actual delivery will be implemented with the webhook dispatcher.
	middleware.SetFlash(w, fmt.Sprintf("Test event queued for %s.", wh.URL))
	http.Redirect(w, r, fmt.Sprintf("/admin/webhooks/%s", webhookID), http.StatusSeeOther)
}
