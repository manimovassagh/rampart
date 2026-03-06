package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/webhook"
)

// mockWebhookStore implements WebhookStore for testing.
type mockWebhookStore struct {
	webhooks   []*model.Webhook
	createErr  error
	getErr     error
	listErr    error
	updateErr  error
	deleteErr  error
	deliveries []*model.WebhookDelivery
	deliverErr error
}

func (m *mockWebhookStore) CreateWebhook(_ context.Context, wh *model.Webhook) (*model.Webhook, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	wh.ID = uuid.New()
	wh.CreatedAt = time.Now()
	wh.UpdatedAt = time.Now()
	m.webhooks = append(m.webhooks, wh)
	return wh, nil
}

func (m *mockWebhookStore) GetWebhook(_ context.Context, id uuid.UUID) (*model.Webhook, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, w := range m.webhooks {
		if w.ID == id {
			return w, nil
		}
	}
	return nil, nil
}

func (m *mockWebhookStore) ListWebhooks(_ context.Context, _ uuid.UUID) ([]*model.Webhook, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.webhooks, nil
}

func (m *mockWebhookStore) UpdateWebhook(_ context.Context, _ *model.Webhook) error {
	return m.updateErr
}

func (m *mockWebhookStore) DeleteWebhook(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockWebhookStore) GetWebhookDeliveries(_ context.Context, _ uuid.UUID, _ int) ([]*model.WebhookDelivery, error) {
	if m.deliverErr != nil {
		return nil, m.deliverErr
	}
	return m.deliveries, nil
}

// mockDispatcherStore implements webhook.Store for creating a real dispatcher.
type mockDispatcherStore struct {
	webhooks   []*model.Webhook
	deliveries []*model.WebhookDelivery
}

func (m *mockDispatcherStore) GetWebhooksForEvent(_ context.Context, _ uuid.UUID, _ string) ([]*model.Webhook, error) {
	return m.webhooks, nil
}

func (m *mockDispatcherStore) GetWebhook(_ context.Context, id uuid.UUID) (*model.Webhook, error) {
	for _, w := range m.webhooks {
		if w.ID == id {
			return w, nil
		}
	}
	return nil, nil
}

func (m *mockDispatcherStore) CreateWebhookDelivery(_ context.Context, d *model.WebhookDelivery) error {
	d.ID = uuid.New()
	d.DeliveredAt = time.Now()
	m.deliveries = append(m.deliveries, d)
	return nil
}

func (m *mockDispatcherStore) GetPendingRetries(_ context.Context, _ int) ([]*model.WebhookDelivery, error) {
	return nil, nil
}

func (m *mockDispatcherStore) UpdateWebhookDelivery(_ context.Context, _ *model.WebhookDelivery) error {
	return nil
}

func newTestWebhookHandler(store WebhookStore) *WebhookHandler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dispStore := &mockDispatcherStore{}
	dispatcher := webhook.NewDispatcher(dispStore, logger)
	return NewWebhookHandler(store, dispatcher, logger)
}

func webhookRequestWithAuth(method, path string, body any) (*http.Request, *httptest.ResponseRecorder) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	authUser := &middleware.AuthenticatedUser{
		UserID: uuid.New(),
		OrgID:  uuid.New(),
	}
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)

	return req, httptest.NewRecorder()
}

func TestWebhookCreateSuccess(t *testing.T) {
	store := &mockWebhookStore{}
	h := newTestWebhookHandler(store)

	reqBody := model.CreateWebhookRequest{
		URL:         "https://example.com/webhook",
		Events:      []string{"user.login", "user.created"},
		Enabled:     true,
		Description: "Test webhook",
	}

	req, rr := webhookRequestWithAuth(http.MethodPost, "/api/v1/admin/webhooks", reqBody)
	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var created model.Webhook
	if err := json.NewDecoder(rr.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if created.URL != "https://example.com/webhook" {
		t.Errorf("expected URL https://example.com/webhook, got %s", created.URL)
	}
	if created.Secret == "" {
		t.Error("expected secret to be generated")
	}
	if len(created.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(created.Events))
	}
}

func TestWebhookCreateMissingURL(t *testing.T) {
	store := &mockWebhookStore{}
	h := newTestWebhookHandler(store)

	reqBody := model.CreateWebhookRequest{
		Events: []string{"user.login"},
	}

	req, rr := webhookRequestWithAuth(http.MethodPost, "/api/v1/admin/webhooks", reqBody)
	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestWebhookCreateMissingEvents(t *testing.T) {
	store := &mockWebhookStore{}
	h := newTestWebhookHandler(store)

	reqBody := model.CreateWebhookRequest{
		URL: "https://example.com/webhook",
	}

	req, rr := webhookRequestWithAuth(http.MethodPost, "/api/v1/admin/webhooks", reqBody)
	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestWebhookList(t *testing.T) {
	store := &mockWebhookStore{
		webhooks: []*model.Webhook{
			{
				ID:      uuid.New(),
				URL:     "https://example.com/hook1",
				Secret:  "abcd1234efgh5678",
				Events:  []string{"user.login"},
				Enabled: true,
			},
		},
	}
	h := newTestWebhookHandler(store)

	req, rr := webhookRequestWithAuth(http.MethodGet, "/api/v1/admin/webhooks", nil)
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var webhooks []*model.Webhook
	if err := json.NewDecoder(rr.Body).Decode(&webhooks); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(webhooks) != 1 {
		t.Fatalf("expected 1 webhook, got %d", len(webhooks))
	}

	// Secret should be masked.
	if webhooks[0].Secret == "abcd1234efgh5678" {
		t.Error("expected secret to be masked in list response")
	}
}

func TestWebhookGetNotFound(t *testing.T) {
	store := &mockWebhookStore{}
	h := newTestWebhookHandler(store)

	req, rr := webhookRequestWithAuth(http.MethodGet, "/api/v1/admin/webhooks/"+uuid.New().String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uuid.New().String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.Get(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestWebhookDelete(t *testing.T) {
	store := &mockWebhookStore{}
	h := newTestWebhookHandler(store)

	req, rr := webhookRequestWithAuth(http.MethodDelete, "/api/v1/admin/webhooks/"+uuid.New().String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uuid.New().String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.Delete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rr.Code)
	}
}

func TestWebhookCreateNoAuth(t *testing.T) {
	store := &mockWebhookStore{}
	h := newTestWebhookHandler(store)

	reqBody := model.CreateWebhookRequest{
		URL:    "https://example.com/webhook",
		Events: []string{"user.login"},
	}

	b, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/webhooks", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "ShortSecret", input: "abcd", expected: "****"},
		{name: "ExactlyEight", input: "abcdefgh", expected: "****"},
		{name: "LongSecret", input: "abcd1234efgh5678", expected: "abcd****5678"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := maskSecret(tc.input)
			if result != tc.expected {
				t.Errorf("maskSecret(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
