package webhook

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// mockStore implements Store for testing.
type mockStore struct {
	webhooks   []*model.Webhook
	deliveries []*model.WebhookDelivery
	pending    []*model.WebhookDelivery
}

func (m *mockStore) GetWebhooksForEvent(_ context.Context, _ uuid.UUID, _ string) ([]*model.Webhook, error) {
	return m.webhooks, nil
}

func (m *mockStore) GetWebhook(_ context.Context, id uuid.UUID) (*model.Webhook, error) {
	for _, w := range m.webhooks {
		if w.ID == id {
			return w, nil
		}
	}
	return nil, nil
}

func (m *mockStore) CreateWebhookDelivery(_ context.Context, delivery *model.WebhookDelivery) error {
	delivery.ID = uuid.New()
	delivery.DeliveredAt = time.Now()
	m.deliveries = append(m.deliveries, delivery)
	return nil
}

func (m *mockStore) GetPendingRetries(_ context.Context, _ int) ([]*model.WebhookDelivery, error) {
	return m.pending, nil
}

func (m *mockStore) UpdateWebhookDelivery(_ context.Context, delivery *model.WebhookDelivery) error {
	for i, d := range m.deliveries {
		if d.ID == delivery.ID {
			m.deliveries[i] = delivery
			return nil
		}
	}
	m.deliveries = append(m.deliveries, delivery)
	return nil
}

func TestSignAndVerify(t *testing.T) {
	secret := "test-secret-key"
	body := []byte(`{"event":"user.login","data":{}}`)

	signature := Sign(secret, body)

	if signature[:7] != "sha256=" {
		t.Errorf("expected signature to start with sha256=, got %s", signature[:7])
	}

	if !VerifySignature(secret, body, signature) {
		t.Error("expected signature to verify successfully")
	}

	if VerifySignature("wrong-secret", body, signature) {
		t.Error("expected verification to fail with wrong secret")
	}

	if VerifySignature(secret, []byte("tampered body"), signature) {
		t.Error("expected verification to fail with tampered body")
	}
}

func TestSignDeterministic(t *testing.T) {
	secret := "deterministic-test"
	body := []byte(`{"test":true}`)

	sig1 := Sign(secret, body)
	sig2 := Sign(secret, body)

	if sig1 != sig2 {
		t.Errorf("expected deterministic signatures, got %s and %s", sig1, sig2)
	}
}

func TestDispatchSuccess(t *testing.T) {
	var receivedBody []byte
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	webhookID := uuid.New()
	store := &mockStore{
		webhooks: []*model.Webhook{
			{
				ID:      webhookID,
				OrgID:   uuid.New(),
				URL:     server.URL,
				Secret:  "test-secret",
				Events:  []string{"user.login"},
				Enabled: true,
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := NewDispatcher(store, logger)

	orgID := store.webhooks[0].OrgID
	err := d.Dispatch(context.Background(), orgID, "user.login", map[string]string{"user": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify payload was received.
	if len(receivedBody) == 0 {
		t.Fatal("expected body to be received")
	}

	var payload model.WebhookPayload
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if payload.Event != "user.login" {
		t.Errorf("expected event user.login, got %s", payload.Event)
	}
	if payload.WebhookID != webhookID {
		t.Errorf("expected webhook_id %s, got %s", webhookID, payload.WebhookID)
	}

	// Verify headers.
	if receivedHeaders.Get(headerSignature) == "" {
		t.Error("expected X-Rampart-Signature header")
	}
	if receivedHeaders.Get(headerEvent) != "user.login" {
		t.Errorf("expected X-Rampart-Event header to be user.login, got %s", receivedHeaders.Get(headerEvent))
	}
	if receivedHeaders.Get(headerDelivery) == "" {
		t.Error("expected X-Rampart-Delivery header")
	}

	// Verify signature is valid.
	sig := receivedHeaders.Get(headerSignature)
	if !VerifySignature("test-secret", receivedBody, sig) {
		t.Error("expected signature to verify against received body")
	}

	// Verify delivery was recorded.
	if len(store.deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(store.deliveries))
	}
	if !store.deliveries[0].Success {
		t.Error("expected delivery to be marked successful")
	}
}

func TestDispatchFailureSchedulesRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	store := &mockStore{
		webhooks: []*model.Webhook{
			{
				ID:      uuid.New(),
				OrgID:   uuid.New(),
				URL:     server.URL,
				Secret:  "secret",
				Events:  []string{"user.created"},
				Enabled: true,
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := NewDispatcher(store, logger)

	orgID := store.webhooks[0].OrgID
	err := d.Dispatch(context.Background(), orgID, "user.created", nil)
	if err != nil {
		t.Fatalf("Dispatch should not return error (errors are per-webhook): %v", err)
	}

	if len(store.deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(store.deliveries))
	}

	delivery := store.deliveries[0]
	if delivery.Success {
		t.Error("expected delivery to be marked as failed")
	}
	if delivery.NextRetryAt == nil {
		t.Error("expected retry to be scheduled")
	}
	if delivery.ResponseStatus != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", delivery.ResponseStatus)
	}
}

func TestDispatchNoWebhooks(t *testing.T) {
	store := &mockStore{webhooks: nil}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := NewDispatcher(store, logger)

	err := d.Dispatch(context.Background(), uuid.New(), "user.login", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.deliveries) != 0 {
		t.Errorf("expected no deliveries, got %d", len(store.deliveries))
	}
}

func TestScheduleRetryMaxAttempts(t *testing.T) {
	delivery := &model.WebhookDelivery{Attempt: maxAttempts}
	scheduleRetry(delivery, maxAttempts)

	if delivery.NextRetryAt != nil {
		t.Error("expected no retry after max attempts")
	}
}

func TestScheduleRetryExponentialBackoff(t *testing.T) {
	for attempt := 1; attempt < maxAttempts; attempt++ {
		delivery := &model.WebhookDelivery{}
		scheduleRetry(delivery, attempt)

		if delivery.NextRetryAt == nil {
			t.Errorf("expected retry scheduled for attempt %d", attempt)
			continue
		}

		expectedDelay := retryDelays[attempt-1]
		actualDelay := time.Until(*delivery.NextRetryAt)

		// Allow 2 second tolerance for timing.
		if actualDelay < expectedDelay-2*time.Second || actualDelay > expectedDelay+2*time.Second {
			t.Errorf("attempt %d: expected delay ~%v, got %v", attempt, expectedDelay, actualDelay)
		}
	}
}
