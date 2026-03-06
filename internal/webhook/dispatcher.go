package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// Delivery and retry constants.
const (
	deliveryTimeout = 10 * time.Second
	maxAttempts     = 5
	maxResponseBody = 4096

	headerSignature = "X-Rampart-Signature"
	headerEvent     = "X-Rampart-Event"
	headerDelivery  = "X-Rampart-Delivery"
)

// retryDelays defines exponential backoff intervals per attempt.
var retryDelays = [maxAttempts]time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	12 * time.Hour,
}

// Store defines the database operations required by the Dispatcher.
type Store interface {
	GetWebhooksForEvent(ctx context.Context, orgID uuid.UUID, eventType string) ([]*model.Webhook, error)
	GetWebhook(ctx context.Context, id uuid.UUID) (*model.Webhook, error)
	CreateWebhookDelivery(ctx context.Context, delivery *model.WebhookDelivery) error
	GetPendingRetries(ctx context.Context, limit int) ([]*model.WebhookDelivery, error)
	UpdateWebhookDelivery(ctx context.Context, delivery *model.WebhookDelivery) error
}

// Dispatcher handles webhook event delivery and retries.
type Dispatcher struct {
	store  Store
	logger *slog.Logger
	client *http.Client
}

// NewDispatcher creates a new webhook dispatcher.
func NewDispatcher(store Store, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		store:  store,
		logger: logger,
		client: &http.Client{Timeout: deliveryTimeout},
	}
}

// Dispatch sends an event to all webhooks subscribed to the given event type.
func (d *Dispatcher) Dispatch(ctx context.Context, orgID uuid.UUID, eventType string, data any) error {
	webhooks, err := d.store.GetWebhooksForEvent(ctx, orgID, eventType)
	if err != nil {
		return fmt.Errorf("fetching webhooks for event %s: %w", eventType, err)
	}

	for _, wh := range webhooks {
		payload := &model.WebhookPayload{
			Event:     eventType,
			Timestamp: time.Now().UTC(),
			Data:      data,
			WebhookID: wh.ID,
		}
		if deliverErr := d.deliver(ctx, wh, payload); deliverErr != nil {
			d.logger.Error("webhook delivery failed",
				"webhook_id", wh.ID,
				"url", wh.URL,
				"event", eventType,
				"error", deliverErr)
		}
	}
	return nil
}

// deliver sends a single webhook payload and records the result.
func (d *Dispatcher) deliver(ctx context.Context, wh *model.Webhook, payload *model.WebhookPayload) error {
	return d.deliverAttempt(ctx, wh, payload, 1)
}

func (d *Dispatcher) deliverAttempt(ctx context.Context, wh *model.Webhook, payload *model.WebhookPayload, attempt int) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	signature := Sign(wh.Secret, body)
	deliveryID := uuid.New()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(headerSignature, signature)
	req.Header.Set(headerEvent, payload.Event)
	req.Header.Set(headerDelivery, deliveryID.String())

	delivery := &model.WebhookDelivery{
		WebhookID: wh.ID,
		EventType: payload.Event,
		Payload:   body,
		Attempt:   attempt,
	}

	resp, err := d.client.Do(req)
	if err != nil {
		delivery.Error = err.Error()
		delivery.Success = false
		scheduleRetry(delivery, attempt)
		if storeErr := d.store.CreateWebhookDelivery(ctx, delivery); storeErr != nil {
			d.logger.Error("failed to record delivery", "error", storeErr)
		}
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))

	delivery.ResponseStatus = resp.StatusCode
	delivery.ResponseBody = string(respBody)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		delivery.Success = true
	} else {
		delivery.Success = false
		delivery.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		scheduleRetry(delivery, attempt)
	}

	if storeErr := d.store.CreateWebhookDelivery(ctx, delivery); storeErr != nil {
		d.logger.Error("failed to record delivery", "error", storeErr)
	}

	if !delivery.Success {
		return fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// StartRetryWorker runs a background goroutine that polls for pending retries.
func (d *Dispatcher) StartRetryWorker(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				d.processRetries(ctx)
			}
		}
	}()
}

func (d *Dispatcher) processRetries(ctx context.Context) {
	pending, err := d.store.GetPendingRetries(ctx, 50)
	if err != nil {
		d.logger.Error("failed to fetch pending retries", "error", err)
		return
	}

	for _, delivery := range pending {
		wh, err := d.store.GetWebhook(ctx, delivery.WebhookID)
		if err != nil || wh == nil {
			d.logger.Error("failed to fetch webhook for retry", "webhook_id", delivery.WebhookID, "error", err)
			// Mark as done to avoid infinite retries on deleted webhooks.
			delivery.NextRetryAt = nil
			delivery.Error = "webhook not found"
			if updateErr := d.store.UpdateWebhookDelivery(ctx, delivery); updateErr != nil {
				d.logger.Error("failed to update delivery", "error", updateErr)
			}
			continue
		}

		var payload model.WebhookPayload
		if err := json.Unmarshal(delivery.Payload, &payload); err != nil {
			d.logger.Error("failed to unmarshal delivery payload", "delivery_id", delivery.ID, "error", err)
			delivery.NextRetryAt = nil
			delivery.Error = "invalid payload"
			if updateErr := d.store.UpdateWebhookDelivery(ctx, delivery); updateErr != nil {
				d.logger.Error("failed to update delivery", "error", updateErr)
			}
			continue
		}

		body := delivery.Payload
		signature := Sign(wh.Secret, body)
		deliveryID := uuid.New()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
		if err != nil {
			d.logger.Error("failed to create retry request", "error", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(headerSignature, signature)
		req.Header.Set(headerEvent, payload.Event)
		req.Header.Set(headerDelivery, deliveryID.String())

		resp, err := d.client.Do(req)
		if err != nil {
			delivery.Error = err.Error()
			delivery.Attempt++
			scheduleRetry(delivery, delivery.Attempt)
			if updateErr := d.store.UpdateWebhookDelivery(ctx, delivery); updateErr != nil {
				d.logger.Error("failed to update delivery", "error", updateErr)
			}
			continue
		}

		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
		_ = resp.Body.Close()

		delivery.ResponseStatus = resp.StatusCode
		delivery.ResponseBody = string(respBody)
		delivery.Attempt++

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			delivery.Success = true
			delivery.NextRetryAt = nil
			delivery.Error = ""
		} else {
			delivery.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
			scheduleRetry(delivery, delivery.Attempt)
		}

		if updateErr := d.store.UpdateWebhookDelivery(ctx, delivery); updateErr != nil {
			d.logger.Error("failed to update delivery", "error", updateErr)
		}
	}
}

// Sign computes the HMAC-SHA256 signature for a webhook payload.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature checks if a signature matches the expected HMAC-SHA256 of the body.
func VerifySignature(secret string, body []byte, signature string) bool {
	expected := Sign(secret, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func scheduleRetry(delivery *model.WebhookDelivery, attempt int) {
	if attempt >= maxAttempts {
		delivery.NextRetryAt = nil
		return
	}
	nextRetry := time.Now().UTC().Add(retryDelays[attempt-1])
	delivery.NextRetryAt = &nextRetry
}
