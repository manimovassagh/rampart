package audit

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// mockEventStore captures audit events for testing.
type mockEventStore struct {
	mu     sync.Mutex
	events []*model.AuditEvent
	err    error
}

func (m *mockEventStore) CreateAuditEvent(_ context.Context, event *model.AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, event)
	return nil
}

func (m *mockEventStore) getEvents() []*model.AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]*model.AuditEvent, len(m.events))
	copy(cp, m.events)
	return cp
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewLogger(t *testing.T) {
	store := &mockEventStore{}
	logger := NewLogger(store, testLogger())
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
}

func TestLogCreatesEventWithCorrectFields(t *testing.T) {
	store := &mockEventStore{}
	l := NewLogger(store, testLogger())

	orgID := uuid.New()
	actorID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/login", http.NoBody)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("User-Agent", "test-agent/1.0")

	details := map[string]any{"reason": "test"}

	l.Log(context.Background(), req, orgID, model.EventUserLogin, &actorID, "admin", "user", "user-123", "johndoe", details)

	// Wait for the goroutine to complete
	time.Sleep(50 * time.Millisecond)

	events := store.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.OrgID != orgID {
		t.Errorf("OrgID = %v, want %v", e.OrgID, orgID)
	}
	if e.EventType != model.EventUserLogin {
		t.Errorf("EventType = %q, want %q", e.EventType, model.EventUserLogin)
	}
	if e.ActorID == nil || *e.ActorID != actorID {
		t.Errorf("ActorID = %v, want %v", e.ActorID, actorID)
	}
	if e.ActorName != "admin" {
		t.Errorf("ActorName = %q, want admin", e.ActorName)
	}
	if e.TargetType != "user" {
		t.Errorf("TargetType = %q, want user", e.TargetType)
	}
	if e.TargetID != "user-123" {
		t.Errorf("TargetID = %q, want user-123", e.TargetID)
	}
	if e.TargetName != "johndoe" {
		t.Errorf("TargetName = %q, want johndoe", e.TargetName)
	}
	if e.IPAddress != "192.168.1.1:12345" {
		t.Errorf("IPAddress = %q, want 192.168.1.1:12345", e.IPAddress)
	}
	if e.UserAgent != "test-agent/1.0" {
		t.Errorf("UserAgent = %q, want test-agent/1.0", e.UserAgent)
	}
	if e.Details["reason"] != "test" {
		t.Errorf("Details[reason] = %v, want test", e.Details["reason"])
	}
}

func TestLogExtractsXForwardedFor(t *testing.T) {
	store := &mockEventStore{}
	l := NewLogger(store, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	l.Log(context.Background(), req, uuid.New(), model.EventUserLogin, nil, "", "", "", "", nil)

	time.Sleep(50 * time.Millisecond)

	events := store.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].IPAddress != "10.0.0.1" {
		t.Errorf("IPAddress = %q, want 10.0.0.1", events[0].IPAddress)
	}
}

func TestLogExtractsXRealIP(t *testing.T) {
	store := &mockEventStore{}
	l := NewLogger(store, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Real-IP", "10.0.0.2")

	l.Log(context.Background(), req, uuid.New(), model.EventUserLogin, nil, "", "", "", "", nil)

	time.Sleep(50 * time.Millisecond)

	events := store.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].IPAddress != "10.0.0.2" {
		t.Errorf("IPAddress = %q, want 10.0.0.2", events[0].IPAddress)
	}
}

func TestLogTruncatesLongUserAgent(t *testing.T) {
	store := &mockEventStore{}
	l := NewLogger(store, testLogger())

	longUA := ""
	for i := 0; i < 600; i++ {
		longUA += "x"
	}

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("User-Agent", longUA)

	l.Log(context.Background(), req, uuid.New(), model.EventUserLogin, nil, "", "", "", "", nil)

	time.Sleep(50 * time.Millisecond)

	events := store.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if len(events[0].UserAgent) != 500 {
		t.Errorf("UserAgent length = %d, want 500", len(events[0].UserAgent))
	}
}

func TestLogNilRequest(t *testing.T) {
	store := &mockEventStore{}
	l := NewLogger(store, testLogger())

	l.Log(context.Background(), nil, uuid.New(), model.EventUserCreated, nil, "system", "user", "u1", "test", nil)

	time.Sleep(50 * time.Millisecond)

	events := store.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].IPAddress != "" {
		t.Errorf("IPAddress = %q, want empty", events[0].IPAddress)
	}
	if events[0].UserAgent != "" {
		t.Errorf("UserAgent = %q, want empty", events[0].UserAgent)
	}
}

func TestLogNilReceiver(t *testing.T) {
	var l *Logger
	// Should not panic
	l.Log(context.Background(), nil, uuid.New(), model.EventUserLogin, nil, "", "", "", "", nil)
}

func TestLogSimple(t *testing.T) {
	store := &mockEventStore{}
	l := NewLogger(store, testLogger())

	orgID := uuid.New()
	l.LogSimple(context.Background(), nil, orgID, model.EventUserDeleted, nil, "admin", "user", "u1", "jane")

	time.Sleep(50 * time.Millisecond)

	events := store.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != model.EventUserDeleted {
		t.Errorf("EventType = %q, want %q", events[0].EventType, model.EventUserDeleted)
	}
	if events[0].Details != nil {
		t.Errorf("Details = %v, want nil", events[0].Details)
	}
}

func TestLogSimpleNilReceiver(t *testing.T) {
	var l *Logger
	// Should not panic
	l.LogSimple(context.Background(), nil, uuid.New(), model.EventUserLogin, nil, "", "", "", "")
}

func TestLogStoreError(t *testing.T) {
	store := &mockEventStore{err: http.ErrServerClosed}
	l := NewLogger(store, testLogger())

	// Should not panic even when store returns error
	l.Log(context.Background(), nil, uuid.New(), model.EventUserLogin, nil, "", "", "", "", nil)

	time.Sleep(50 * time.Millisecond)

	events := store.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events when store errors, got %d", len(events))
	}
}

func TestMarshalDetails(t *testing.T) {
	type detail struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	result := MarshalDetails(detail{Name: "test", Count: 42})
	if result == nil {
		t.Fatal("MarshalDetails returned nil")
	}
	if result["name"] != "test" {
		t.Errorf("name = %v, want test", result["name"])
	}
	// JSON numbers decode as float64
	if result["count"] != float64(42) {
		t.Errorf("count = %v, want 42", result["count"])
	}
}

func TestMarshalDetailsNil(t *testing.T) {
	result := MarshalDetails(nil)
	if result != nil {
		t.Errorf("MarshalDetails(nil) = %v, want nil", result)
	}
}

func TestMarshalDetailsUnmarshalable(t *testing.T) {
	// Channels cannot be marshaled to JSON
	result := MarshalDetails(make(chan int))
	if result != nil {
		t.Errorf("MarshalDetails(chan) = %v, want nil", result)
	}
}

func TestExtractIPPriority(t *testing.T) {
	// X-Forwarded-For takes precedence over X-Real-IP
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.Header.Set("X-Real-IP", "10.0.0.2")
	req.RemoteAddr = "192.168.1.1:1234"

	ip := extractIP(req)
	if ip != "10.0.0.1" {
		t.Errorf("extractIP = %q, want 10.0.0.1 (X-Forwarded-For takes precedence)", ip)
	}
}

func TestExtractIPFallsBackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = "192.168.1.1:1234"

	ip := extractIP(req)
	if ip != "192.168.1.1:1234" {
		t.Errorf("extractIP = %q, want 192.168.1.1:1234", ip)
	}
}
