package cluster

import (
	"testing"
)

func TestAdvisoryLockID(t *testing.T) {
	// Ensure the lock ID is stable and non-zero.
	if advisoryLockID == 0 {
		t.Error("advisory lock ID must be non-zero")
	}
}

func TestNewLeader(t *testing.T) {
	l := NewLeader(nil, nil)
	if l == nil {
		t.Fatal("expected non-nil Leader")
	}
	if l.IsLeader() {
		t.Error("new leader should not be leader initially")
	}
}

func TestIsLeader_DefaultFalse(t *testing.T) {
	l := NewLeader(nil, nil)
	if l.IsLeader() {
		t.Error("expected IsLeader() = false by default")
	}
}
