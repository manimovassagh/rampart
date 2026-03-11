package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/google/uuid"
	"github.com/manimovassagh/rampart/internal/model"
)

type mockResetStore struct {
	user        *model.User
	tokenErr    error
	userID      uuid.UUID
	orgSettings *model.OrgSettings
}

func (m *mockResetStore) FindUserByEmail(_ context.Context, _ string) (*model.User, error) {
	return m.user, nil
}

func (m *mockResetStore) CreatePasswordResetToken(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}

func (m *mockResetStore) ConsumePasswordResetToken(_ context.Context, _ string) (uuid.UUID, error) {
	if m.tokenErr != nil {
		return uuid.Nil, m.tokenErr
	}
	return m.userID, nil
}

func (m *mockResetStore) UpdatePassword(_ context.Context, _, _ uuid.UUID, _ []byte) error {
	return nil
}

// ── stub methods to satisfy store.OrgSettingsReadWriter ──

func (m *mockResetStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.orgSettings, nil
}
func (m *mockResetStore) UpdateOrgSettings(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	return nil, nil
}

// ── stub methods to satisfy store.UserReader ──

func (m *mockResetStore) GetUserByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if m.user != nil {
		return m.user, nil
	}
	return &model.User{ID: id, OrgID: uuid.New(), Enabled: true}, nil
}
func (m *mockResetStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockResetStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}

// ── stub methods to satisfy store.UserWriter ──

func (m *mockResetStore) CreateUser(_ context.Context, _ *model.User) (*model.User, error) {
	return nil, nil
}
func (m *mockResetStore) UpdateUser(_ context.Context, _, _ uuid.UUID, _ *model.UpdateUserRequest) (*model.User, error) {
	return nil, nil
}
func (m *mockResetStore) DeleteUser(_ context.Context, _, _ uuid.UUID) error     { return nil }
func (m *mockResetStore) UpdateLastLoginAt(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockResetStore) IncrementFailedLogins(_ context.Context, _ uuid.UUID, _ int, _ time.Duration) error {
	return nil
}
func (m *mockResetStore) ResetFailedLogins(_ context.Context, _ uuid.UUID) error { return nil }

// ── stub methods to satisfy store.PasswordResetTokenStore ──

func (m *mockResetStore) DeleteExpiredPasswordResetTokens(_ context.Context) (int64, error) {
	return 0, nil
}

type mockEmailSender struct {
	sent    bool
	enabled bool
}

func (m *mockEmailSender) Send(_, _, _ string) error { m.sent = true; return nil }
func (m *mockEmailSender) Enabled() bool             { return m.enabled }

func TestForgotPasswordAlwaysReturns200(t *testing.T) {
	store := &mockResetStore{user: nil} // no user found
	sender := &mockEmailSender{enabled: false}
	h := NewPasswordResetHandler(store, sender, slog.Default(), "http://localhost:8080")

	body := `{"email":"nonexistent@test.com"}`
	req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ForgotPassword(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestForgotPasswordEmptyEmail(t *testing.T) {
	store := &mockResetStore{}
	sender := &mockEmailSender{}
	h := NewPasswordResetHandler(store, sender, slog.Default(), "http://localhost:8080")

	body := `{"email":""}`
	req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ForgotPassword(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestResetPasswordSuccess(t *testing.T) {
	uid := uuid.New()
	store := &mockResetStore{userID: uid}
	sender := &mockEmailSender{}
	h := NewPasswordResetHandler(store, sender, slog.Default(), "http://localhost:8080")

	body := `{"token":"abc123","new_password":"NewSecure1234!"}`
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ResetPassword(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["message"] == "" {
		t.Fatal("expected success message")
	}
}

func TestResetPasswordShortPassword(t *testing.T) {
	store := &mockResetStore{userID: uuid.New()}
	sender := &mockEmailSender{}
	h := NewPasswordResetHandler(store, sender, slog.Default(), "http://localhost:8080")

	body := `{"token":"abc123","new_password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ResetPassword(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestResetPasswordInvalidToken(t *testing.T) {
	store := &mockResetStore{tokenErr: errInvalidToken}
	sender := &mockEmailSender{}
	h := NewPasswordResetHandler(store, sender, slog.Default(), "http://localhost:8080")

	body := `{"token":"badtoken","new_password":"ValidPass123!"}`
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ResetPassword(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestResetPasswordRejectsWeakPasswordPerOrgPolicy(t *testing.T) {
	uid := uuid.New()
	orgID := uuid.New()

	ms := &mockResetStore{
		userID: uid,
		orgSettings: &model.OrgSettings{
			OrgID:                    orgID,
			PasswordMinLength:        12,
			PasswordRequireUppercase: true,
			PasswordRequireLowercase: true,
			PasswordRequireNumbers:   true,
			PasswordRequireSymbols:   true,
		},
	}

	// Override GetUserByID to return a user with our specific orgID.
	ms.user = &model.User{ID: uid, OrgID: orgID, Enabled: true}

	sender := &mockEmailSender{}
	h := NewPasswordResetHandler(ms, sender, slog.Default(), "http://localhost:8080")

	tests := []struct {
		name     string
		password string
		wantCode int
	}{
		{"too short for org policy", "Abc1!", http.StatusBadRequest},
		{"no uppercase", "abcdefgh1234!", http.StatusBadRequest},
		{"no digit", "Abcdefghijklm!", http.StatusBadRequest},
		{"no symbol", "Abcdefgh1234", http.StatusBadRequest},
		{"meets org policy", "Abcdefgh123!", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"token":"valid-token","new_password":"` + tt.password + `"}`
			req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			h.ResetPassword(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("password %q: expected %d, got %d: %s", tt.password, tt.wantCode, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestBuildResetEmail_HTMLEscapesName(t *testing.T) {
	body := buildResetEmail(`<script>alert("xss")</script>`, "https://example.com/reset?token=abc")

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

func TestBuildResetEmail_NormalNameUnchanged(t *testing.T) {
	body := buildResetEmail("Alice", "https://example.com/reset?token=abc")
	if !strings.Contains(body, "Hi Alice,") {
		t.Fatal("expected normal name to appear unchanged in email body")
	}
}

func TestBuildResetEmail_AmpersandEscaped(t *testing.T) {
	body := buildResetEmail("Tom & Jerry", "https://example.com/reset?token=abc")
	if strings.Contains(body, "Tom & Jerry") {
		t.Fatal("expected ampersand to be escaped")
	}
	if !strings.Contains(body, "Tom &amp; Jerry") {
		t.Fatal("expected &amp; in email body")
	}
}

var errInvalidToken = &resetError{"invalid token"}

type resetError struct{ msg string }

func (e *resetError) Error() string { return e.msg }
