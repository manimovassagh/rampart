package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/token"
)

// ══════════════════════════════════════════════════════════════════════════
// WebAuthn handler tests
// ══════════════════════════════════════════════════════════════════════════

// mockWebAuthnStore implements WebAuthnStore for testing.
type mockWebAuthnStore struct {
	userByID              *model.User
	userByIDErr           error
	orgSettings           *model.OrgSettings
	orgSettingsErr        error
	effectiveRoles        []string
	effectiveRolesErr     error
	webauthnCreds         []*model.WebAuthnCredential
	webauthnCredsErr      error
	createCredErr         error
	deleteCredErr         error
	updateSignCountErr    error
	sessionData           []byte
	sessionDataErr        error
	storeSessionErr       error
	verifiedDevice        *model.MFADevice
	verifiedDeviceErr     error
	createDevice          *model.MFADevice
	createDeviceErr       error
	verifyDeviceErr       error
	resetFailedErr        error
	updateLastLoginErr    error
	countWebAuthnCreds    int
	countWebAuthnCredsErr error
}

func (m *mockWebAuthnStore) GetUserByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return m.userByID, m.userByIDErr
}
func (m *mockWebAuthnStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockWebAuthnStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockWebAuthnStore) FindUserByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}

// UserWriter
func (m *mockWebAuthnStore) CreateUser(_ context.Context, _ *model.User) (*model.User, error) {
	return nil, nil
}
func (m *mockWebAuthnStore) UpdateUser(_ context.Context, _, _ uuid.UUID, _ *model.UpdateUserRequest) (*model.User, error) {
	return nil, nil
}
func (m *mockWebAuthnStore) DeleteUser(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockWebAuthnStore) UpdatePassword(_ context.Context, _, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockWebAuthnStore) UpdateLastLoginAt(_ context.Context, _ uuid.UUID) error {
	return m.updateLastLoginErr
}
func (m *mockWebAuthnStore) IncrementFailedLogins(_ context.Context, _ uuid.UUID, _ int, _ time.Duration) error {
	return nil
}
func (m *mockWebAuthnStore) ResetFailedLogins(_ context.Context, _ uuid.UUID) error {
	return m.resetFailedErr
}

// OrgSettingsReadWriter
func (m *mockWebAuthnStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.orgSettings, m.orgSettingsErr
}
func (m *mockWebAuthnStore) UpdateOrgSettings(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	return nil, nil
}

// GroupReader
func (m *mockWebAuthnStore) GetGroupByID(_ context.Context, _ uuid.UUID) (*model.Group, error) {
	return nil, nil
}
func (m *mockWebAuthnStore) GetGroupMembers(_ context.Context, _ uuid.UUID) ([]*model.GroupMember, error) {
	return nil, nil
}
func (m *mockWebAuthnStore) GetGroupRoles(_ context.Context, _ uuid.UUID) ([]*model.GroupRoleAssignment, error) {
	return nil, nil
}
func (m *mockWebAuthnStore) GetUserGroups(_ context.Context, _ uuid.UUID) ([]*model.Group, error) {
	return nil, nil
}
func (m *mockWebAuthnStore) GetEffectiveUserRoles(_ context.Context, _ uuid.UUID) ([]string, error) {
	return m.effectiveRoles, m.effectiveRolesErr
}
func (m *mockWebAuthnStore) CountGroupMembers(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockWebAuthnStore) CountGroupRoles(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockWebAuthnStore) CountGroups(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

// WebAuthnCredentialStore
func (m *mockWebAuthnStore) CreateWebAuthnCredential(_ context.Context, _ *model.WebAuthnCredential) error {
	return m.createCredErr
}
func (m *mockWebAuthnStore) GetWebAuthnCredentialsByUserID(_ context.Context, _ uuid.UUID) ([]*model.WebAuthnCredential, error) {
	return m.webauthnCreds, m.webauthnCredsErr
}
func (m *mockWebAuthnStore) UpdateWebAuthnSignCount(_ context.Context, _ []byte, _ uint32) error {
	return m.updateSignCountErr
}
func (m *mockWebAuthnStore) DeleteWebAuthnCredential(_ context.Context, _, _ uuid.UUID) error {
	return m.deleteCredErr
}
func (m *mockWebAuthnStore) CountWebAuthnCredentials(_ context.Context, _ uuid.UUID) (int, error) {
	return m.countWebAuthnCreds, m.countWebAuthnCredsErr
}

// WebAuthnSessionStore
func (m *mockWebAuthnStore) StoreWebAuthnSessionData(_ context.Context, _ uuid.UUID, _ []byte, _ string, _ time.Time) error {
	return m.storeSessionErr
}
func (m *mockWebAuthnStore) GetWebAuthnSessionData(_ context.Context, _ uuid.UUID, _ string) ([]byte, error) {
	return m.sessionData, m.sessionDataErr
}
func (m *mockWebAuthnStore) DeleteExpiredWebAuthnSessions(_ context.Context) (int64, error) {
	return 0, nil
}

// MFADeviceStore
func (m *mockWebAuthnStore) GetVerifiedMFADevice(_ context.Context, _ uuid.UUID) (*model.MFADevice, error) {
	return m.verifiedDevice, m.verifiedDeviceErr
}
func (m *mockWebAuthnStore) GetPendingMFADevice(_ context.Context, _ uuid.UUID) (*model.MFADevice, error) {
	return nil, nil
}
func (m *mockWebAuthnStore) CreateMFADevice(_ context.Context, _ uuid.UUID, _, _, _ string) (*model.MFADevice, error) {
	return m.createDevice, m.createDeviceErr
}
func (m *mockWebAuthnStore) VerifyMFADevice(_ context.Context, _, _ uuid.UUID) error {
	return m.verifyDeviceErr
}
func (m *mockWebAuthnStore) DeleteUnverifiedMFADevices(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockWebAuthnStore) DisableMFA(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockWebAuthnStore) StoreBackupCodes(_ context.Context, _ uuid.UUID, _ [][]byte) error {
	return nil
}
func (m *mockWebAuthnStore) ConsumeBackupCode(_ context.Context, _ uuid.UUID, _ []byte) (bool, error) {
	return false, nil
}

func newWebAuthnAuthenticatedRequest(target string, body []byte, userID, orgID uuid.UUID) *http.Request { //nolint:unparam // body param used conditionally
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	} else {
		r = httptest.NewRequest(http.MethodPost, target, http.NoBody)
	}
	ctx := middleware.SetAuthenticatedUser(r.Context(), &middleware.AuthenticatedUser{
		UserID:            userID,
		OrgID:             orgID,
		PreferredUsername: "testuser",
		Email:             "test@rampart.local",
	})
	return r.WithContext(ctx)
}

// ── BeginRegistration tests ───────────────────────────────────────────

func TestMfaSamlSocial_WebAuthnBeginRegistrationUnauthenticated(t *testing.T) {
	h := NewWebAuthnHandler(
		&mockWebAuthnStore{},
		&mockSessionStore{},
		noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/register/begin", http.NoBody)
	w := httptest.NewRecorder()

	h.BeginRegistration(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMfaSamlSocial_WebAuthnBeginRegistrationUserNotFound(t *testing.T) {
	s := &mockWebAuthnStore{userByID: nil}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	userID := uuid.New()
	orgID := uuid.New()
	r := newWebAuthnAuthenticatedRequest("/mfa/webauthn/register/begin", nil, userID, orgID)
	w := httptest.NewRecorder()

	h.BeginRegistration(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMfaSamlSocial_WebAuthnBeginRegistrationGetUserError(t *testing.T) {
	s := &mockWebAuthnStore{userByIDErr: fmt.Errorf("db error")}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	userID := uuid.New()
	orgID := uuid.New()
	r := newWebAuthnAuthenticatedRequest("/mfa/webauthn/register/begin", nil, userID, orgID)
	w := httptest.NewRecorder()

	h.BeginRegistration(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMfaSamlSocial_WebAuthnBeginRegistrationGetCredsError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	s := &mockWebAuthnStore{
		userByID: &model.User{
			ID:       userID,
			OrgID:    orgID,
			Username: "testuser",
			Email:    "test@rampart.local",
			Enabled:  true,
		},
		webauthnCredsErr: fmt.Errorf("db error"),
	}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := newWebAuthnAuthenticatedRequest("/mfa/webauthn/register/begin", nil, userID, orgID)
	w := httptest.NewRecorder()

	h.BeginRegistration(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ── FinishRegistration tests ──────────────────────────────────────────

func TestMfaSamlSocial_WebAuthnFinishRegistrationUnauthenticated(t *testing.T) {
	h := NewWebAuthnHandler(
		&mockWebAuthnStore{},
		&mockSessionStore{},
		noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/register/complete", http.NoBody)
	w := httptest.NewRecorder()

	h.FinishRegistration(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMfaSamlSocial_WebAuthnFinishRegistrationUserNotFound(t *testing.T) {
	s := &mockWebAuthnStore{userByID: nil}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	userID := uuid.New()
	orgID := uuid.New()
	r := newWebAuthnAuthenticatedRequest("/mfa/webauthn/register/complete", nil, userID, orgID)
	w := httptest.NewRecorder()

	h.FinishRegistration(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMfaSamlSocial_WebAuthnFinishRegistrationSessionExpired(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	s := &mockWebAuthnStore{
		userByID: &model.User{
			ID:       userID,
			OrgID:    orgID,
			Username: "testuser",
			Email:    "test@rampart.local",
			Enabled:  true,
		},
		sessionDataErr: fmt.Errorf("session not found"),
	}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := newWebAuthnAuthenticatedRequest("/mfa/webauthn/register/complete", nil, userID, orgID)
	w := httptest.NewRecorder()

	h.FinishRegistration(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMfaSamlSocial_WebAuthnFinishRegistrationBadSessionData(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	s := &mockWebAuthnStore{
		userByID: &model.User{
			ID:       userID,
			OrgID:    orgID,
			Username: "testuser",
			Email:    "test@rampart.local",
			Enabled:  true,
		},
		sessionData: []byte("not valid json"),
	}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := newWebAuthnAuthenticatedRequest("/mfa/webauthn/register/complete", nil, userID, orgID)
	w := httptest.NewRecorder()

	h.FinishRegistration(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ── BeginLogin tests ──────────────────────────────────────────────────

func TestMfaSamlSocial_WebAuthnBeginLoginMissingMFAToken(t *testing.T) {
	h := NewWebAuthnHandler(
		&mockWebAuthnStore{},
		&mockSessionStore{},
		noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	cases := []struct {
		name string
		body string
	}{
		{"empty_body", `{}`},
		{"empty_token", `{"mfa_token": ""}`},
		{"invalid_json", `not json`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/login/begin", bytes.NewReader([]byte(tc.body)))
			w := httptest.NewRecorder()

			h.BeginLogin(w, r)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestMfaSamlSocial_WebAuthnBeginLoginInvalidMFAToken(t *testing.T) {
	h := NewWebAuthnHandler(
		&mockWebAuthnStore{},
		&mockSessionStore{},
		noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	body := []byte(`{"mfa_token": "invalid-token"}`)
	r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/login/begin", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.BeginLogin(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMfaSamlSocial_WebAuthnBeginLoginUserNotFound(t *testing.T) {
	userID := uuid.New()
	mfaToken := generateTestMFAToken(userID)

	s := &mockWebAuthnStore{userByID: nil}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	body := fmt.Appendf(nil, `{"mfa_token": "%s"}`, mfaToken)
	r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/login/begin", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.BeginLogin(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMfaSamlSocial_WebAuthnBeginLoginUserDisabled(t *testing.T) {
	userID := uuid.New()
	mfaToken := generateTestMFAToken(userID)

	s := &mockWebAuthnStore{
		userByID: &model.User{
			ID:      userID,
			Enabled: false,
		},
	}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	body := fmt.Appendf(nil, `{"mfa_token": "%s"}`, mfaToken)
	r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/login/begin", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.BeginLogin(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMfaSamlSocial_WebAuthnBeginLoginNoPasskeys(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	mfaToken := generateTestMFAToken(userID)

	s := &mockWebAuthnStore{
		userByID: &model.User{
			ID:       userID,
			OrgID:    orgID,
			Username: "testuser",
			Email:    "test@rampart.local",
			Enabled:  true,
		},
		webauthnCreds: []*model.WebAuthnCredential{}, // empty
	}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	body := fmt.Appendf(nil, `{"mfa_token": "%s"}`, mfaToken)
	r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/login/begin", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.BeginLogin(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "No passkeys") {
		t.Errorf("body = %q, want substring 'No passkeys'", w.Body.String())
	}
}

// ── FinishLogin tests ─────────────────────────────────────────────────

func TestMfaSamlSocial_WebAuthnFinishLoginMissingMFAToken(t *testing.T) {
	h := NewWebAuthnHandler(
		&mockWebAuthnStore{},
		&mockSessionStore{},
		noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	// No mfa_token query param
	r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/login/complete", http.NoBody)
	w := httptest.NewRecorder()

	h.FinishLogin(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMfaSamlSocial_WebAuthnFinishLoginInvalidMFAToken(t *testing.T) {
	h := NewWebAuthnHandler(
		&mockWebAuthnStore{},
		&mockSessionStore{},
		noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/login/complete?mfa_token=invalid-token", http.NoBody)
	w := httptest.NewRecorder()

	h.FinishLogin(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMfaSamlSocial_WebAuthnFinishLoginUserNotFound(t *testing.T) {
	userID := uuid.New()
	mfaToken := generateTestMFAToken(userID)

	s := &mockWebAuthnStore{userByID: nil}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/login/complete?mfa_token="+mfaToken, http.NoBody)
	w := httptest.NewRecorder()

	h.FinishLogin(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMfaSamlSocial_WebAuthnFinishLoginSessionExpired(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	mfaToken := generateTestMFAToken(userID)

	s := &mockWebAuthnStore{
		userByID: &model.User{
			ID:       userID,
			OrgID:    orgID,
			Username: "testuser",
			Email:    "test@rampart.local",
			Enabled:  true,
		},
		sessionDataErr: fmt.Errorf("session expired"),
	}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/login/complete?mfa_token="+mfaToken, http.NoBody)
	w := httptest.NewRecorder()

	h.FinishLogin(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMfaSamlSocial_WebAuthnFinishLoginBadSessionData(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	mfaToken := generateTestMFAToken(userID)

	s := &mockWebAuthnStore{
		userByID: &model.User{
			ID:       userID,
			OrgID:    orgID,
			Username: "testuser",
			Email:    "test@rampart.local",
			Enabled:  true,
		},
		sessionData: []byte("not json"),
	}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/login/complete?mfa_token="+mfaToken, http.NoBody)
	w := httptest.NewRecorder()

	h.FinishLogin(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ── ListCredentials tests ─────────────────────────────────────────────

func TestMfaSamlSocial_WebAuthnListCredentialsUnauthenticated(t *testing.T) {
	h := NewWebAuthnHandler(
		&mockWebAuthnStore{},
		&mockSessionStore{},
		noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := httptest.NewRequest(http.MethodGet, "/mfa/webauthn/credentials", http.NoBody)
	w := httptest.NewRecorder()

	h.ListCredentials(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMfaSamlSocial_WebAuthnListCredentialsEmpty(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	s := &mockWebAuthnStore{
		webauthnCreds: []*model.WebAuthnCredential{},
	}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := newWebAuthnAuthenticatedRequest("/mfa/webauthn/credentials", nil, userID, orgID)
	r.Method = http.MethodGet
	w := httptest.NewRecorder()

	h.ListCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "[]") {
		t.Errorf("body = %q, want empty array", w.Body.String())
	}
}

func TestMfaSamlSocial_WebAuthnListCredentialsStoreError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	s := &mockWebAuthnStore{
		webauthnCredsErr: fmt.Errorf("db error"),
	}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := newWebAuthnAuthenticatedRequest("/mfa/webauthn/credentials", nil, userID, orgID)
	r.Method = http.MethodGet
	w := httptest.NewRecorder()

	h.ListCredentials(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMfaSamlSocial_WebAuthnListCredentialsSuccess(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	credID := uuid.New()
	now := time.Now()
	s := &mockWebAuthnStore{
		webauthnCreds: []*model.WebAuthnCredential{
			{
				ID:        credID,
				UserID:    userID,
				Name:      "My Passkey",
				CreatedAt: now,
			},
		},
	}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	r := newWebAuthnAuthenticatedRequest("/mfa/webauthn/credentials", nil, userID, orgID)
	r.Method = http.MethodGet
	w := httptest.NewRecorder()

	h.ListCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "My Passkey") {
		t.Errorf("body = %q, want substring 'My Passkey'", w.Body.String())
	}
}

// ── DeleteCredential tests ────────────────────────────────────────────

func TestMfaSamlSocial_WebAuthnDeleteCredentialUnauthenticated(t *testing.T) {
	h := NewWebAuthnHandler(
		&mockWebAuthnStore{},
		&mockSessionStore{},
		noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	router := chi.NewRouter()
	router.Delete("/mfa/webauthn/credentials/{id}", h.DeleteCredential)

	req := httptest.NewRequest(http.MethodDelete, "/mfa/webauthn/credentials/"+uuid.New().String(), http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMfaSamlSocial_WebAuthnDeleteCredentialInvalidID(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	h := NewWebAuthnHandler(
		&mockWebAuthnStore{},
		&mockSessionStore{},
		noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	router := chi.NewRouter()
	router.Delete("/mfa/webauthn/credentials/{id}", h.DeleteCredential)

	req := httptest.NewRequest(http.MethodDelete, "/mfa/webauthn/credentials/not-a-uuid", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), &middleware.AuthenticatedUser{
		UserID: userID,
		OrgID:  orgID,
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMfaSamlSocial_WebAuthnDeleteCredentialSuccess(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	credID := uuid.New()
	s := &mockWebAuthnStore{}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	router := chi.NewRouter()
	router.Delete("/mfa/webauthn/credentials/{id}", h.DeleteCredential)

	req := httptest.NewRequest(http.MethodDelete, "/mfa/webauthn/credentials/"+credID.String(), http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), &middleware.AuthenticatedUser{
		UserID: userID,
		OrgID:  orgID,
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "Passkey deleted") {
		t.Errorf("body = %q, want substring 'Passkey deleted'", w.Body.String())
	}
}

func TestMfaSamlSocial_WebAuthnDeleteCredentialStoreError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	credID := uuid.New()
	s := &mockWebAuthnStore{deleteCredErr: fmt.Errorf("db error")}
	h := NewWebAuthnHandler(
		s, &mockSessionStore{}, noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	router := chi.NewRouter()
	router.Delete("/mfa/webauthn/credentials/{id}", h.DeleteCredential)

	req := httptest.NewRequest(http.MethodDelete, "/mfa/webauthn/credentials/"+credID.String(), http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), &middleware.AuthenticatedUser{
		UserID: userID,
		OrgID:  orgID,
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ══════════════════════════════════════════════════════════════════════════
// SAML handler tests
// ══════════════════════════════════════════════════════════════════════════

// mockSAMLStore implements SAMLStore for testing.
type mockSAMLStore struct {
	samlProvider         *model.SAMLProvider
	samlProviderErr      error
	samlProviders        []*model.SAMLProvider
	samlProvidersErr     error
	orgIDBySlug          uuid.UUID
	orgIDBySlugErr       error
	storeSAMLReqErr      error
	consumeSAMLReq       bool
	consumeSAMLReqErr    error
	assertionConsumed    bool
	assertionConsumedErr error
	userByEmail          *model.User
	userByEmailErr       error
	createdUser          *model.User
	createUserErr        error
	orgSettings          *model.OrgSettings
	orgSettingsErr       error
	effectiveRoles       []string
	effectiveRolesErr    error
	updateLastLoginErr   error
}

func (m *mockSAMLStore) GetSAMLProviderByID(_ context.Context, _ uuid.UUID) (*model.SAMLProvider, error) {
	return m.samlProvider, m.samlProviderErr
}
func (m *mockSAMLStore) CreateSAMLProvider(_ context.Context, _ *model.SAMLProvider) (*model.SAMLProvider, error) {
	return nil, nil
}
func (m *mockSAMLStore) ListSAMLProviders(_ context.Context, _ uuid.UUID) ([]*model.SAMLProvider, error) {
	return m.samlProviders, m.samlProvidersErr
}
func (m *mockSAMLStore) GetEnabledSAMLProviders(_ context.Context, _ uuid.UUID) ([]*model.SAMLProvider, error) {
	return m.samlProviders, m.samlProvidersErr
}
func (m *mockSAMLStore) UpdateSAMLProvider(_ context.Context, _ uuid.UUID, _ *model.UpdateSAMLProviderRequest) (*model.SAMLProvider, error) {
	return nil, nil
}
func (m *mockSAMLStore) DeleteSAMLProvider(_ context.Context, _ uuid.UUID) error { return nil }

// SAMLRequestStore
func (m *mockSAMLStore) StoreSAMLRequest(_ context.Context, _ string, _ uuid.UUID, _ time.Time) error {
	return m.storeSAMLReqErr
}
func (m *mockSAMLStore) ConsumeSAMLRequest(_ context.Context, _ string, _ uuid.UUID) (bool, error) {
	return m.consumeSAMLReq, m.consumeSAMLReqErr
}
func (m *mockSAMLStore) ConsumeOrRecordSAMLAssertion(_ context.Context, _ string, _ uuid.UUID, _ time.Time) (bool, error) {
	return m.assertionConsumed, m.assertionConsumedErr
}
func (m *mockSAMLStore) DeleteExpiredSAMLRequests(_ context.Context) (int64, error) {
	return 0, nil
}

// UserReader
func (m *mockSAMLStore) GetUserByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockSAMLStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return m.userByEmail, m.userByEmailErr
}
func (m *mockSAMLStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockSAMLStore) FindUserByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}

// UserWriter
func (m *mockSAMLStore) CreateUser(_ context.Context, u *model.User) (*model.User, error) {
	if m.createdUser != nil {
		return m.createdUser, m.createUserErr
	}
	u.ID = uuid.New()
	return u, m.createUserErr
}
func (m *mockSAMLStore) UpdateUser(_ context.Context, _, _ uuid.UUID, _ *model.UpdateUserRequest) (*model.User, error) {
	return nil, nil
}
func (m *mockSAMLStore) DeleteUser(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockSAMLStore) UpdatePassword(_ context.Context, _, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockSAMLStore) UpdateLastLoginAt(_ context.Context, _ uuid.UUID) error {
	return m.updateLastLoginErr
}
func (m *mockSAMLStore) IncrementFailedLogins(_ context.Context, _ uuid.UUID, _ int, _ time.Duration) error {
	return nil
}
func (m *mockSAMLStore) ResetFailedLogins(_ context.Context, _ uuid.UUID) error { return nil }

// OrgReader
func (m *mockSAMLStore) GetOrganizationByID(_ context.Context, _ uuid.UUID) (*model.Organization, error) {
	return nil, nil
}
func (m *mockSAMLStore) GetDefaultOrganizationID(_ context.Context) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockSAMLStore) GetOrganizationIDBySlug(_ context.Context, _ string) (uuid.UUID, error) {
	return m.orgIDBySlug, m.orgIDBySlugErr
}

// OrgSettingsReadWriter
func (m *mockSAMLStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.orgSettings, m.orgSettingsErr
}
func (m *mockSAMLStore) UpdateOrgSettings(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	return nil, nil
}

// GroupReader
func (m *mockSAMLStore) GetGroupByID(_ context.Context, _ uuid.UUID) (*model.Group, error) {
	return nil, nil
}
func (m *mockSAMLStore) GetGroupMembers(_ context.Context, _ uuid.UUID) ([]*model.GroupMember, error) {
	return nil, nil
}
func (m *mockSAMLStore) GetGroupRoles(_ context.Context, _ uuid.UUID) ([]*model.GroupRoleAssignment, error) {
	return nil, nil
}
func (m *mockSAMLStore) GetUserGroups(_ context.Context, _ uuid.UUID) ([]*model.Group, error) {
	return nil, nil
}
func (m *mockSAMLStore) GetEffectiveUserRoles(_ context.Context, _ uuid.UUID) ([]string, error) {
	return m.effectiveRoles, m.effectiveRolesErr
}
func (m *mockSAMLStore) CountGroupMembers(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockSAMLStore) CountGroupRoles(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockSAMLStore) CountGroups(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

// SocialAccountStore
func (m *mockSAMLStore) CreateSocialAccount(_ context.Context, _ *model.SocialAccount) (*model.SocialAccount, error) {
	return nil, nil
}
func (m *mockSAMLStore) GetSocialAccount(_ context.Context, _, _ string) (*model.SocialAccount, error) {
	return nil, nil
}
func (m *mockSAMLStore) GetSocialAccountsByUserID(_ context.Context, _ uuid.UUID) ([]*model.SocialAccount, error) {
	return nil, nil
}
func (m *mockSAMLStore) UpdateSocialAccountTokens(_ context.Context, _ uuid.UUID, _, _ string, _ *time.Time) error {
	return nil
}
func (m *mockSAMLStore) DeleteSocialAccount(_ context.Context, _ uuid.UUID) error { return nil }

func generateTestSelfSignedCert(key *rsa.PrivateKey) *x509.Certificate {
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		panic("failed to create test certificate: " + err.Error())
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		panic("failed to parse test certificate: " + err.Error())
	}
	return cert
}

func generateTestMFAToken(userID uuid.UUID) string {
	tok, err := token.GenerateMFAToken(testPrivKey, testKID, testIssuer, userID)
	if err != nil {
		panic("failed to generate MFA token: " + err.Error())
	}
	return tok
}

func newTestSAMLHandler(s *mockSAMLStore) *SAMLHandler {
	cert := generateTestSelfSignedCert(testPrivKey)
	return NewSAMLHandler(
		s, &mockSessionStore{}, noopLogger(), nil,
		testPrivKey, cert,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)
}

// ── SAML Metadata tests ──────────────────────────────────────────────

func TestMfaSamlSocial_SAMLMetadataInvalidProviderID(t *testing.T) {
	h := newTestSAMLHandler(&mockSAMLStore{})

	router := chi.NewRouter()
	router.Get("/saml/{providerID}/metadata", h.Metadata)

	req := httptest.NewRequest(http.MethodGet, "/saml/not-a-uuid/metadata", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMfaSamlSocial_SAMLMetadataProviderNotFound(t *testing.T) {
	h := newTestSAMLHandler(&mockSAMLStore{samlProvider: nil})

	router := chi.NewRouter()
	router.Get("/saml/{providerID}/metadata", h.Metadata)

	req := httptest.NewRequest(http.MethodGet, "/saml/"+uuid.New().String()+"/metadata", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestMfaSamlSocial_SAMLMetadataProviderDBError(t *testing.T) {
	h := newTestSAMLHandler(&mockSAMLStore{samlProviderErr: fmt.Errorf("db error")})

	router := chi.NewRouter()
	router.Get("/saml/{providerID}/metadata", h.Metadata)

	req := httptest.NewRequest(http.MethodGet, "/saml/"+uuid.New().String()+"/metadata", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestMfaSamlSocial_SAMLMetadataSuccess(t *testing.T) {
	providerID := uuid.New()
	orgID := uuid.New()

	// Generate a PEM-encoded certificate for the IdP
	idpCert := generateTestSelfSignedCert(testPrivKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: idpCert.Raw})

	s := &mockSAMLStore{
		samlProvider: &model.SAMLProvider{
			ID:          providerID,
			OrgID:       orgID,
			Name:        "Test IdP",
			EntityID:    "https://idp.example.com",
			SSOURL:      "https://idp.example.com/sso",
			Certificate: string(certPEM),
			Enabled:     true,
		},
	}
	h := newTestSAMLHandler(s)

	router := chi.NewRouter()
	router.Get("/saml/{providerID}/metadata", h.Metadata)

	req := httptest.NewRequest(http.MethodGet, "/saml/"+providerID.String()+"/metadata", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/xml" {
		t.Errorf("Content-Type = %q, want application/xml", ct)
	}
}

// ── SAML InitiateLogin tests ─────────────────────────────────────────

func TestMfaSamlSocial_SAMLInitiateLoginInvalidProviderID(t *testing.T) {
	h := newTestSAMLHandler(&mockSAMLStore{})

	router := chi.NewRouter()
	router.Get("/saml/{providerID}/login", h.InitiateLogin)

	req := httptest.NewRequest(http.MethodGet, "/saml/not-a-uuid/login", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMfaSamlSocial_SAMLInitiateLoginProviderNotFound(t *testing.T) {
	h := newTestSAMLHandler(&mockSAMLStore{samlProvider: nil})

	router := chi.NewRouter()
	router.Get("/saml/{providerID}/login", h.InitiateLogin)

	req := httptest.NewRequest(http.MethodGet, "/saml/"+uuid.New().String()+"/login", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestMfaSamlSocial_SAMLInitiateLoginProviderDisabled(t *testing.T) {
	providerID := uuid.New()
	s := &mockSAMLStore{
		samlProvider: &model.SAMLProvider{
			ID:      providerID,
			Enabled: false,
		},
	}
	h := newTestSAMLHandler(s)

	router := chi.NewRouter()
	router.Get("/saml/{providerID}/login", h.InitiateLogin)

	req := httptest.NewRequest(http.MethodGet, "/saml/"+providerID.String()+"/login", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestMfaSamlSocial_SAMLInitiateLoginSuccess(t *testing.T) {
	providerID := uuid.New()
	orgID := uuid.New()

	idpCert := generateTestSelfSignedCert(testPrivKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: idpCert.Raw})

	s := &mockSAMLStore{
		samlProvider: &model.SAMLProvider{
			ID:          providerID,
			OrgID:       orgID,
			Name:        "Test IdP",
			EntityID:    "https://idp.example.com",
			SSOURL:      "https://idp.example.com/sso",
			Certificate: string(certPEM),
			Enabled:     true,
		},
	}
	h := newTestSAMLHandler(s)

	router := chi.NewRouter()
	router.Get("/saml/{providerID}/login", h.InitiateLogin)

	req := httptest.NewRequest(http.MethodGet, "/saml/"+providerID.String()+"/login", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusFound, w.Body.String())
	}
	location := w.Header().Get("Location")
	if !strings.Contains(location, "idp.example.com") {
		t.Errorf("Location = %q, want redirect to IdP", location)
	}
}

// ── SAML ACS tests ───────────────────────────────────────────────────

func TestMfaSamlSocial_SAMLACSInvalidProviderID(t *testing.T) {
	h := newTestSAMLHandler(&mockSAMLStore{})

	router := chi.NewRouter()
	router.Post("/saml/{providerID}/acs", h.ACS)

	req := httptest.NewRequest(http.MethodPost, "/saml/not-a-uuid/acs", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMfaSamlSocial_SAMLACSProviderNotFound(t *testing.T) {
	h := newTestSAMLHandler(&mockSAMLStore{samlProvider: nil})

	router := chi.NewRouter()
	router.Post("/saml/{providerID}/acs", h.ACS)

	req := httptest.NewRequest(http.MethodPost, "/saml/"+uuid.New().String()+"/acs", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestMfaSamlSocial_SAMLACSProviderDisabled(t *testing.T) {
	providerID := uuid.New()
	s := &mockSAMLStore{
		samlProvider: &model.SAMLProvider{
			ID:      providerID,
			Enabled: false,
		},
	}
	h := newTestSAMLHandler(s)

	router := chi.NewRouter()
	router.Post("/saml/{providerID}/acs", h.ACS)

	req := httptest.NewRequest(http.MethodPost, "/saml/"+providerID.String()+"/acs", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ── SAML ListProviders tests ─────────────────────────────────────────

func TestMfaSamlSocial_SAMLListProvidersMissingOrg(t *testing.T) {
	h := newTestSAMLHandler(&mockSAMLStore{})

	req := httptest.NewRequest(http.MethodGet, "/saml/providers", http.NoBody)
	w := httptest.NewRecorder()

	h.ListProviders(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMfaSamlSocial_SAMLListProvidersOrgNotFound(t *testing.T) {
	s := &mockSAMLStore{orgIDBySlugErr: fmt.Errorf("not found")}
	h := newTestSAMLHandler(s)

	req := httptest.NewRequest(http.MethodGet, "/saml/providers?org=nonexistent", http.NoBody)
	w := httptest.NewRecorder()

	h.ListProviders(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestMfaSamlSocial_SAMLListProvidersDBError(t *testing.T) {
	orgID := uuid.New()
	s := &mockSAMLStore{
		orgIDBySlug:      orgID,
		samlProvidersErr: fmt.Errorf("db error"),
	}
	h := newTestSAMLHandler(s)

	req := httptest.NewRequest(http.MethodGet, "/saml/providers?org=myorg", http.NoBody)
	w := httptest.NewRecorder()

	h.ListProviders(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMfaSamlSocial_SAMLListProvidersSuccess(t *testing.T) {
	orgID := uuid.New()
	providerID := uuid.New()
	s := &mockSAMLStore{
		orgIDBySlug: orgID,
		samlProviders: []*model.SAMLProvider{
			{
				ID:      providerID,
				OrgID:   orgID,
				Name:    "Okta SSO",
				Enabled: true,
			},
		},
	}
	h := newTestSAMLHandler(s)

	req := httptest.NewRequest(http.MethodGet, "/saml/providers?org=myorg", http.NoBody)
	w := httptest.NewRecorder()

	h.ListProviders(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "Okta SSO") {
		t.Errorf("body = %q, want substring 'Okta SSO'", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "login_url") {
		t.Errorf("body = %q, want substring 'login_url'", w.Body.String())
	}
}

func TestMfaSamlSocial_SAMLListProvidersEmpty(t *testing.T) {
	orgID := uuid.New()
	s := &mockSAMLStore{
		orgIDBySlug:   orgID,
		samlProviders: []*model.SAMLProvider{},
	}
	h := newTestSAMLHandler(s)

	req := httptest.NewRequest(http.MethodGet, "/saml/providers?org=myorg", http.NoBody)
	w := httptest.NewRecorder()

	h.ListProviders(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "[]") {
		t.Errorf("body = %q, want empty array", w.Body.String())
	}
}

// ── SAML helper tests ────────────────────────────────────────────────

func TestMfaSamlSocial_ExtractInResponseToEmpty(t *testing.T) {
	result := extractInResponseTo("")
	if result != "" {
		t.Errorf("got %q, want empty string", result)
	}
}

func TestMfaSamlSocial_ExtractInResponseToInvalidBase64(t *testing.T) {
	result := extractInResponseTo("not-valid-base64!!!")
	if result != "" {
		t.Errorf("got %q, want empty string", result)
	}
}

func TestMfaSamlSocial_ExtractInResponseToInvalidXML(t *testing.T) {
	// Valid base64 but not valid XML
	result := extractInResponseTo("bm90IHhtbA==") // "not xml"
	if result != "" {
		t.Errorf("got %q, want empty string", result)
	}
}

func TestMfaSamlSocial_ParseCertFromKey(t *testing.T) {
	cert, err := ParseCertFromKey(testPrivKey)
	if err != nil {
		t.Fatalf("ParseCertFromKey failed: %v", err)
	}
	if cert == nil {
		t.Fatal("expected non-nil certificate")
	}
	if cert.PublicKey == nil {
		t.Error("expected certificate to have a public key")
	}
}

// ══════════════════════════════════════════════════════════════════════════
// No-panic tests: ensure handlers do not panic on any input
// ══════════════════════════════════════════════════════════════════════════

func TestMfaSamlSocial_NoPanicOnNilBody(t *testing.T) {
	// MFA handlers with nil body should not panic
	mfaStore := &mockMFAStore{}
	mfaH := newTestMFAHandler(mfaStore)

	t.Run("EnrollTOTP_nil_body", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/mfa/totp/enroll", http.NoBody)
		w := httptest.NewRecorder()
		mfaH.EnrollTOTP(w, r) // should not panic
	})

	t.Run("VerifyTOTPSetup_nil_body", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify-setup", http.NoBody)
		w := httptest.NewRecorder()
		mfaH.VerifyTOTPSetup(w, r) // should not panic
	})

	t.Run("DisableTOTP_nil_body", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/mfa/totp/disable", http.NoBody)
		w := httptest.NewRecorder()
		mfaH.DisableTOTP(w, r) // should not panic
	})

	// MFA verify handler with nil body
	mfaVerifyStore := &mockMFAVerifyStore{}
	mfaVerifyH := newTestMFAVerifyHandler(mfaVerifyStore)

	t.Run("VerifyTOTP_nil_body", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", http.NoBody)
		w := httptest.NewRecorder()
		mfaVerifyH.VerifyTOTP(w, r) // should not panic
	})

	// WebAuthn handlers with nil body
	waH := NewWebAuthnHandler(
		&mockWebAuthnStore{},
		&mockSessionStore{},
		noopLogger(), nil, nil,
		testPrivKey, &testPrivKey.PublicKey,
		testKID, testIssuer,
		15*time.Minute, 7*24*time.Hour,
	)

	t.Run("BeginLogin_nil_body", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/login/begin", http.NoBody)
		w := httptest.NewRecorder()
		waH.BeginLogin(w, r) // should not panic
	})

	t.Run("FinishLogin_nil_body", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/mfa/webauthn/login/complete", http.NoBody)
		w := httptest.NewRecorder()
		waH.FinishLogin(w, r) // should not panic
	})
}
