package webhook

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

type mockStore struct {
	webhooks   []*model.Webhook
	deliveries []*model.WebhookDelivery
	events     map[uuid.UUID]*model.AuditEvent
	created    []*model.WebhookDelivery
	updates    []deliveryUpdate
}

type deliveryUpdate struct {
	ID     uuid.UUID
	Status string
}

func (m *mockStore) GetEnabledWebhooksForEvent(_ context.Context, _ uuid.UUID, _ string) ([]*model.Webhook, error) {
	return m.webhooks, nil
}

func (m *mockStore) CreateWebhookDelivery(_ context.Context, d *model.WebhookDelivery) error {
	d.ID = uuid.New()
	m.created = append(m.created, d)
	return nil
}

func (m *mockStore) UpdateWebhookDelivery(_ context.Context, id uuid.UUID, status string, _ int, _ *int, _ string, _ *time.Time, _ *time.Time) error {
	m.updates = append(m.updates, deliveryUpdate{ID: id, Status: status})
	return nil
}

func (m *mockStore) GetPendingDeliveries(_ context.Context, _ int) ([]*model.WebhookDelivery, error) {
	return m.deliveries, nil
}

func (m *mockStore) GetWebhookByID(_ context.Context, id uuid.UUID) (*model.Webhook, error) {
	for _, w := range m.webhooks {
		if w.ID == id {
			return w, nil
		}
	}
	return nil, nil
}

func (m *mockStore) GetAuditEventByID(_ context.Context, id uuid.UUID) (*model.AuditEvent, error) {
	if e, ok := m.events[id]; ok {
		return e, nil
	}
	return nil, nil
}

// newTestDispatcher creates a dispatcher with the default transport (no SSRF
// validation) so tests can use httptest.NewServer on localhost.
func newTestDispatcher(s *mockStore) *Dispatcher {
	d := NewDispatcher(s, slog.New(slog.NewTextHandler(io.Discard, nil)))
	d.client.Transport = http.DefaultTransport
	return d
}

func TestDispatch_EnqueuesDeliveries(t *testing.T) {
	orgID := uuid.New()
	whID := uuid.New()
	store := &mockStore{
		webhooks: []*model.Webhook{
			{ID: whID, OrgID: orgID, URL: "https://example.com/hook", Secret: "secret", Enabled: true, EventTypes: []string{"*"}},
		},
	}

	d := NewDispatcher(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	event := &model.AuditEvent{
		ID:        uuid.New(),
		OrgID:     orgID,
		EventType: "user.login",
		CreatedAt: time.Now(),
	}

	d.Dispatch(context.Background(), event)

	if len(store.created) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(store.created))
	}
	if store.created[0].WebhookID != whID {
		t.Errorf("expected webhook ID %s, got %s", whID, store.created[0].WebhookID)
	}
	if store.created[0].EventID != event.ID {
		t.Errorf("expected event ID %s, got %s", event.ID, store.created[0].EventID)
	}
}

func TestDeliver_SuccessfulDelivery(t *testing.T) {
	// Set up a test server that returns 200
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Rampart-Signature-256") == "" {
			t.Error("missing signature header")
		}
		if r.Header.Get("X-Rampart-Event") != "user.login" {
			t.Errorf("unexpected event header: %s", r.Header.Get("X-Rampart-Event"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	whID := uuid.New()
	eventID := uuid.New()
	orgID := uuid.New()

	store := &mockStore{
		webhooks: []*model.Webhook{
			{ID: whID, OrgID: orgID, URL: srv.URL, Secret: "test-secret", Enabled: true},
		},
		events: map[uuid.UUID]*model.AuditEvent{
			eventID: {ID: eventID, OrgID: orgID, EventType: "user.login", CreatedAt: time.Now()},
		},
		deliveries: []*model.WebhookDelivery{
			{ID: uuid.New(), WebhookID: whID, EventID: eventID, Status: "pending", Attempts: 0},
		},
	}

	d := newTestDispatcher(store)
	d.ProcessPending(context.Background())

	if len(store.updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(store.updates))
	}
	if store.updates[0].Status != "success" {
		t.Errorf("expected status 'success', got %q", store.updates[0].Status)
	}
}

func TestDeliver_FailedDelivery_Retries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	whID := uuid.New()
	eventID := uuid.New()
	orgID := uuid.New()

	store := &mockStore{
		webhooks: []*model.Webhook{
			{ID: whID, OrgID: orgID, URL: srv.URL, Secret: "secret", Enabled: true},
		},
		events: map[uuid.UUID]*model.AuditEvent{
			eventID: {ID: eventID, OrgID: orgID, EventType: "user.login", CreatedAt: time.Now()},
		},
		deliveries: []*model.WebhookDelivery{
			{ID: uuid.New(), WebhookID: whID, EventID: eventID, Status: "pending", Attempts: 0},
		},
	}

	d := newTestDispatcher(store)
	d.ProcessPending(context.Background())

	if len(store.updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(store.updates))
	}
	// First failure should still be pending (for retry)
	if store.updates[0].Status != "pending" {
		t.Errorf("expected status 'pending' for retry, got %q", store.updates[0].Status)
	}
}

func TestDeliver_MaxAttempts_PermanentFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	whID := uuid.New()
	eventID := uuid.New()
	orgID := uuid.New()

	store := &mockStore{
		webhooks: []*model.Webhook{
			{ID: whID, OrgID: orgID, URL: srv.URL, Secret: "secret", Enabled: true},
		},
		events: map[uuid.UUID]*model.AuditEvent{
			eventID: {ID: eventID, OrgID: orgID, EventType: "user.login", CreatedAt: time.Now()},
		},
		deliveries: []*model.WebhookDelivery{
			{ID: uuid.New(), WebhookID: whID, EventID: eventID, Status: "pending", Attempts: 4}, // 4 attempts already
		},
	}

	d := newTestDispatcher(store)
	d.ProcessPending(context.Background())

	if len(store.updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(store.updates))
	}
	if store.updates[0].Status != "failed" {
		t.Errorf("expected status 'failed' after max attempts, got %q", store.updates[0].Status)
	}
}

func TestSSRFSafeTransport_BlocksLoopback(t *testing.T) {
	// Verify the SSRF-safe transport blocks connections to loopback addresses.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: newSSRFSafeTransport(),
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Do(req)
	if err == nil {
		t.Fatal("expected SSRF-safe transport to block loopback connection")
	}
}

func TestSign(t *testing.T) {
	sig := sign([]byte(`{"test":true}`), "my-secret")
	if len(sig) != 64 {
		t.Errorf("expected 64 char hex signature, got %d chars", len(sig))
	}
}

func TestBuildPayload(t *testing.T) {
	actorID := uuid.New()
	event := &model.AuditEvent{
		ID:         uuid.New(),
		OrgID:      uuid.New(),
		EventType:  "user.login",
		ActorID:    &actorID,
		ActorName:  "admin",
		TargetType: "user",
		TargetID:   uuid.New().String(),
		TargetName: "testuser",
		CreatedAt:  time.Now(),
		Details:    map[string]any{"ip": "1.2.3.4"},
	}

	payload := buildPayload(event)

	if payload.Type != "user.login" {
		t.Errorf("expected type 'user.login', got %q", payload.Type)
	}
	if payload.Actor == nil {
		t.Fatal("expected actor to be set")
	}
	if payload.Actor.Name != "admin" {
		t.Errorf("expected actor name 'admin', got %q", payload.Actor.Name)
	}
	if payload.Target == nil {
		t.Fatal("expected target to be set")
	}
	if payload.Target.Type != "user" {
		t.Errorf("expected target type 'user', got %q", payload.Target.Type)
	}
}
