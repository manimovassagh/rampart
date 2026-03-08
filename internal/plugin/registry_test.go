package plugin

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- Test EventHook ---

type testEventHook struct {
	name      string
	events    []string
	called    int
	mu        sync.Mutex
	returnErr error
}

func (h *testEventHook) Name() string { return h.name }
func (h *testEventHook) SupportedEvents() []string { return h.events }
func (h *testEventHook) HandleEvent(_ context.Context, _ *model.AuditEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.called++
	return h.returnErr
}

func TestDispatchEvent_AllEvents(t *testing.T) {
	reg := NewRegistry(discardLogger())
	hook := &testEventHook{name: "all-hook"}
	reg.RegisterEventHook(hook)

	reg.DispatchEvent(context.Background(), &model.AuditEvent{EventType: "user.login"})
	reg.DispatchEvent(context.Background(), &model.AuditEvent{EventType: "user.created"})

	if hook.called != 2 {
		t.Errorf("expected 2 calls, got %d", hook.called)
	}
}

func TestDispatchEvent_FilteredEvents(t *testing.T) {
	reg := NewRegistry(discardLogger())
	hook := &testEventHook{name: "login-only", events: []string{"user.login"}}
	reg.RegisterEventHook(hook)

	reg.DispatchEvent(context.Background(), &model.AuditEvent{EventType: "user.login"})
	reg.DispatchEvent(context.Background(), &model.AuditEvent{EventType: "user.created"})

	if hook.called != 1 {
		t.Errorf("expected 1 call, got %d", hook.called)
	}
}

func TestDispatchEvent_ErrorContinues(t *testing.T) {
	reg := NewRegistry(discardLogger())
	failing := &testEventHook{name: "failing", returnErr: errors.New("boom")}
	passing := &testEventHook{name: "passing"}
	reg.RegisterEventHook(failing)
	reg.RegisterEventHook(passing)

	reg.DispatchEvent(context.Background(), &model.AuditEvent{EventType: "user.login"})

	if passing.called != 1 {
		t.Error("second hook should still be called when first fails")
	}
}

// --- Test ClaimEnricher ---

type testEnricher struct {
	name      string
	key       string
	value     any
	returnErr error
}

func (e *testEnricher) Name() string { return e.name }
func (e *testEnricher) EnrichClaims(_ context.Context, _ uuid.UUID, claims map[string]any) error {
	if e.returnErr != nil {
		return e.returnErr
	}
	claims[e.key] = e.value
	return nil
}

func TestEnrichClaims(t *testing.T) {
	reg := NewRegistry(discardLogger())
	reg.RegisterClaimEnricher(&testEnricher{name: "dept", key: "department", value: "engineering"})
	reg.RegisterClaimEnricher(&testEnricher{name: "level", key: "level", value: 5})

	claims := map[string]any{}
	if err := reg.EnrichClaims(context.Background(), uuid.New(), claims); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if claims["department"] != "engineering" {
		t.Errorf("expected department=engineering, got %v", claims["department"])
	}
	if claims["level"] != 5 {
		t.Errorf("expected level=5, got %v", claims["level"])
	}
}

func TestEnrichClaims_ErrorContinues(t *testing.T) {
	reg := NewRegistry(discardLogger())
	reg.RegisterClaimEnricher(&testEnricher{name: "failing", returnErr: errors.New("boom")})
	reg.RegisterClaimEnricher(&testEnricher{name: "ok", key: "ok", value: true})

	claims := map[string]any{}
	_ = reg.EnrichClaims(context.Background(), uuid.New(), claims)

	if claims["ok"] != true {
		t.Error("second enricher should still run when first fails")
	}
}

// --- Test AuthMethod ---

type testAuthMethod struct {
	name string
}

func (m *testAuthMethod) Name() string { return m.name }
func (m *testAuthMethod) Authenticate(_ context.Context, _ *http.Request) (*AuthResult, error) {
	return &AuthResult{}, nil
}
func (m *testAuthMethod) Routes() []Route { return nil }

func TestAuthMethod_Registration(t *testing.T) {
	reg := NewRegistry(discardLogger())
	reg.RegisterAuthMethod(&testAuthMethod{name: "magic-link"})
	reg.RegisterAuthMethod(&testAuthMethod{name: "biometric"})

	if _, ok := reg.GetAuthMethod("magic-link"); !ok {
		t.Error("expected magic-link to be registered")
	}
	if _, ok := reg.GetAuthMethod("nonexistent"); ok {
		t.Error("expected nonexistent to not be found")
	}

	names := reg.AuthMethodNames()
	if len(names) != 2 {
		t.Errorf("expected 2 auth methods, got %d", len(names))
	}
}

// --- Test Middleware ---

type testMiddleware struct {
	name     string
	priority int
}

func (m *testMiddleware) Name() string { return m.name }
func (m *testMiddleware) Priority() int { return m.priority }
func (m *testMiddleware) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}

func TestMiddleware_Priority(t *testing.T) {
	reg := NewRegistry(discardLogger())
	reg.RegisterMiddleware(&testMiddleware{name: "low", priority: 10})
	reg.RegisterMiddleware(&testMiddleware{name: "high", priority: 1})
	reg.RegisterMiddleware(&testMiddleware{name: "mid", priority: 5})

	mws := reg.Middlewares()
	if mws[0].Name() != "high" || mws[1].Name() != "mid" || mws[2].Name() != "low" {
		t.Errorf("expected priority order high,mid,low, got %s,%s,%s", mws[0].Name(), mws[1].Name(), mws[2].Name())
	}
}

// --- Test ListPlugins ---

func TestListPlugins(t *testing.T) {
	reg := NewRegistry(discardLogger())
	reg.RegisterEventHook(&testEventHook{name: "hook1"})
	reg.RegisterClaimEnricher(&testEnricher{name: "enricher1"})
	reg.RegisterAuthMethod(&testAuthMethod{name: "method1"})
	reg.RegisterMiddleware(&testMiddleware{name: "mw1"})

	infos := reg.ListPlugins()
	if len(infos) != 4 {
		t.Errorf("expected 4 plugins, got %d", len(infos))
	}
}

// --- Test Close ---

type closableHook struct {
	testEventHook
	closed bool
}

func (h *closableHook) Close() error {
	h.closed = true
	return nil
}

func TestClose(t *testing.T) {
	reg := NewRegistry(discardLogger())
	hook := &closableHook{testEventHook: testEventHook{name: "closable"}}
	reg.RegisterEventHook(hook)

	_ = reg.Close()

	if !hook.closed {
		t.Error("expected Close to be called on closable plugin")
	}
}
