package builtin

import (
	"context"
	"testing"

	"github.com/manimovassagh/rampart/internal/plugin"
)

func TestRateLimiterMetadata(t *testing.T) {
	rl := NewRateLimiter()
	if rl.Name() != rateLimitName {
		t.Errorf("expected name %s, got %s", rateLimitName, rl.Name())
	}
	if rl.Version() != rateLimitVersion {
		t.Errorf("expected version %s, got %s", rateLimitVersion, rl.Version())
	}
	hooks := rl.Hooks()
	if len(hooks) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(hooks))
	}
}

func TestRateLimiterInitWithConfig(t *testing.T) {
	rl := NewRateLimiter()
	err := rl.Init(context.Background(), map[string]any{
		"max_attempts":   5,
		"window_seconds": 60,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rl.maxAttempts != 5 {
		t.Errorf("expected maxAttempts=5, got %d", rl.maxAttempts)
	}
}

func TestRateLimiterInitNilConfig(t *testing.T) {
	rl := NewRateLimiter()
	if err := rl.Init(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rl.maxAttempts != defaultMaxAttempts {
		t.Errorf("expected default maxAttempts=%d, got %d", defaultMaxAttempts, rl.maxAttempts)
	}
}

func TestRateLimiterAllowsNormalTraffic(t *testing.T) {
	rl := NewRateLimiter()
	_ = rl.Init(context.Background(), nil)

	ctx := context.Background()
	req := &plugin.AuthRequest{IP: "10.0.0.1"}

	resp, err := rl.OnPreAuth(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allow {
		t.Error("expected allow=true for first request")
	}
}

func TestRateLimiterBlocksAfterMaxAttempts(t *testing.T) {
	rl := NewRateLimiter()
	_ = rl.Init(context.Background(), map[string]any{"max_attempts": 3})

	ctx := context.Background()
	ip := "10.0.0.2"
	req := &plugin.AuthRequest{IP: ip}
	failResult := &plugin.AuthResult{Success: false}

	// Simulate 3 failed attempts
	for i := 0; i < 3; i++ {
		_ = rl.OnPostAuth(ctx, req, failResult)
	}

	// 4th attempt should be blocked
	resp, err := rl.OnPreAuth(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allow {
		t.Error("expected allow=false after exceeding max attempts")
	}
	if resp.Reason != rateLimitReason {
		t.Errorf("expected reason %q, got %q", rateLimitReason, resp.Reason)
	}
}

func TestRateLimiterResetsOnSuccess(t *testing.T) {
	rl := NewRateLimiter()
	_ = rl.Init(context.Background(), map[string]any{"max_attempts": 3})

	ctx := context.Background()
	ip := "10.0.0.3"
	req := &plugin.AuthRequest{IP: ip}

	// 2 failures
	for i := 0; i < 2; i++ {
		_ = rl.OnPostAuth(ctx, req, &plugin.AuthResult{Success: false})
	}

	// Successful login resets counter
	_ = rl.OnPostAuth(ctx, req, &plugin.AuthResult{Success: true})

	// Should be allowed again
	resp, err := rl.OnPreAuth(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allow {
		t.Error("expected allow=true after successful login reset")
	}
}

func TestRateLimiterAllowsEmptyIP(t *testing.T) {
	rl := NewRateLimiter()
	_ = rl.Init(context.Background(), nil)

	resp, err := rl.OnPreAuth(context.Background(), &plugin.AuthRequest{IP: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allow {
		t.Error("expected allow=true for empty IP")
	}
}

func TestRateLimiterAllowsNilRequest(t *testing.T) {
	rl := NewRateLimiter()
	_ = rl.Init(context.Background(), nil)

	resp, err := rl.OnPreAuth(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Allow {
		t.Error("expected allow=true for nil request")
	}
}

func TestRateLimiterDifferentIPsIndependent(t *testing.T) {
	rl := NewRateLimiter()
	_ = rl.Init(context.Background(), map[string]any{"max_attempts": 2})

	ctx := context.Background()
	req1 := &plugin.AuthRequest{IP: "10.0.0.10"}
	req2 := &plugin.AuthRequest{IP: "10.0.0.11"}

	// 2 failures for IP1
	for i := 0; i < 2; i++ {
		_ = rl.OnPostAuth(ctx, req1, &plugin.AuthResult{Success: false})
	}

	// IP1 should be blocked
	resp1, _ := rl.OnPreAuth(ctx, req1)
	if resp1.Allow {
		t.Error("expected IP1 to be blocked")
	}

	// IP2 should still be allowed
	resp2, _ := rl.OnPreAuth(ctx, req2)
	if !resp2.Allow {
		t.Error("expected IP2 to be allowed")
	}
}

func TestRateLimiterShutdown(t *testing.T) {
	rl := NewRateLimiter()
	if err := rl.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
