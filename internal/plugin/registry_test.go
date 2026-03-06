package plugin

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
)

// mockPlugin is a minimal test plugin.
type mockPlugin struct {
	name    string
	hooks   []HookPoint
	initErr error
	shutErr error
}

func (m *mockPlugin) Name() string                                    { return m.name }
func (m *mockPlugin) Version() string                                 { return "0.1.0" }
func (m *mockPlugin) Description() string                             { return "test plugin" }
func (m *mockPlugin) Init(_ context.Context, _ map[string]any) error  { return m.initErr }
func (m *mockPlugin) Shutdown(_ context.Context) error                { return m.shutErr }
func (m *mockPlugin) Hooks() []HookPoint                              { return m.hooks }

// mockAuthPlugin implements Plugin + AuthHook.
type mockAuthPlugin struct {
	mockPlugin
	preAuthResp *AuthResponse
	preAuthErr  error
	postAuthErr error
	postAuthCb  func(*AuthRequest, *AuthResult)
}

func (m *mockAuthPlugin) OnPreAuth(_ context.Context, req *AuthRequest) (*AuthResponse, error) {
	return m.preAuthResp, m.preAuthErr
}

func (m *mockAuthPlugin) OnPostAuth(_ context.Context, req *AuthRequest, result *AuthResult) error {
	if m.postAuthCb != nil {
		m.postAuthCb(req, result)
	}
	return m.postAuthErr
}

// mockTokenPlugin implements Plugin + TokenHook.
type mockTokenPlugin struct {
	mockPlugin
	claims map[string]any
}

func (m *mockTokenPlugin) OnCustomClaims(_ context.Context, _ string, existing map[string]any) (map[string]any, error) {
	return m.claims, nil
}

func (m *mockTokenPlugin) OnPreTokenIssue(_ context.Context, claims map[string]any) (map[string]any, error) {
	return claims, nil
}

// mockRegPlugin implements Plugin + RegistrationHook.
type mockRegPlugin struct {
	mockPlugin
	preRegResp *RegistrationResponse
}

func (m *mockRegPlugin) OnPreRegister(_ context.Context, _ *RegistrationRequest) (*RegistrationResponse, error) {
	return m.preRegResp, nil
}

func (m *mockRegPlugin) OnPostRegister(_ context.Context, _ string) error {
	return nil
}

// mockEventPlugin implements Plugin + EventHook.
type mockEventPlugin struct {
	mockPlugin
	received []*Event
}

func (m *mockEventPlugin) OnEvent(_ context.Context, event *Event) error {
	m.received = append(m.received, event)
	return nil
}

// panicAuthPlugin panics on pre-auth to test recovery.
type panicAuthPlugin struct {
	mockPlugin
}

func (m *panicAuthPlugin) OnPreAuth(_ context.Context, _ *AuthRequest) (*AuthResponse, error) {
	panic("boom")
}

func (m *panicAuthPlugin) OnPostAuth(_ context.Context, _ *AuthRequest, _ *AuthResult) error {
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestRegistryRegisterAndPlugins(t *testing.T) {
	reg := NewRegistry(testLogger())

	p := &mockPlugin{name: "test-plugin", hooks: []HookPoint{HookPreAuth}}
	if err := reg.Register(p, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	infos := reg.Plugins()
	if len(infos) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(infos))
	}
	if infos[0].Name != "test-plugin" {
		t.Errorf("expected name test-plugin, got %s", infos[0].Name)
	}
}

func TestRegistryDuplicateNameRejected(t *testing.T) {
	reg := NewRegistry(testLogger())

	p1 := &mockPlugin{name: "dup"}
	p2 := &mockPlugin{name: "dup"}
	if err := reg.Register(p1, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := reg.Register(p2, nil); err == nil {
		t.Fatal("expected error for duplicate plugin name")
	}
}

func TestRegistryInitErrorRejected(t *testing.T) {
	reg := NewRegistry(testLogger())

	p := &mockPlugin{name: "bad", initErr: errors.New("init failed")}
	if err := reg.Register(p, nil); err == nil {
		t.Fatal("expected error when init fails")
	}
}

func TestRegistryRunPreAuthAllow(t *testing.T) {
	reg := NewRegistry(testLogger())

	p := &mockAuthPlugin{
		mockPlugin:  mockPlugin{name: "auth-allow", hooks: []HookPoint{HookPreAuth}},
		preAuthResp: &AuthResponse{Allow: true},
	}
	if err := reg.Register(p, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err := reg.RunPreAuth(context.Background(), &AuthRequest{IP: "1.2.3.4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allow {
		t.Error("expected allow=true")
	}
}

func TestRegistryRunPreAuthDeny(t *testing.T) {
	reg := NewRegistry(testLogger())

	p := &mockAuthPlugin{
		mockPlugin:  mockPlugin{name: "auth-deny", hooks: []HookPoint{HookPreAuth}},
		preAuthResp: &AuthResponse{Allow: false, Reason: "blocked"},
	}
	if err := reg.Register(p, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err := reg.RunPreAuth(context.Background(), &AuthRequest{IP: "1.2.3.4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allow {
		t.Error("expected allow=false")
	}
	if resp.Reason != "blocked" {
		t.Errorf("expected reason blocked, got %s", resp.Reason)
	}
}

func TestRegistryRunPostAuth(t *testing.T) {
	called := false
	reg := NewRegistry(testLogger())

	p := &mockAuthPlugin{
		mockPlugin: mockPlugin{name: "post-auth", hooks: []HookPoint{HookPostAuth}},
		postAuthCb: func(_ *AuthRequest, _ *AuthResult) { called = true },
	}
	if err := reg.Register(p, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := reg.RunPostAuth(context.Background(), &AuthRequest{}, &AuthResult{Success: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("post-auth callback was not called")
	}
}

func TestRegistryRunCustomClaims(t *testing.T) {
	reg := NewRegistry(testLogger())

	p := &mockTokenPlugin{
		mockPlugin: mockPlugin{name: "token", hooks: []HookPoint{HookCustomClaims}},
		claims:     map[string]any{"role": "admin"},
	}
	if err := reg.Register(p, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := reg.RunCustomClaims(context.Background(), "user-1", map[string]any{"sub": "user-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["role"] != "admin" {
		t.Errorf("expected role=admin, got %v", result["role"])
	}
	if result["sub"] != "user-1" {
		t.Errorf("expected sub=user-1, got %v", result["sub"])
	}
}

func TestRegistryRunPreRegisterDeny(t *testing.T) {
	reg := NewRegistry(testLogger())

	p := &mockRegPlugin{
		mockPlugin: mockPlugin{name: "reg-deny", hooks: []HookPoint{HookPreRegister}},
		preRegResp: &RegistrationResponse{Allow: false, Reason: "blocked domain"},
	}
	if err := reg.Register(p, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err := reg.RunPreRegister(context.Background(), &RegistrationRequest{Email: "a@evil.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allow {
		t.Error("expected allow=false")
	}
}

func TestRegistryRunEvent(t *testing.T) {
	reg := NewRegistry(testLogger())

	p := &mockEventPlugin{
		mockPlugin: mockPlugin{name: "event-log", hooks: []HookPoint{HookEvent}},
	}
	if err := reg.Register(p, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	event := &Event{Type: "user.login", ActorID: "u1", Timestamp: time.Now()}
	err := reg.RunEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(p.received))
	}
	if p.received[0].Type != "user.login" {
		t.Errorf("expected event type user.login, got %s", p.received[0].Type)
	}
}

func TestRegistryShutdown(t *testing.T) {
	reg := NewRegistry(testLogger())

	p := &mockPlugin{name: "shutdown-test"}
	if err := reg.Register(p, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := reg.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistryShutdownError(t *testing.T) {
	reg := NewRegistry(testLogger())

	p := &mockPlugin{name: "shutdown-err", shutErr: errors.New("shutdown failed")}
	if err := reg.Register(p, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := reg.Shutdown(context.Background()); err == nil {
		t.Fatal("expected shutdown error")
	}
}

func TestRegistryPreAuthPanicRecovery(t *testing.T) {
	reg := NewRegistry(testLogger())

	p := &panicAuthPlugin{
		mockPlugin: mockPlugin{name: "panic-auth", hooks: []HookPoint{HookPreAuth}},
	}
	if err := reg.Register(p, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not panic, should log error and continue
	resp, err := reg.RunPreAuth(context.Background(), &AuthRequest{IP: "1.2.3.4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allow {
		t.Error("expected allow=true after panic recovery (continue to default)")
	}
}

func TestRegistryNoHooksReturnsDefaults(t *testing.T) {
	reg := NewRegistry(testLogger())

	resp, err := reg.RunPreAuth(context.Background(), &AuthRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allow {
		t.Error("expected allow=true with no hooks")
	}

	regResp, err := reg.RunPreRegister(context.Background(), &RegistrationRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !regResp.Allow {
		t.Error("expected allow=true with no hooks")
	}

	claims, err := reg.RunCustomClaims(context.Background(), "u1", map[string]any{"sub": "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims["sub"] != "u1" {
		t.Errorf("expected claims to pass through")
	}

	routes := reg.Routes()
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
}
