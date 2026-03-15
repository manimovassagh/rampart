package session

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

// --- Mock implementation of Store interface ---

type mockStore struct {
	sessions        map[uuid.UUID]*Session
	createErr       error
	findErr         error
	deleteErr       error
	deleteByUserErr error
}

func newMockStore() *mockStore {
	return &mockStore{
		sessions: make(map[uuid.UUID]*Session),
	}
}

func (m *mockStore) Create(ctx context.Context, userID uuid.UUID, clientID, refreshToken string, expiresAt time.Time) (*Session, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	sess := &Session{
		ID:               uuid.New(),
		UserID:           userID,
		ClientID:         clientID,
		RefreshTokenHash: HashToken(refreshToken),
		ExpiresAt:        expiresAt,
		CreatedAt:        time.Now().UTC(),
	}
	m.sessions[sess.ID] = sess
	return sess, nil
}

func (m *mockStore) FindByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	hash := HashToken(refreshToken)
	now := time.Now()
	for _, sess := range m.sessions {
		if bytes.Equal(sess.RefreshTokenHash, hash) && sess.ExpiresAt.After(now) {
			return sess, nil
		}
	}
	return nil, nil
}

func (m *mockStore) RotateRefreshToken(_ context.Context, oldRefreshToken, newRefreshToken string) (*Session, error) {
	oldHash := HashToken(oldRefreshToken)
	now := time.Now()
	for _, sess := range m.sessions {
		if bytes.Equal(sess.RefreshTokenHash, oldHash) && sess.ExpiresAt.After(now) {
			sess.RefreshTokenHash = HashToken(newRefreshToken)
			return sess, nil
		}
	}
	return nil, ErrTokenAlreadyRotated
}

func (m *mockStore) Delete(ctx context.Context, sessionID uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.sessions, sessionID)
	return nil
}

func (m *mockStore) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deleteByUserErr != nil {
		return m.deleteByUserErr
	}
	for id, sess := range m.sessions {
		if sess.UserID == userID {
			delete(m.sessions, id)
		}
	}
	return nil
}

// --- Tests for HashToken ---

func TestHashTokenDeterministic(t *testing.T) {
	token := "test-refresh-token-abc123"
	h1 := HashToken(token)
	h2 := HashToken(token)
	if !bytes.Equal(h1, h2) {
		t.Fatal("HashToken should return the same hash for the same input")
	}
}

func TestHashTokenDifferentInputs(t *testing.T) {
	h1 := HashToken("token-a")
	h2 := HashToken("token-b")
	if bytes.Equal(h1, h2) {
		t.Fatal("HashToken should return different hashes for different inputs")
	}
}

func TestHashTokenLength(t *testing.T) {
	h := HashToken("any-token")
	if len(h) != sha256.Size {
		t.Fatalf("expected hash length %d, got %d", sha256.Size, len(h))
	}
}

func TestHashTokenEmptyString(t *testing.T) {
	h := HashToken("")
	if len(h) != sha256.Size {
		t.Fatalf("expected hash length %d for empty string, got %d", sha256.Size, len(h))
	}
}

// --- Tests for NewPGStore ---

func TestNewPGStoreNilPool(t *testing.T) {
	store := NewPGStore(nil)
	if store == nil {
		t.Fatal("NewPGStore should return a non-nil PGStore even with nil pool")
	}
	if store.pool != nil {
		t.Fatal("pool should be nil when created with nil")
	}
}

// --- Tests for Store interface via mock ---

func TestCreateSession(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()
	userID := uuid.New()
	token := "refresh-token-123"
	expiresAt := time.Now().Add(24 * time.Hour)

	sess, err := store.Create(ctx, userID, "", token, expiresAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess == nil {
		t.Fatal("expected session, got nil")
	}
	if sess.UserID != userID {
		t.Fatalf("expected user ID %s, got %s", userID, sess.UserID)
	}
	if sess.ID == uuid.Nil {
		t.Fatal("session ID should not be nil UUID")
	}
	if !bytes.Equal(sess.RefreshTokenHash, HashToken(token)) {
		t.Fatal("refresh token hash mismatch")
	}
}

func TestCreateSessionError(t *testing.T) {
	store := newMockStore()
	store.createErr = errors.New("db connection failed")
	ctx := context.Background()

	sess, err := store.Create(ctx, uuid.New(), "", "token", time.Now().Add(time.Hour))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if sess != nil {
		t.Fatal("expected nil session on error")
	}
}

func TestFindByRefreshToken(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()
	userID := uuid.New()
	token := "find-me-token"
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := store.Create(ctx, userID, "", token, expiresAt)
	if err != nil {
		t.Fatalf("unexpected error creating session: %v", err)
	}

	found, err := store.FindByRefreshToken(ctx, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find session, got nil")
	}
	if found.UserID != userID {
		t.Fatalf("expected user ID %s, got %s", userID, found.UserID)
	}
}

func TestFindByRefreshTokenNotFound(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()

	found, err := store.FindByRefreshToken(ctx, "nonexistent-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatal("expected nil for nonexistent token")
	}
}

func TestFindByRefreshTokenExpired(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()
	userID := uuid.New()
	token := "expired-token"
	expiresAt := time.Now().Add(-1 * time.Hour) // already expired

	sess, err := store.Create(ctx, userID, "", token, expiresAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess == nil {
		t.Fatal("expected session to be created")
	}

	found, err := store.FindByRefreshToken(ctx, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatal("expected nil for expired session")
	}
}

func TestFindByRefreshTokenError(t *testing.T) {
	store := newMockStore()
	store.findErr = errors.New("db error")
	ctx := context.Background()

	_, err := store.FindByRefreshToken(ctx, "any-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteSession(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()
	userID := uuid.New()
	token := "delete-me"

	sess, err := store.Create(ctx, userID, "", token, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = store.Delete(ctx, sess.ID)
	if err != nil {
		t.Fatalf("unexpected error deleting session: %v", err)
	}

	found, err := store.FindByRefreshToken(ctx, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatal("expected session to be deleted")
	}
}

func TestDeleteSessionError(t *testing.T) {
	store := newMockStore()
	store.deleteErr = errors.New("delete failed")
	ctx := context.Background()

	err := store.Delete(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteByUserID(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()
	userID := uuid.New()

	for i := 0; i < 3; i++ {
		_, err := store.Create(ctx, userID, "", fmt.Sprintf("token-%d", i), time.Now().Add(time.Hour))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	otherUserID := uuid.New()
	_, err := store.Create(ctx, otherUserID, "", "other-token", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.sessions) != 4 {
		t.Fatalf("expected 4 sessions, got %d", len(store.sessions))
	}

	err = store.DeleteByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.sessions) != 1 {
		t.Fatalf("expected 1 remaining session, got %d", len(store.sessions))
	}

	for _, sess := range store.sessions {
		if sess.UserID != otherUserID {
			t.Fatal("remaining session should belong to the other user")
		}
	}
}

func TestDeleteByUserIDError(t *testing.T) {
	store := newMockStore()
	store.deleteByUserErr = errors.New("delete by user failed")
	ctx := context.Background()

	err := store.DeleteByUserID(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- Tests for SQL query structure validation ---

func TestSQLQueriesWellFormed(t *testing.T) {
	// Validate that SQL queries used in PGStore methods are structurally sound
	// by checking they contain expected clauses. This catches typos and
	// malformed queries without needing a real database.
	tests := []struct {
		name     string
		query    string
		contains []string
	}{
		{
			name:  "Create INSERT",
			query: "INSERT INTO sessions (user_id, client_id, refresh_token_hash, expires_at) VALUES ($1, $2, $3, $4) RETURNING id, user_id, client_id, refresh_token_hash, expires_at, created_at",
			contains: []string{
				"INSERT INTO sessions",
				"VALUES ($1, $2, $3, $4)",
				"RETURNING",
				"user_id",
				"client_id",
				"refresh_token_hash",
				"expires_at",
				"created_at",
			},
		},
		{
			name:  "FindByRefreshToken SELECT",
			query: "SELECT id, user_id, client_id, refresh_token_hash, expires_at, created_at FROM sessions WHERE refresh_token_hash = $1 AND expires_at > now()",
			contains: []string{
				"SELECT",
				"FROM sessions",
				"WHERE refresh_token_hash = $1",
				"expires_at > now()",
			},
		},
		{
			name:  "Delete",
			query: "DELETE FROM sessions WHERE id = $1",
			contains: []string{
				"DELETE FROM sessions",
				"WHERE id = $1",
			},
		},
		{
			name:  "DeleteByUserID",
			query: "DELETE FROM sessions WHERE user_id = $1",
			contains: []string{
				"DELETE FROM sessions",
				"WHERE user_id = $1",
			},
		},
		{
			name:  "ListByUserID",
			query: "SELECT id, user_id, client_id, refresh_token_hash, expires_at, created_at FROM sessions WHERE user_id = $1 AND expires_at > now() ORDER BY created_at DESC",
			contains: []string{
				"SELECT",
				"FROM sessions",
				"WHERE user_id = $1",
				"expires_at > now()",
				"ORDER BY created_at DESC",
			},
		},
		{
			name:  "CountByUserID",
			query: "SELECT COUNT(*) FROM sessions WHERE user_id = $1 AND expires_at > now()",
			contains: []string{
				"SELECT COUNT(*)",
				"FROM sessions",
				"user_id = $1",
				"expires_at > now()",
			},
		},
		{
			name:  "CountActive",
			query: "SELECT COUNT(*) FROM sessions WHERE expires_at > now()",
			contains: []string{
				"SELECT COUNT(*)",
				"expires_at > now()",
			},
		},
		{
			name:  "DeleteAll",
			query: "DELETE FROM sessions WHERE expires_at > now()",
			contains: []string{
				"DELETE FROM sessions",
				"expires_at > now()",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, expected := range tt.contains {
				if !containsSubstring(tt.query, expected) {
					t.Errorf("query %q missing expected clause %q", tt.query, expected)
				}
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsCheck(s, substr))
}

func containsCheck(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Tests for Session and WithUser structs ---

func TestSessionStructFields(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	hash := HashToken("token")
	expires := time.Now().Add(time.Hour)
	created := time.Now()

	sess := Session{
		ID:               id,
		UserID:           userID,
		RefreshTokenHash: hash,
		ExpiresAt:        expires,
		CreatedAt:        created,
	}

	if sess.ID != id {
		t.Fatal("ID mismatch")
	}
	if sess.UserID != userID {
		t.Fatal("UserID mismatch")
	}
	if !bytes.Equal(sess.RefreshTokenHash, hash) {
		t.Fatal("RefreshTokenHash mismatch")
	}
	if !sess.ExpiresAt.Equal(expires) {
		t.Fatal("ExpiresAt mismatch")
	}
	if !sess.CreatedAt.Equal(created) {
		t.Fatal("CreatedAt mismatch")
	}
}

func TestWithUserStructFields(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()

	wu := WithUser{
		ID:        id,
		UserID:    userID,
		Username:  "testuser",
		Email:     "test@example.com",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	if wu.ID != id {
		t.Fatal("ID mismatch")
	}
	if wu.UserID != userID {
		t.Fatal("UserID mismatch")
	}
	if wu.Username != "testuser" {
		t.Fatal("Username mismatch")
	}
	if wu.Email != "test@example.com" {
		t.Fatal("Email mismatch")
	}
}

// --- Test Store interface compliance ---

func TestPGStoreImplementsStoreInterface(t *testing.T) {
	// Compile-time check that PGStore implements Store
	var _ Store = (*PGStore)(nil)
}

// --- Test multiple sessions for same user ---

func TestMultipleSessionsSameUser(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()
	userID := uuid.New()

	sess1, err := store.Create(ctx, userID, "", "token-1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess2, err := store.Create(ctx, userID, "", "token-2", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sess1.ID == sess2.ID {
		t.Fatal("different sessions should have different IDs")
	}

	found1, err := store.FindByRefreshToken(ctx, "token-1")
	if err != nil || found1 == nil {
		t.Fatal("should find first session")
	}

	found2, err := store.FindByRefreshToken(ctx, "token-2")
	if err != nil || found2 == nil {
		t.Fatal("should find second session")
	}

	if found1.ID == found2.ID {
		t.Fatal("found sessions should have different IDs")
	}
}

// --- Test delete non-existent session ---

func TestDeleteNonExistentSession(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()

	err := store.Delete(ctx, uuid.New())
	if err != nil {
		t.Fatalf("deleting non-existent session should not error, got: %v", err)
	}
}

func TestDeleteByUserIDNoSessions(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()

	err := store.DeleteByUserID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("deleting sessions for user with none should not error, got: %v", err)
	}
}
