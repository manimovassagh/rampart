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
