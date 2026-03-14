package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
)

func TestWebAuthnCredentialCRUD(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)
	user := testUser(t, db, org.ID)

	credID := []byte("test-credential-id-" + uniqueSlug(""))
	cred := &model.WebAuthnCredential{
		UserID:          user.ID,
		CredentialID:    credID,
		PublicKey:       []byte("test-public-key"),
		AttestationType: "none",
		Transport:       []string{"internal"},
		FlagsRaw:        0x01,
		AAGUID:          []byte("test-aaguid-1234"),
		SignCount:       0,
		Name:            "My Passkey",
	}

	// Create
	err := db.CreateWebAuthnCredential(ctx, cred)
	if err != nil {
		t.Fatalf("CreateWebAuthnCredential: %v", err)
	}

	// Get by user ID
	creds, err := db.GetWebAuthnCredentialsByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetWebAuthnCredentialsByUserID: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if creds[0].Name != "My Passkey" {
		t.Errorf("name: got %q, want %q", creds[0].Name, "My Passkey")
	}
	if creds[0].AttestationType != "none" {
		t.Errorf("attestation_type: got %q, want %q", creds[0].AttestationType, "none")
	}
	if creds[0].SignCount != 0 {
		t.Errorf("sign_count: got %d, want 0", creds[0].SignCount)
	}

	// Count
	count, err := db.CountWebAuthnCredentials(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountWebAuthnCredentials: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	// Update sign count
	err = db.UpdateWebAuthnSignCount(ctx, credID, 5)
	if err != nil {
		t.Fatalf("UpdateWebAuthnSignCount: %v", err)
	}

	creds, _ = db.GetWebAuthnCredentialsByUserID(ctx, user.ID)
	if len(creds) == 1 && creds[0].SignCount != 5 {
		t.Errorf("sign_count after update: got %d, want 5", creds[0].SignCount)
	}

	// Delete
	err = db.DeleteWebAuthnCredential(ctx, creds[0].ID, user.ID)
	if err != nil {
		t.Fatalf("DeleteWebAuthnCredential: %v", err)
	}

	creds, _ = db.GetWebAuthnCredentialsByUserID(ctx, user.ID)
	if len(creds) != 0 {
		t.Errorf("expected 0 credentials after delete, got %d", len(creds))
	}
}

func TestWebAuthnCredentialsEmptyForUnknownUser(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	creds, err := db.GetWebAuthnCredentialsByUserID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetWebAuthnCredentialsByUserID: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("expected 0 credentials for unknown user, got %d", len(creds))
	}

	count, err := db.CountWebAuthnCredentials(ctx, uuid.New())
	if err != nil {
		t.Fatalf("CountWebAuthnCredentials: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0 for unknown user, got %d", count)
	}
}

func TestWebAuthnSessionData(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)
	user := testUser(t, db, org.ID)

	sessionData := []byte(`{"challenge":"abc123","user_id":"test"}`)

	// Store session data
	err := db.StoreWebAuthnSessionData(ctx, user.ID, sessionData, "registration", time.Now().Add(5*time.Minute))
	if err != nil {
		t.Fatalf("StoreWebAuthnSessionData: %v", err)
	}

	// Retrieve (one-time use)
	retrieved, err := db.GetWebAuthnSessionData(ctx, user.ID, "registration")
	if err != nil {
		t.Fatalf("GetWebAuthnSessionData: %v", err)
	}
	if string(retrieved) != string(sessionData) {
		t.Errorf("session_data: got %q, want %q", string(retrieved), string(sessionData))
	}

	// Second retrieval should fail (already consumed)
	_, err = db.GetWebAuthnSessionData(ctx, user.ID, "registration")
	if err == nil {
		t.Error("expected error on second retrieval (session already consumed)")
	}
}

func TestWebAuthnSessionDataOverwrite(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	org := testOrg(t, db)
	user := testUser(t, db, org.ID)

	// Store first
	err := db.StoreWebAuthnSessionData(ctx, user.ID, []byte("first"), "authentication", time.Now().Add(5*time.Minute))
	if err != nil {
		t.Fatalf("StoreWebAuthnSessionData (1): %v", err)
	}

	// Store second (should overwrite first for same ceremony)
	err = db.StoreWebAuthnSessionData(ctx, user.ID, []byte("second"), "authentication", time.Now().Add(5*time.Minute))
	if err != nil {
		t.Fatalf("StoreWebAuthnSessionData (2): %v", err)
	}

	// Should get second
	data, err := db.GetWebAuthnSessionData(ctx, user.ID, "authentication")
	if err != nil {
		t.Fatalf("GetWebAuthnSessionData: %v", err)
	}
	if string(data) != "second" {
		t.Errorf("session_data: got %q, want %q", string(data), "second")
	}
}

func TestDeleteExpiredWebAuthnSessions(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	_, err := db.DeleteExpiredWebAuthnSessions(ctx)
	if err != nil {
		t.Fatalf("DeleteExpiredWebAuthnSessions: %v", err)
	}
}
