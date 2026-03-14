package database

import (
	"context"
	"testing"
	"time"
)

func TestConnectInvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := Connect(ctx, "postgres://invalid:invalid@localhost:9999/nonexistent?sslmode=disable")
	if err == nil {
		t.Fatal("expected error connecting to invalid database")
	}
}

func TestConnectMalformedURL(t *testing.T) {
	ctx := context.Background()
	_, err := Connect(ctx, "not-a-valid-url")
	if err == nil {
		t.Fatal("expected error for malformed URL")
	}
}

func TestPingNilPool(t *testing.T) {
	db := &DB{}
	err := db.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error pinging nil pool")
	}
}

func TestCloseNilPool(t *testing.T) {
	db := &DB{}
	// Should not panic
	db.Close()
}

func TestQueryCtxAppliesTimeout(t *testing.T) {
	parent := context.Background()
	ctx, cancel := queryCtx(parent)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline")
	}
	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > defaultQueryTimeout {
		t.Fatalf("expected deadline within %v, got %v remaining", defaultQueryTimeout, remaining)
	}
}

func TestQueryCtxRespectsExistingTighterDeadline(t *testing.T) {
	tighterTimeout := 1 * time.Second
	parent, parentCancel := context.WithTimeout(context.Background(), tighterTimeout)
	defer parentCancel()

	ctx, cancel := queryCtx(parent)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline")
	}
	remaining := time.Until(deadline)
	// The effective deadline should be the tighter parent deadline, not defaultQueryTimeout.
	if remaining > tighterTimeout {
		t.Fatalf("expected deadline within %v (parent), got %v remaining", tighterTimeout, remaining)
	}
}
