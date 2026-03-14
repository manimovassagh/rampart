// Package webhook provides event delivery to registered webhook endpoints.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

const (
	maxAttempts     = 5
	deliveryTimeout = 10 * time.Second
	signatureHeader = "X-Rampart-Signature-256"
	eventHeader     = "X-Rampart-Event"
	deliveryHeader  = "X-Rampart-Delivery"
)

// Store defines the database operations needed by the dispatcher.
type Store interface {
	GetEnabledWebhooksForEvent(ctx context.Context, orgID uuid.UUID, eventType string) ([]*model.Webhook, error)
	CreateWebhookDelivery(ctx context.Context, d *model.WebhookDelivery) error
	UpdateWebhookDelivery(ctx context.Context, id uuid.UUID, status string, attempts int, responseCode *int, lastError string, nextRetry *time.Time, completedAt *time.Time) error
	GetPendingDeliveries(ctx context.Context, limit int) ([]*model.WebhookDelivery, error)
	GetWebhookByID(ctx context.Context, id uuid.UUID) (*model.Webhook, error)
	GetAuditEventByID(ctx context.Context, id uuid.UUID) (*model.AuditEvent, error)
}

// Dispatcher handles webhook event delivery.
type Dispatcher struct {
	store  Store
	client *http.Client
	logger *slog.Logger
}

// NewDispatcher creates a webhook dispatcher.
func NewDispatcher(s Store, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		store: s,
		client: &http.Client{
			Timeout: deliveryTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if err := ValidateWebhookURL(req.URL.String()); err != nil {
					return fmt.Errorf("redirect blocked: %w", err)
				}
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		logger: logger,
	}
}

// Dispatch enqueues webhook deliveries for an audit event.
// Called asynchronously after an audit event is persisted.
func (d *Dispatcher) Dispatch(ctx context.Context, event *model.AuditEvent) {
	webhooks, err := d.store.GetEnabledWebhooksForEvent(ctx, event.OrgID, event.EventType)
	if err != nil {
		d.logger.Error("failed to get webhooks for event", "event_type", event.EventType, "error", err)
		return
	}

	now := time.Now()
	for _, wh := range webhooks {
		delivery := &model.WebhookDelivery{
			WebhookID:   wh.ID,
			EventID:     event.ID,
			Status:      "pending",
			Attempts:    0,
			NextRetryAt: &now,
		}
		if err := d.store.CreateWebhookDelivery(ctx, delivery); err != nil {
			d.logger.Error("failed to create webhook delivery", "webhook_id", wh.ID, "event_id", event.ID, "error", err)
		}
	}
}

// ProcessPending processes pending deliveries (called by background worker).
func (d *Dispatcher) ProcessPending(ctx context.Context) {
	deliveries, err := d.store.GetPendingDeliveries(ctx, 50)
	if err != nil {
		d.logger.Error("failed to get pending deliveries", "error", err)
		return
	}

	for _, delivery := range deliveries {
		d.deliver(ctx, delivery)
	}
}

func (d *Dispatcher) deliver(ctx context.Context, delivery *model.WebhookDelivery) {
	wh, err := d.store.GetWebhookByID(ctx, delivery.WebhookID)
	if err != nil || wh == nil || !wh.Enabled {
		now := time.Now()
		_ = d.store.UpdateWebhookDelivery(ctx, delivery.ID, "failed", delivery.Attempts, nil, "webhook not found or disabled", nil, &now)
		return
	}

	event, err := d.store.GetAuditEventByID(ctx, delivery.EventID)
	if err != nil || event == nil {
		now := time.Now()
		_ = d.store.UpdateWebhookDelivery(ctx, delivery.ID, "failed", delivery.Attempts, nil, "event not found", nil, &now)
		return
	}

	payload := buildPayload(event)
	body, err := json.Marshal(payload)
	if err != nil {
		now := time.Now()
		_ = d.store.UpdateWebhookDelivery(ctx, delivery.ID, "failed", delivery.Attempts, nil, "marshal error", nil, &now)
		return
	}

	sig := sign(body, wh.Secret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		now := time.Now()
		_ = d.store.UpdateWebhookDelivery(ctx, delivery.ID, "failed", delivery.Attempts, nil, err.Error(), nil, &now)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(signatureHeader, "sha256="+sig)
	req.Header.Set(eventHeader, event.EventType)
	req.Header.Set(deliveryHeader, delivery.ID.String())

	resp, err := d.client.Do(req)
	attempts := delivery.Attempts + 1

	if err != nil {
		d.handleFailure(ctx, delivery, attempts, nil, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		now := time.Now()
		code := resp.StatusCode
		_ = d.store.UpdateWebhookDelivery(ctx, delivery.ID, "success", attempts, &code, "", nil, &now)
		return
	}

	code := resp.StatusCode
	d.handleFailure(ctx, delivery, attempts, &code, fmt.Sprintf("HTTP %d", resp.StatusCode))
}

func (d *Dispatcher) handleFailure(ctx context.Context, delivery *model.WebhookDelivery, attempts int, code *int, errMsg string) {
	if attempts >= maxAttempts {
		now := time.Now()
		_ = d.store.UpdateWebhookDelivery(ctx, delivery.ID, "failed", attempts, code, errMsg, nil, &now)
		d.logger.Warn("webhook delivery permanently failed", "delivery_id", delivery.ID, "attempts", attempts, "error", errMsg)
		return
	}

	// Exponential backoff: 10s, 40s, 90s, 160s
	backoff := time.Duration(attempts*attempts*10) * time.Second
	next := time.Now().Add(backoff)
	_ = d.store.UpdateWebhookDelivery(ctx, delivery.ID, "pending", attempts, code, errMsg, &next, nil)
}

func buildPayload(event *model.AuditEvent) *model.WebhookPayload {
	p := &model.WebhookPayload{
		ID:        event.ID.String(),
		Type:      event.EventType,
		Timestamp: event.CreatedAt.UTC().Format(time.RFC3339),
		OrgID:     event.OrgID.String(),
		Details:   event.Details,
	}

	if event.ActorID != nil || event.ActorName != "" {
		actor := &model.WebhookActor{Name: event.ActorName}
		if event.ActorID != nil {
			actor.ID = event.ActorID.String()
		}
		p.Actor = actor
	}

	if event.TargetType != "" || event.TargetID != "" {
		p.Target = &model.WebhookTarget{
			Type: event.TargetType,
			ID:   event.TargetID,
			Name: event.TargetName,
		}
	}

	return p
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
