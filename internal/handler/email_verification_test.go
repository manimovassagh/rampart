package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
)

// mockEmailVerificationStore implements EmailVerificationStore for testing.
type mockEmailVerificationStore struct {
	user          *model.User
	tokenConsumed bool
	consumeErr    error
	markErr       error
	settings      *model.OrgSettings
}

func (m *mockEmailVerificationStore) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *mockEmailVerificationStore) FindUserByEmail(_ context.Context, email string) (*model.User, error) {
	if m.user != nil && m.user.Email == email {
		return m.user, nil
	}
	return nil, nil
}

func (m *mockEmailVerificationStore) GetUserByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if m.user != nil && m.user.ID == id {
		return m.user, nil
	}
	return nil, nil
}

func (m *mockEmailVerificationStore) CreateEmailVerificationToken(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}

func (m *mockEmailVerificationStore) ConsumeEmailVerificationToken(_ context.Context, token string) (uuid.UUID, error) {
	if m.consumeErr != nil {
		return uuid.Nil, m.consumeErr
	}
	if m.tokenConsumed || token == "" {
		return uuid.Nil, fmt.Errorf("invalid token")
	}
	m.tokenConsumed = true
	return m.user.ID, nil
}

func (m *mockEmailVerificationStore) MarkEmailVerified(_ context.Context, _ uuid.UUID) error {
	if m.markErr != nil {
		return m.markErr
	}
	m.user.EmailVerified = true
	return nil
}

func (m *mockEmailVerificationStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.settings, nil
}

// ── stub methods to satisfy store.OrgReader ──

func (m *mockEmailVerificationStore) GetOrganizationByID(_ context.Context, _ uuid.UUID) (*model.Organization, error) {
	return nil, nil
}
func (m *mockEmailVerificationStore) GetOrganizationIDBySlug(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// ── stub methods to satisfy store.UserReader ──

func (m *mockEmailVerificationStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockEmailVerificationStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}

// ── stub methods to satisfy store.EmailVerificationTokenStore ──

func (m *mockEmailVerificationStore) DeleteExpiredEmailVerificationTokens(_ context.Context) (int64, error) {
	return 0, nil
}

// ── stub methods to satisfy store.OrgSettingsReadWriter ──

func (m *mockEmailVerificationStore) UpdateOrgSettings(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	return nil, nil
}

type noopEmailSender struct{}

func (n *noopEmailSender) Send(_, _, _ string) error { return nil }
func (n *noopEmailSender) Enabled() bool             { return false }

func TestVerifyEmail_Success(t *testing.T) {
	userID := uuid.New()
	store := &mockEmailVerificationStore{
		user: &model.User{
			ID:      userID,
			Email:   "test@example.com",
			Enabled: true,
		},
	}

	h := NewEmailVerificationHandler(store, &noopEmailSender{}, noopLogger(), "http://localhost")

	req := httptest.NewRequest(http.MethodGet, "/verify-email?token=validtoken123", http.NoBody)
	w := httptest.NewRecorder()
	h.VerifyEmail(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["message"] == "" {
		t.Error("expected non-empty message")
	}
	if !store.user.EmailVerified {
		t.Error("expected email_verified to be true")
	}
}

func TestVerifyEmail_MissingToken(t *testing.T) {
	h := NewEmailVerificationHandler(&mockEmailVerificationStore{}, &noopEmailSender{}, noopLogger(), "http://localhost")

	req := httptest.NewRequest(http.MethodGet, "/verify-email", http.NoBody)
	w := httptest.NewRecorder()
	h.VerifyEmail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	store := &mockEmailVerificationStore{
		consumeErr: fmt.Errorf("invalid token"),
	}

	h := NewEmailVerificationHandler(store, &noopEmailSender{}, noopLogger(), "http://localhost")

	req := httptest.NewRequest(http.MethodGet, "/verify-email?token=badtoken", http.NoBody)
	w := httptest.NewRecorder()
	h.VerifyEmail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestSendVerification_AlwaysReturns200(t *testing.T) {
	h := NewEmailVerificationHandler(&mockEmailVerificationStore{}, &noopEmailSender{}, noopLogger(), "http://localhost")

	body := `{"email":"nonexistent@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/verify-email/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.SendVerification(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (email enumeration prevention)", w.Code)
	}
}

func TestBuildVerificationEmail_HTMLEscapesName(t *testing.T) {
	body := buildVerificationEmail(`<script>alert("xss")</script>`, "https://example.com/verify?token=abc")

	if strings.Contains(body, "<script>") {
		t.Fatal("expected <script> tag to be HTML-escaped, but found raw <script> in email body")
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatal("expected HTML-escaped name (&lt;script&gt;) in email body")
	}
	if !strings.Contains(body, "Hi &lt;script&gt;") {
		t.Fatal("expected escaped name after 'Hi ' greeting")
	}
}

func TestBuildVerificationEmail_NormalNameUnchanged(t *testing.T) {
	body := buildVerificationEmail("Alice", "https://example.com/verify?token=abc")
	if !strings.Contains(body, "Hi Alice,") {
		t.Fatal("expected normal name to appear unchanged in email body")
	}
}

func TestBuildVerificationEmail_AmpersandEscaped(t *testing.T) {
	body := buildVerificationEmail("Tom & Jerry", "https://example.com/verify?token=abc")
	if strings.Contains(body, "Tom & Jerry") {
		t.Fatal("expected ampersand to be escaped")
	}
	if !strings.Contains(body, "Tom &amp; Jerry") {
		t.Fatal("expected &amp; in email body")
	}
}

func TestSendVerification_MissingEmail(t *testing.T) {
	h := NewEmailVerificationHandler(&mockEmailVerificationStore{}, &noopEmailSender{}, noopLogger(), "http://localhost")

	body := `{"email":""}`
	req := httptest.NewRequest(http.MethodPost, "/verify-email/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.SendVerification(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
