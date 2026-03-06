package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/apierror"
	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/webhook"
)

const (
	hmacSecretBytes      = 32
	defaultDeliveryLimit = 50
)

// WebhookStore defines the database operations for webhook management.
type WebhookStore interface {
	CreateWebhook(ctx context.Context, wh *model.Webhook) (*model.Webhook, error)
	GetWebhook(ctx context.Context, id uuid.UUID) (*model.Webhook, error)
	ListWebhooks(ctx context.Context, orgID uuid.UUID) ([]*model.Webhook, error)
	UpdateWebhook(ctx context.Context, wh *model.Webhook) error
	DeleteWebhook(ctx context.Context, id uuid.UUID) error
	GetWebhookDeliveries(ctx context.Context, webhookID uuid.UUID, limit int) ([]*model.WebhookDelivery, error)
}

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

	webhooks, err := h.store.ListWebhooks(r.Context(), orgID)
	if err != nil {
		h.logger.Error("failed to list webhooks", "error", err)
		apierror.InternalError(w)
		return
	}

	// Mask secrets in list response.
	for _, wh := range webhooks {
		wh.Secret = maskSecret(wh.Secret)
	}

	writeJSON(w, http.StatusOK, webhooks, h.logger)
}

// Get handles GET /api/v1/admin/webhooks/{id}.
func (h *WebhookHandler) Get(w http.ResponseWriter, r *http.Request) {
	webhookID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	wh, err := h.store.GetWebhook(r.Context(), webhookID)
	if err != nil {
		h.logger.Error("failed to get webhook", "error", err)
		apierror.InternalError(w)
		return
	}
	if wh == nil {
		apierror.NotFound(w)
		return
	}

	wh.Secret = maskSecret(wh.Secret)
	writeJSON(w, http.StatusOK, wh, h.logger)
}

// Update handles PUT /api/v1/admin/webhooks/{id}.
func (h *WebhookHandler) Update(w http.ResponseWriter, r *http.Request) {
	webhookID, ok := parseUUIDParam(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req model.UpdateWebhookRequest
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

	existing, err := h.store.GetWebhook(r.Context(), webhookID)
	if err != nil {
		h.logger.Error("failed to get webhook", "error", err)
		apierror.InternalError(w)
		return
	}
	if existing == nil {
		apierror.NotFound(w)
		return
	}

	existing.URL = req.URL
	existing.Events = req.Events
	existing.Enabled = req.Enabled
	existing.Description = req.Description

	if err := h.store.UpdateWebhook(r.Context(), existing); err != nil {
		h.logger.Error("failed to update webhook", "error", err)
		apierror.InternalError(w)
		return
	}

	existing.Secret = maskSecret(existing.Secret)
	writeJSON(w, http.StatusOK, existing, h.logger)
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

	deliveries, err := h.store.GetWebhookDeliveries(r.Context(), webhookID, limit)
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

	wh, err := h.store.GetWebhook(r.Context(), webhookID)
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

func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "****" + secret[len(secret)-4:]
}
