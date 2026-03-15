package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/session"
)

const testRefreshBody = "grant_type=refresh_token&refresh_token=sometoken" //nolint:goconst // test constant, intentionally duplicated

// ────────────────────────────────────────────────────────────────────────────
// Login handler: uncovered edge cases
// ────────────────────────────────────────────────────────────────────────────

func TestAuthFlowExtra_LoginLockedAccount(t *testing.T) {
	user := newTestUser()
	lockUntil := time.Now().Add(10 * time.Minute)
	user.LockedUntil = &lockUntil
	store := &mockLoginStore{
		defaultOrgID: user.OrgID,
		emailUser:    user,
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier":"admin@rampart.local","password":"Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(w.Body.String(), "temporarily locked") {
		t.Errorf("expected locked message, got %s", w.Body.String())
	}
}

func TestAuthFlowExtra_LoginSSOOnlyAccount(t *testing.T) {
	user := newTestUser()
	user.PasswordHash = []byte{} // SSO-only user has no local password
	store := &mockLoginStore{
		defaultOrgID: user.OrgID,
		emailUser:    user,
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier":"admin@rampart.local","password":"Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(w.Body.String(), "SSO") {
		t.Errorf("expected SSO message, got %s", w.Body.String())
	}
}

func TestAuthFlowExtra_LoginEmailNotVerifiedBlocked(t *testing.T) {
	user := newTestUser()
	user.EmailVerified = false
	store := &mockLoginStore{
		defaultOrgID: user.OrgID,
		emailUser:    user,
		orgSettings: &model.OrgSettings{
			EmailVerificationRequired: true,
		},
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier":"admin@rampart.local","password":"Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if !strings.Contains(w.Body.String(), "email_not_verified") {
		t.Errorf("expected email_not_verified code, got %s", w.Body.String())
	}
}

func TestAuthFlowExtra_LoginEmailVerifiedAllowed(t *testing.T) {
	user := newTestUser()
	user.EmailVerified = true
	store := &mockLoginStore{
		defaultOrgID: user.OrgID,
		emailUser:    user,
		orgSettings: &model.OrgSettings{
			EmailVerificationRequired: true,
		},
	}
	sessions := &mockSessionStore{}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"identifier":"admin@rampart.local","password":"Str0ng!Pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Token handler: OAuth refresh_token grant type (handleRefreshToken)
// ────────────────────────────────────────────────────────────────────────────

func newTestTokenHandler(s *mockTokenStore, sess *mockSessionStore) *TokenHandler {
	return NewTokenHandler(s, sess, noopLogger(), testPrivKey, testKID, testIssuer, 15*time.Minute, 7*24*time.Hour)
}

func TestAuthFlowExtra_OAuthRefreshTokenGrant_Success(t *testing.T) {
	user := &model.User{
		ID:       uuid.New(),
		OrgID:    uuid.New(),
		Username: "admin",
		Email:    "admin@test.com",
		Enabled:  true,
	}
	sess := &session.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	store := &mockTokenStore{userByID: user}
	sessions := &mockSessionStore{found: sess}
	h := newTestTokenHandler(store, sessions)

	body := "grant_type=refresh_token&refresh_token=valid-refresh-token"
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp TokenResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh_token")
	}
	if resp.TokenType != tokenTypeBearer {
		t.Errorf("token_type = %q, want Bearer", resp.TokenType)
	}
}

func TestAuthFlowExtra_OAuthRefreshTokenGrant_MissingToken(t *testing.T) {
	h := newTestTokenHandler(&mockTokenStore{}, &mockSessionStore{})

	body := "grant_type=refresh_token"
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthFlowExtra_OAuthRefreshTokenGrant_InvalidSession(t *testing.T) {
	h := newTestTokenHandler(&mockTokenStore{}, &mockSessionStore{found: nil})

	body := "grant_type=refresh_token&refresh_token=bogus"
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthFlowExtra_OAuthRefreshTokenGrant_SessionFindError(t *testing.T) {
	h := newTestTokenHandler(&mockTokenStore{}, &mockSessionStore{findErr: fmt.Errorf("db error")})

	body := testRefreshBody
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAuthFlowExtra_OAuthRefreshTokenGrant_DisabledUser(t *testing.T) {
	user := &model.User{
		ID:      uuid.New(),
		OrgID:   uuid.New(),
		Enabled: false,
	}
	sess := &session.Session{ID: uuid.New(), UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)}
	h := newTestTokenHandler(
		&mockTokenStore{userByID: user},
		&mockSessionStore{found: sess},
	)

	body := testRefreshBody
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthFlowExtra_OAuthRefreshTokenGrant_UserNotFound(t *testing.T) {
	sess := &session.Session{ID: uuid.New(), UserID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	h := newTestTokenHandler(
		&mockTokenStore{userByID: nil},
		&mockSessionStore{found: sess},
	)

	body := testRefreshBody
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthFlowExtra_OAuthRefreshTokenGrant_UserFetchError(t *testing.T) {
	sess := &session.Session{ID: uuid.New(), UserID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	h := newTestTokenHandler(
		&mockTokenStore{userByIDErr: fmt.Errorf("db error")},
		&mockSessionStore{found: sess},
	)

	body := testRefreshBody
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Token(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Token handler: Revoke edge cases
// ────────────────────────────────────────────────────────────────────────────

func TestAuthFlowExtra_RevokeFindSessionError(t *testing.T) {
	sessions := &mockSessionStore{findErr: fmt.Errorf("session db down")}
	h := newTestTokenHandler(&mockTokenStore{}, sessions)

	body := "token=some-refresh-token"
	req := httptest.NewRequest(http.MethodPost, "/oauth/revoke", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Revoke(w, req)

	// Per RFC 7009: respond 200 even on server errors for token lookup
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthFlowExtra_RevokeDeleteSessionError(t *testing.T) {
	sess := &session.Session{ID: uuid.New(), UserID: uuid.New()}
	sessions := &mockSessionStore{found: sess, deleteErr: fmt.Errorf("delete failed")}
	h := newTestTokenHandler(&mockTokenStore{}, sessions)

	body := "token=valid-refresh-token"
	req := httptest.NewRequest(http.MethodPost, "/oauth/revoke", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Revoke(w, req)

	// Per RFC 7009: still returns 200 even if delete fails
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Me handler: edge cases
// ────────────────────────────────────────────────────────────────────────────

func TestAuthFlowExtra_MeSocialAccountStoreError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	authUser := &middleware.AuthenticatedUser{
		UserID:            userID,
		OrgID:             orgID,
		PreferredUsername: "testuser",
		Email:             "test@example.com",
		EmailVerified:     true,
	}

	store := &mockMeStore{err: fmt.Errorf("db error fetching social accounts")}

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h := newTestMeHandler(store)
	h.Me(w, req)

	// Should still return 200 with user info, social accounts just omitted on error
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp MeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != userID.String() {
		t.Errorf("id = %q, want %q", resp.ID, userID.String())
	}
	if resp.SocialAccounts != nil {
		t.Errorf("expected nil social accounts on store error, got %v", resp.SocialAccounts)
	}
}

func TestAuthFlowExtra_MeEmptySocialAccounts(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	authUser := &middleware.AuthenticatedUser{
		UserID:            userID,
		OrgID:             orgID,
		PreferredUsername: "testuser",
		Email:             "test@example.com",
	}

	store := &mockMeStore{accounts: []*model.SocialAccount{}} // empty, not nil

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h := newTestMeHandler(store)
	h.Me(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp MeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	// Empty slice should result in omitted (nil) social_accounts due to omitempty
	if resp.SocialAccounts != nil {
		t.Errorf("expected nil social accounts for empty list, got %v", resp.SocialAccounts)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Password reset handler: edge cases
// ────────────────────────────────────────────────────────────────────────────

func TestAuthFlowExtra_ForgotPasswordInvalidJSON(t *testing.T) {
	store := &mockResetStore{}
	sender := &mockEmailSender{}
	h := NewPasswordResetHandler(store, &mockResetSessionStore{}, sender, noopLogger(), nil, "http://localhost:8080")

	req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ForgotPassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthFlowExtra_ResetPasswordInvalidJSON(t *testing.T) {
	store := &mockResetStore{}
	sender := &mockEmailSender{}
	h := NewPasswordResetHandler(store, &mockResetSessionStore{}, sender, noopLogger(), nil, "http://localhost:8080")

	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ResetPassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthFlowExtra_ResetPasswordMissingToken(t *testing.T) {
	store := &mockResetStore{}
	sender := &mockEmailSender{}
	h := NewPasswordResetHandler(store, &mockResetSessionStore{}, sender, noopLogger(), nil, "http://localhost:8080")

	body := `{"token":"","new_password":"ValidPass123!"}`
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ResetPassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthFlowExtra_ResetPasswordMissingNewPassword(t *testing.T) {
	store := &mockResetStore{}
	sender := &mockEmailSender{}
	h := NewPasswordResetHandler(store, &mockResetSessionStore{}, sender, noopLogger(), nil, "http://localhost:8080")

	body := `{"token":"validtoken","new_password":""}`
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ResetPassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthFlowExtra_ResetPasswordEmptyBody(t *testing.T) {
	store := &mockResetStore{}
	sender := &mockEmailSender{}
	h := NewPasswordResetHandler(store, &mockResetSessionStore{}, sender, noopLogger(), nil, "http://localhost:8080")

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ResetPassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthFlowExtra_BuildResetEmailEmptyName(t *testing.T) {
	body := buildResetEmail("", "https://example.com/reset?token=abc")
	if !strings.Contains(body, "Hi there,") {
		t.Fatal("expected default 'Hi there,' for empty name")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Email verification handler: edge cases
// ────────────────────────────────────────────────────────────────────────────

func TestAuthFlowExtra_SendVerificationInvalidJSON(t *testing.T) {
	h := NewEmailVerificationHandler(&mockEmailVerificationStore{}, &noopEmailSender{}, noopLogger(), "http://localhost")

	req := httptest.NewRequest(http.MethodPost, "/verify-email/send", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.SendVerification(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthFlowExtra_VerifyEmailMarkError(t *testing.T) {
	userID := uuid.New()
	store := &mockEmailVerificationStore{
		user: &model.User{
			ID:      userID,
			Email:   "test@example.com",
			Enabled: true,
		},
		markErr: fmt.Errorf("db error marking verified"),
	}

	h := NewEmailVerificationHandler(store, &noopEmailSender{}, noopLogger(), "http://localhost")

	req := httptest.NewRequest(http.MethodGet, "/verify-email?token=validtoken123", http.NoBody)
	w := httptest.NewRecorder()
	h.VerifyEmail(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestAuthFlowExtra_VerifyEmailAlreadyConsumedToken(t *testing.T) {
	userID := uuid.New()
	store := &mockEmailVerificationStore{
		user: &model.User{
			ID:      userID,
			Email:   "test@example.com",
			Enabled: true,
		},
		tokenConsumed: true, // simulate already-used token
	}

	h := NewEmailVerificationHandler(store, &noopEmailSender{}, noopLogger(), "http://localhost")

	req := httptest.NewRequest(http.MethodGet, "/verify-email?token=already-used", http.NoBody)
	w := httptest.NewRecorder()
	h.VerifyEmail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthFlowExtra_BuildVerificationEmailEmptyName(t *testing.T) {
	body := buildVerificationEmail("", "https://example.com/verify?token=abc")
	if !strings.Contains(body, "Hi there,") {
		t.Fatal("expected default 'Hi there,' for empty name")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// DefaultLockoutPolicy unit tests
// ────────────────────────────────────────────────────────────────────────────

func TestAuthFlowExtra_DefaultLockoutPolicyNilSettings(t *testing.T) {
	maxAttempts, lockoutDuration := defaultLockoutPolicy(nil)
	if maxAttempts != defaultMaxFailedAttempts {
		t.Errorf("maxAttempts = %d, want %d", maxAttempts, defaultMaxFailedAttempts)
	}
	if lockoutDuration != defaultLockoutDurationMins*time.Minute {
		t.Errorf("lockoutDuration = %v, want %v", lockoutDuration, defaultLockoutDurationMins*time.Minute)
	}
}

func TestAuthFlowExtra_DefaultLockoutPolicyCustomSettings(t *testing.T) {
	settings := &model.OrgSettings{
		MaxFailedLoginAttempts: 10,
		LockoutDuration:        30 * time.Minute,
	}
	maxAttempts, lockoutDuration := defaultLockoutPolicy(settings)
	if maxAttempts != 10 {
		t.Errorf("maxAttempts = %d, want 10", maxAttempts)
	}
	if lockoutDuration != 30*time.Minute {
		t.Errorf("lockoutDuration = %v, want 30m", lockoutDuration)
	}
}

func TestAuthFlowExtra_DefaultLockoutPolicyZeroSettings(t *testing.T) {
	// Zero values in settings should fall back to defaults
	settings := &model.OrgSettings{
		MaxFailedLoginAttempts: 0,
		LockoutDuration:        0,
	}
	maxAttempts, lockoutDuration := defaultLockoutPolicy(settings)
	if maxAttempts != defaultMaxFailedAttempts {
		t.Errorf("maxAttempts = %d, want %d (default)", maxAttempts, defaultMaxFailedAttempts)
	}
	if lockoutDuration != defaultLockoutDurationMins*time.Minute {
		t.Errorf("lockoutDuration = %v, want %v (default)", lockoutDuration, defaultLockoutDurationMins*time.Minute)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Token handler: Revoke with no token parameter at all
// ────────────────────────────────────────────────────────────────────────────

func TestAuthFlowExtra_RevokeNoTokenParam(t *testing.T) {
	h := newTestTokenHandler(&mockTokenStore{}, &mockSessionStore{})

	// Submit form with no token field at all
	req := httptest.NewRequest(http.MethodPost, "/oauth/revoke", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Revoke(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Me handler: multiple social accounts
// ────────────────────────────────────────────────────────────────────────────

func TestAuthFlowExtra_MeMultipleSocialAccounts(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	authUser := &middleware.AuthenticatedUser{
		UserID:            userID,
		OrgID:             orgID,
		PreferredUsername: "socialuser",
		Email:             "user@test.com",
		EmailVerified:     true,
	}

	store := &mockMeStore{
		accounts: []*model.SocialAccount{
			{ID: uuid.New(), UserID: userID, Provider: "google", Email: "user@gmail.com", Name: "G User"},
			{ID: uuid.New(), UserID: userID, Provider: "github", Email: "user@github.com", Name: "GH User"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h := newTestMeHandler(store)
	h.Me(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp MeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.SocialAccounts) != 2 {
		t.Fatalf("social_accounts count = %d, want 2", len(resp.SocialAccounts))
	}
	if resp.SocialAccounts[0].Provider != "google" {
		t.Errorf("first provider = %q, want google", resp.SocialAccounts[0].Provider)
	}
	if resp.SocialAccounts[1].Provider != "github" {
		t.Errorf("second provider = %q, want github", resp.SocialAccounts[1].Provider)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Login handler: Refresh with context.Background to exercise nil audit logger path
// ────────────────────────────────────────────────────────────────────────────

func TestAuthFlowExtra_RefreshWithContextBackground(t *testing.T) {
	user := newTestUser()
	sess := &session.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	store := &mockLoginStore{userByID: user}
	sessions := &mockSessionStore{found: sess}
	h := newTestLoginHandler(store, sessions)

	body := []byte(`{"refresh_token":"some-refresh-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/token/refresh", bytes.NewReader(body))
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}
