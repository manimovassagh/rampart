package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
)

const testActorName = "admin"

func TestGetAuditEventByID(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	actorID := uuid.New()
	event := &model.AuditEvent{
		OrgID:      org.ID,
		EventType:  "user.created",
		ActorID:    &actorID,
		ActorName:  testActorName,
		TargetType: "user",
		TargetID:   uuid.New().String(),
		TargetName: "newuser",
		IPAddress:  "192.168.1.1",
		UserAgent:  "test-agent",
		Details:    map[string]any{"method": "api"},
	}

	err := db.CreateAuditEvent(ctx, event)
	if err != nil {
		t.Fatalf("CreateAuditEvent: %v", err)
	}
	if event.ID == uuid.Nil {
		t.Fatal("expected event ID to be populated")
	}

	// Get by ID
	got, err := db.GetAuditEventByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("GetAuditEventByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected event, got nil")
	}
	if got.EventType != "user.created" {
		t.Errorf("event_type: got %q, want %q", got.EventType, "user.created")
	}
	if got.ActorName != testActorName {
		t.Errorf("actor_name: got %q, want %q", got.ActorName, "admin")
	}
	if got.TargetName != "newuser" {
		t.Errorf("target_name: got %q, want %q", got.TargetName, "newuser")
	}
	if got.IPAddress != "192.168.1.1" {
		t.Errorf("ip_address: got %q, want %q", got.IPAddress, "192.168.1.1")
	}
	// Verify details are preserved
	if got.Details == nil {
		t.Fatal("expected details to be populated")
	}
	if got.Details["method"] != "api" {
		t.Errorf("details[method]: got %v, want %q", got.Details["method"], "api")
	}
}

func TestLoginCountsByDay(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	// Create a login event
	actorID := uuid.New()
	err := db.CreateAuditEvent(ctx, &model.AuditEvent{
		OrgID:      org.ID,
		EventType:  "user.login",
		ActorID:    &actorID,
		ActorName:  "logintest",
		TargetType: "user",
		TargetID:   uuid.New().String(),
		TargetName: "logintest",
		IPAddress:  "127.0.0.1",
		UserAgent:  "test",
	})
	if err != nil {
		t.Fatalf("CreateAuditEvent: %v", err)
	}

	counts, err := db.LoginCountsByDay(ctx, org.ID, 7)
	if err != nil {
		t.Fatalf("LoginCountsByDay: %v", err)
	}
	if len(counts) < 1 {
		t.Error("expected at least 1 day count")
	}

	// Verify today's count includes our event
	totalLogins := 0
	for _, dc := range counts {
		totalLogins += dc.Count
	}
	if totalLogins < 1 {
		t.Error("expected at least 1 login event in counts")
	}
}

func TestAuditEventNilActor(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	// Create event with nil actor
	event := &model.AuditEvent{
		OrgID:      org.ID,
		EventType:  "system.startup",
		ActorID:    nil,
		ActorName:  "",
		TargetType: "system",
		TargetID:   "",
		TargetName: "",
		IPAddress:  "",
		UserAgent:  "",
	}

	err := db.CreateAuditEvent(ctx, event)
	if err != nil {
		t.Fatalf("CreateAuditEvent: %v", err)
	}
	if event.ID == uuid.Nil {
		t.Fatal("expected event ID to be populated")
	}

	// Retrieve and verify
	got, err := db.GetAuditEventByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("GetAuditEventByID: %v", err)
	}
	if got.EventType != "system.startup" {
		t.Errorf("event_type: got %q, want %q", got.EventType, "system.startup")
	}
}

func TestListAuditEventsFilterAndSearch(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)

	actorID := uuid.New()
	uniqueName := "filterable-" + uniqueSlug("")

	// Create events with different types
	for _, eventType := range []string{"user.login", "user.created", "user.deleted"} {
		err := db.CreateAuditEvent(ctx, &model.AuditEvent{
			OrgID:      org.ID,
			EventType:  eventType,
			ActorID:    &actorID,
			ActorName:  uniqueName,
			TargetType: "user",
			TargetID:   uuid.New().String(),
			TargetName: "target",
			IPAddress:  "10.0.0.1",
			UserAgent:  "test",
		})
		if err != nil {
			t.Fatalf("CreateAuditEvent(%s): %v", eventType, err)
		}
	}

	// Filter by event type
	events, total, err := db.ListAuditEvents(ctx, org.ID, "user.created", "", 100, 0)
	if err != nil {
		t.Fatalf("ListAuditEvents filter: %v", err)
	}
	if total < 1 {
		t.Error("expected at least 1 user.created event")
	}
	for _, e := range events {
		if e.EventType != "user.created" {
			t.Errorf("expected event_type=user.created, got %q", e.EventType)
		}
	}

	// Search by actor name
	_, total, err = db.ListAuditEvents(ctx, org.ID, "", uniqueName, 100, 0)
	if err != nil {
		t.Fatalf("ListAuditEvents search: %v", err)
	}
	if total < 3 {
		t.Errorf("expected at least 3 events matching actor name, got %d", total)
	}

	// Combined filter + search
	_, total, err = db.ListAuditEvents(ctx, org.ID, "user.deleted", uniqueName, 100, 0)
	if err != nil {
		t.Fatalf("ListAuditEvents combined: %v", err)
	}
	if total < 1 {
		t.Error("expected at least 1 matching event with combined filter")
	}

	// Pagination
	events, _, err = db.ListAuditEvents(ctx, org.ID, "", uniqueName, 1, 0)
	if err != nil {
		t.Fatalf("ListAuditEvents pagination: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event with limit=1, got %d", len(events))
	}
}
