package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
)

func TestWebhookCRUD(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	// Create
	created, err := db.CreateWebhook(ctx, &model.Webhook{
		OrgID:       org.ID,
		URL:         "https://example.com/webhook-" + uniqueSlug(""),
		Secret:      "s3cret",
		Description: "Test webhook",
		EventTypes:  []string{"user.login", "user.created"},
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("expected non-nil webhook UUID")
	}
	if !created.Enabled {
		t.Error("expected webhook to be enabled")
	}
	if created.Description != "Test webhook" {
		t.Errorf("description: got %q, want %q", created.Description, "Test webhook")
	}

	// Get by ID
	got, err := db.GetWebhookByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetWebhookByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected webhook, got nil")
	}
	if got.URL != created.URL {
		t.Errorf("url: got %q, want %q", got.URL, created.URL)
	}
	if got.Secret != "s3cret" {
		t.Errorf("secret: got %q, want %q", got.Secret, "s3cret")
	}
	if len(got.EventTypes) != 2 {
		t.Errorf("event_types: expected 2, got %d", len(got.EventTypes))
	}

	// Get by ID not found
	missing, err := db.GetWebhookByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetWebhookByID (miss): %v", err)
	}
	if missing != nil {
		t.Error("expected nil for nonexistent webhook")
	}

	// Update
	updated, err := db.UpdateWebhook(ctx, created.ID, &model.UpdateWebhookRequest{
		URL:         created.URL,
		Description: "Updated description",
		EventTypes:  []string{"user.login"},
		Enabled:     false,
	})
	if err != nil {
		t.Fatalf("UpdateWebhook: %v", err)
	}
	if updated.Description != "Updated description" {
		t.Errorf("description: got %q, want %q", updated.Description, "Updated description")
	}
	if updated.Enabled {
		t.Error("expected webhook to be disabled after update")
	}
	if len(updated.EventTypes) != 1 {
		t.Errorf("event_types: expected 1, got %d", len(updated.EventTypes))
	}

	// List
	webhooks, total, err := db.ListWebhooks(ctx, org.ID, 100, 0)
	if err != nil {
		t.Fatalf("ListWebhooks: %v", err)
	}
	if total < 1 {
		t.Error("expected at least 1 webhook")
	}
	found := false
	for _, w := range webhooks {
		if w.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Error("created webhook not found in list")
	}

	// Delete
	err = db.DeleteWebhook(ctx, created.ID)
	if err != nil {
		t.Fatalf("DeleteWebhook: %v", err)
	}

	// Verify deleted
	gone, err := db.GetWebhookByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetWebhookByID after delete: %v", err)
	}
	if gone != nil {
		t.Error("expected nil after delete")
	}
}

func TestGetEnabledWebhooksForEvent(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	// Create enabled webhook matching user.login
	wh1, err := db.CreateWebhook(ctx, &model.Webhook{
		OrgID:       org.ID,
		URL:         "https://example.com/wh1-" + uniqueSlug(""),
		Secret:      "s1",
		Description: "Login hook",
		EventTypes:  []string{"user.login"},
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	t.Cleanup(func() { _ = db.DeleteWebhook(ctx, wh1.ID) })

	// Create disabled webhook matching user.login
	wh2, err := db.CreateWebhook(ctx, &model.Webhook{
		OrgID:       org.ID,
		URL:         "https://example.com/wh2-" + uniqueSlug(""),
		Secret:      "s2",
		Description: "Disabled hook",
		EventTypes:  []string{"user.login"},
		Enabled:     false,
	})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	t.Cleanup(func() { _ = db.DeleteWebhook(ctx, wh2.ID) })

	// Create wildcard webhook
	wh3, err := db.CreateWebhook(ctx, &model.Webhook{
		OrgID:       org.ID,
		URL:         "https://example.com/wh3-" + uniqueSlug(""),
		Secret:      "s3",
		Description: "Wildcard hook",
		EventTypes:  []string{"*"},
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	t.Cleanup(func() { _ = db.DeleteWebhook(ctx, wh3.ID) })

	// Query for user.login events
	hooks, err := db.GetEnabledWebhooksForEvent(ctx, org.ID, "user.login")
	if err != nil {
		t.Fatalf("GetEnabledWebhooksForEvent: %v", err)
	}

	foundWh1, foundWh2, foundWh3 := false, false, false
	for _, h := range hooks {
		switch h.ID {
		case wh1.ID:
			foundWh1 = true
		case wh2.ID:
			foundWh2 = true
		case wh3.ID:
			foundWh3 = true
		}
	}

	if !foundWh1 {
		t.Error("expected enabled webhook with matching event type")
	}
	if foundWh2 {
		t.Error("disabled webhook should not be returned")
	}
	if !foundWh3 {
		t.Error("expected wildcard webhook to match")
	}
}

func TestWebhookDeliveryCRUD(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	wh, err := db.CreateWebhook(ctx, &model.Webhook{
		OrgID:       org.ID,
		URL:         "https://example.com/delivery-" + uniqueSlug(""),
		Secret:      "s",
		Description: "Delivery test",
		EventTypes:  []string{"user.login"},
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	t.Cleanup(func() { _ = db.DeleteWebhook(ctx, wh.ID) })

	// Create an audit event to use as event_id
	actorID := uuid.New()
	event := &model.AuditEvent{
		OrgID:      org.ID,
		EventType:  "user.login",
		ActorID:    &actorID,
		ActorName:  "delivery-test",
		TargetType: "user",
		TargetID:   uuid.New().String(),
		TargetName: "target",
		IPAddress:  "127.0.0.1",
		UserAgent:  "test",
	}
	err = db.CreateAuditEvent(ctx, event)
	if err != nil {
		t.Fatalf("CreateAuditEvent: %v", err)
	}

	// Create delivery
	delivery := &model.WebhookDelivery{
		WebhookID: wh.ID,
		EventID:   event.ID,
		Status:    "pending",
		Attempts:  0,
	}
	err = db.CreateWebhookDelivery(ctx, delivery)
	if err != nil {
		t.Fatalf("CreateWebhookDelivery: %v", err)
	}

	// List deliveries
	deliveries, total, err := db.ListWebhookDeliveries(ctx, wh.ID, 100, 0)
	if err != nil {
		t.Fatalf("ListWebhookDeliveries: %v", err)
	}
	if total < 1 {
		t.Error("expected at least 1 delivery")
	}
	if len(deliveries) < 1 {
		t.Fatal("expected deliveries in page")
	}

	deliveryID := deliveries[0].ID

	// Update delivery
	responseCode := 200
	completedAt := time.Now()
	err = db.UpdateWebhookDelivery(ctx, deliveryID, "success", 1, &responseCode, "", nil, &completedAt)
	if err != nil {
		t.Fatalf("UpdateWebhookDelivery: %v", err)
	}

	// Delete old deliveries (should not error)
	_, err = db.DeleteOldDeliveries(ctx, 0)
	if err != nil {
		t.Fatalf("DeleteOldDeliveries: %v", err)
	}
}
