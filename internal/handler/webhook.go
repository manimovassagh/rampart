package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/webhook"
)

const (
	hmacSecretBytes      = 32
	defaultDeliveryLimit = 50
)

// WebhookHandler handles admin webhook API endpoints.
type WebhookHandler struct {
	store      WebhookStore
	dispatcher *webhook.Dispatcher
	logger     *slog.Logger
}

// NewWebhookHandler creates a handler with webhook dependencies.
func NewWebhookHandler(store WebhookStore, dispatcher *webhook.Dispatcher, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{store: store, dispatcher: dispatcher, logger: logger}
}

// Create handles POST /api/v1/admin/webhooks.
func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	authUser := middleware.GetAuthenticatedUser(r.Context())
	if authUser == nil {
		apierror.Unauthorized(w, msgAuthRequired)
		return
	}
	orgID := resolveOrgID(r, authUser)

	var req model.CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	if req.URL == "" {
		apierror.BadRequest(w, "Webhook URL is required.")
		return
	}
	if len(req.Events) == 0 {
		apierror.BadRequest(w, "At least one event type is required.")
		return
	}

	secret, err := generateHMACSecret()
	if err != nil {
		h.logger.Error("failed to generate HMAC secret", "error", err)
		apierror.InternalError(w)
		return
	}

	wh := &model.Webhook{
		OrgID:       orgID,
		URL:         req.URL,
		Secret:      secret,
		Events:      req.Events,
		Enabled:     req.Enabled,
		Description: req.Description,
	}

	created, err := h.store.CreateWebhook(r.Context(), wh)
	if err != nil {
		h.logger.Error("failed to create webhook", "error", err)
		apierror.InternalError(w)
		return
	}

	writeJSON(w, http.StatusCreated, created, h.logger)
}

// List handles GET /api/v1/admin/webhooks.
func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	authUser := middleware.GetAuthenticatedUser(r.Context())
	if authUser == nil {
		apierror.Unauthorized(w, msgAuthRequired)
		return
	}
	orgID := resolveOrgID(r, authUser)

	webhooks, _, err := h.store.ListWebhooks(r.Context(), orgID, maxPageLimit, 0)
	if err != nil {
		h.logger.Error("failed to list webhooks", "error", err)
		apierror.InternalError(w)
		return
	}

	// Return responses without secrets.
	responses := make([]*model.WebhookResponse, len(webhooks))
	for i, wh := range webhooks {
		responses[i] = wh.ToResponse()
	}

	writeJSON(w, http.StatusOK, responses, h.logger)
}

// Get handles GET /api/v1/admin/webhooks/{id}.
func (h *WebhookHandler) Get(w http.ResponseWriter, r *http.Request) {
	webhookID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	wh, err := h.store.GetWebhookByID(r.Context(), webhookID)
	if err != nil {
		h.logger.Error("failed to get webhook", "error", err)
		apierror.InternalError(w)
		return
	}
	if wh == nil {
		apierror.NotFound(w)
		return
	}

	writeJSON(w, http.StatusOK, wh.ToResponse(), h.logger)
}

// Update handles PUT /api/v1/admin/webhooks/{id} (toggle enabled).
func (h *WebhookHandler) Update(w http.ResponseWriter, r *http.Request) {
	webhookID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, msgInvalidJSON)
		return
	}

	if err := h.store.UpdateWebhookEnabled(r.Context(), webhookID, req.Enabled); err != nil {
		h.logger.Error("failed to update webhook", "error", err)
		apierror.InternalError(w)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"enabled": req.Enabled}, h.logger)
}

// Delete handles DELETE /api/v1/admin/webhooks/{id}.
func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	webhookID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	if err := h.store.DeleteWebhook(r.Context(), webhookID); err != nil {
		h.logger.Error("failed to delete webhook", "error", err)
		apierror.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Deliveries handles GET /api/v1/admin/webhooks/{id}/deliveries.
func (h *WebhookHandler) Deliveries(w http.ResponseWriter, r *http.Request) {
	webhookID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	limit := queryInt(r, "limit", defaultDeliveryLimit)
	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	deliveries, _, err := h.store.ListWebhookDeliveries(r.Context(), webhookID, limit, 0)
	if err != nil {
		h.logger.Error("failed to get webhook deliveries", "error", err)
		apierror.InternalError(w)
		return
	}

	writeJSON(w, http.StatusOK, deliveries, h.logger)
}

// Test handles POST /api/v1/admin/webhooks/{id}/test.
func (h *WebhookHandler) Test(w http.ResponseWriter, r *http.Request) {
	webhookID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	wh, err := h.store.GetWebhookByID(r.Context(), webhookID)
	if err != nil {
		h.logger.Error("failed to get webhook", "error", err)
		apierror.InternalError(w)
		return
	}
	if wh == nil {
		apierror.NotFound(w)
		return
	}

	testData := map[string]string{
		"message": "This is a test webhook delivery from Rampart.",
	}

	if dispatchErr := h.dispatcher.Dispatch(r.Context(), wh.OrgID, "webhook.test", testData); dispatchErr != nil {
		h.logger.Warn("test webhook delivery failed", "webhook_id", webhookID, "error", dispatchErr)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"}, h.logger)
}

func generateHMACSecret() (string, error) {
	b := make([]byte, hmacSecretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}
