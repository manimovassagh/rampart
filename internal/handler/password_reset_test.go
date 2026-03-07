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
	user     *model.User
	tokenErr error
	userID   uuid.UUID
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

func (m *mockResetStore) UpdatePassword(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
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
	json.NewDecoder(rr.Body).Decode(&resp)
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

var errInvalidToken = &resetErr{"invalid token"}

type resetErr struct{ msg string }

func (e *resetErr) Error() string { return e.msg }
