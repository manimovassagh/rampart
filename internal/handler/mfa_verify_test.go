package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/model"
	"github.com/manimovassagh/rampart/internal/token"
)

// mockMFAVerifyStore implements MFAVerifyStore for testing.
type mockMFAVerifyStore struct {
	userByID           *model.User
	userByIDErr        error
	verifiedDevice     *model.MFADevice
	verifiedDeviceErr  error
	consumeBackupCode  bool
	consumeBackupErr   error
	resetFailedErr     error
	updateLastLoginErr error
	orgSettings        *model.OrgSettings
	orgSettingsErr     error
	effectiveRoles     []string
	effectiveRolesErr  error
}

func (m *mockMFAVerifyStore) GetUserByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return m.userByID, m.userByIDErr
}
func (m *mockMFAVerifyStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockMFAVerifyStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockMFAVerifyStore) FindUserByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}

// UserWriter
func (m *mockMFAVerifyStore) CreateUser(_ context.Context, _ *model.User) (*model.User, error) {
	return nil, nil
}
func (m *mockMFAVerifyStore) UpdateUser(_ context.Context, _, _ uuid.UUID, _ *model.UpdateUserRequest) (*model.User, error) {
	return nil, nil
}
func (m *mockMFAVerifyStore) DeleteUser(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockMFAVerifyStore) UpdatePassword(_ context.Context, _, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockMFAVerifyStore) UpdateLastLoginAt(_ context.Context, _ uuid.UUID) error {
	return m.updateLastLoginErr
}
func (m *mockMFAVerifyStore) IncrementFailedLogins(_ context.Context, _ uuid.UUID, _ int, _ time.Duration) error {
	return nil
}
func (m *mockMFAVerifyStore) ResetFailedLogins(_ context.Context, _ uuid.UUID) error {
	return m.resetFailedErr
}

// MFADeviceStore
func (m *mockMFAVerifyStore) GetVerifiedMFADevice(_ context.Context, _ uuid.UUID) (*model.MFADevice, error) {
	return m.verifiedDevice, m.verifiedDeviceErr
}
func (m *mockMFAVerifyStore) GetPendingMFADevice(_ context.Context, _ uuid.UUID) (*model.MFADevice, error) {
	return nil, nil
}
func (m *mockMFAVerifyStore) CreateMFADevice(_ context.Context, _ uuid.UUID, _, _, _ string) (*model.MFADevice, error) {
	return nil, nil
}
func (m *mockMFAVerifyStore) VerifyMFADevice(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockMFAVerifyStore) DeleteUnverifiedMFADevices(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockMFAVerifyStore) DisableMFA(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockMFAVerifyStore) StoreBackupCodes(_ context.Context, _ uuid.UUID, _ [][]byte) error {
	return nil
}
func (m *mockMFAVerifyStore) ConsumeBackupCode(_ context.Context, _ uuid.UUID, _ []byte) (bool, error) {
	return m.consumeBackupCode, m.consumeBackupErr
}

// OrgSettingsReadWriter
func (m *mockMFAVerifyStore) GetOrgSettings(_ context.Context, _ uuid.UUID) (*model.OrgSettings, error) {
	return m.orgSettings, m.orgSettingsErr
}
func (m *mockMFAVerifyStore) UpdateOrgSettings(_ context.Context, _ uuid.UUID, _ *model.UpdateOrgSettingsRequest) (*model.OrgSettings, error) {
	return nil, nil
}

// GroupReader
func (m *mockMFAVerifyStore) GetGroupByID(_ context.Context, _ uuid.UUID) (*model.Group, error) {
	return nil, nil
}
func (m *mockMFAVerifyStore) GetGroupMembers(_ context.Context, _ uuid.UUID) ([]*model.GroupMember, error) {
	return nil, nil
}
func (m *mockMFAVerifyStore) GetGroupRoles(_ context.Context, _ uuid.UUID) ([]*model.GroupRoleAssignment, error) {
	return nil, nil
}
func (m *mockMFAVerifyStore) GetUserGroups(_ context.Context, _ uuid.UUID) ([]*model.Group, error) {
	return nil, nil
}
func (m *mockMFAVerifyStore) GetEffectiveUserRoles(_ context.Context, _ uuid.UUID) ([]string, error) {
	return m.effectiveRoles, m.effectiveRolesErr
}
func (m *mockMFAVerifyStore) CountGroupMembers(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockMFAVerifyStore) CountGroupRoles(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockMFAVerifyStore) CountGroups(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func newTestMFAVerifyHandler(s *mockMFAVerifyStore) *MFAVerifyHandler {
	return NewMFAVerifyHandler(
		s,
		&mockSessionStore{},
		noopLogger(),
		nil, // audit logger (nil-safe)
		testPrivKey,
		&testPrivKey.PublicKey,
		testKID,
		testIssuer,
		15*time.Minute,
		7*24*time.Hour,
	)
}

func generateMFAToken(userID uuid.UUID) string {
	tok, err := token.GenerateMFAToken(testPrivKey, testKID, testIssuer, userID)
	if err != nil {
		panic("failed to generate MFA token: " + err.Error())
	}
	return tok
}

// ── VerifyTOTP tests ─────────────────────────────────────────────────────

func TestMFAVerifyTOTPSuccess(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	secret := testTOTPSecret
	code := validTOTPCode()
	mfaToken := generateMFAToken(userID)

	s := &mockMFAVerifyStore{
		userByID: &model.User{
			ID:            userID,
			OrgID:         orgID,
			Username:      "testuser",
			Email:         "test@rampart.local",
			EmailVerified: true,
			Enabled:       true,
			GivenName:     "Test",
			FamilyName:    "User",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		verifiedDevice: &model.MFADevice{
			ID:       uuid.New(),
			UserID:   userID,
			Secret:   secret,
			Verified: true,
		},
	}

	h := newTestMFAVerifyHandler(s)

	body := fmt.Appendf(nil, `{"mfa_token": "%s", "code": "%s"}`, mfaToken, code)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyTOTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp LoginResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh_token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("token_type = %q, want Bearer", resp.TokenType)
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("expires_in = %d, want 900", resp.ExpiresIn)
	}
}

func TestMFAVerifyTOTPMissingFields(t *testing.T) {
	s := &mockMFAVerifyStore{}
	h := newTestMFAVerifyHandler(s)

	cases := []struct {
		name string
		body string
	}{
		{"missing_both", `{}`},
		{"missing_code", `{"mfa_token": "some-token"}`},
		{"missing_mfa_token", `{"code": "123456"}`},
		{"empty_code", `{"mfa_token": "some-token", "code": ""}`},
		{"empty_mfa_token", `{"mfa_token": "", "code": "123456"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader([]byte(tc.body)))
			w := httptest.NewRecorder()

			h.VerifyTOTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestMFAVerifyTOTPInvalidMFAToken(t *testing.T) {
	s := &mockMFAVerifyStore{}
	h := newTestMFAVerifyHandler(s)

	body := []byte(`{"mfa_token": "invalid-token", "code": "123456"}`)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyTOTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMFAVerifyTOTPUserNotFound(t *testing.T) {
	userID := uuid.New()
	mfaToken := generateMFAToken(userID)

	s := &mockMFAVerifyStore{
		userByID: nil,
	}
	h := newTestMFAVerifyHandler(s)

	body := fmt.Appendf(nil, `{"mfa_token": "%s", "code": "123456"}`, mfaToken)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyTOTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMFAVerifyTOTPUserDisabled(t *testing.T) {
	userID := uuid.New()
	mfaToken := generateMFAToken(userID)

	s := &mockMFAVerifyStore{
		userByID: &model.User{
			ID:      userID,
			Enabled: false,
		},
	}
	h := newTestMFAVerifyHandler(s)

	body := fmt.Appendf(nil, `{"mfa_token": "%s", "code": "123456"}`, mfaToken)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyTOTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMFAVerifyTOTPGetUserError(t *testing.T) {
	userID := uuid.New()
	mfaToken := generateMFAToken(userID)

	s := &mockMFAVerifyStore{
		userByIDErr: fmt.Errorf("db error"),
	}
	h := newTestMFAVerifyHandler(s)

	body := fmt.Appendf(nil, `{"mfa_token": "%s", "code": "123456"}`, mfaToken)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyTOTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMFAVerifyTOTPInvalidCode(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	mfaToken := generateMFAToken(userID)

	s := &mockMFAVerifyStore{
		userByID: &model.User{
			ID:        userID,
			OrgID:     orgID,
			Username:  "testuser",
			Email:     "test@rampart.local",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		verifiedDevice: &model.MFADevice{
			ID:       uuid.New(),
			UserID:   userID,
			Secret:   testTOTPSecret,
			Verified: true,
		},
		consumeBackupCode: false,
	}
	h := newTestMFAVerifyHandler(s)

	body := fmt.Appendf(nil, `{"mfa_token": "%s", "code": "000000"}`, mfaToken)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyTOTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMFAVerifyTOTPNoDeviceConfigured(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	mfaToken := generateMFAToken(userID)

	s := &mockMFAVerifyStore{
		userByID: &model.User{
			ID:        userID,
			OrgID:     orgID,
			Username:  "testuser",
			Email:     "test@rampart.local",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		verifiedDevice: nil,
	}
	h := newTestMFAVerifyHandler(s)

	body := fmt.Appendf(nil, `{"mfa_token": "%s", "code": "123456"}`, mfaToken)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyTOTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMFAVerifyTOTPGetDeviceError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	mfaToken := generateMFAToken(userID)

	s := &mockMFAVerifyStore{
		userByID: &model.User{
			ID:        userID,
			OrgID:     orgID,
			Username:  "testuser",
			Email:     "test@rampart.local",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		verifiedDeviceErr: fmt.Errorf("db error"),
	}
	h := newTestMFAVerifyHandler(s)

	body := fmt.Appendf(nil, `{"mfa_token": "%s", "code": "123456"}`, mfaToken)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyTOTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMFAVerifyTOTPWithBackupCode(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	mfaToken := generateMFAToken(userID)

	s := &mockMFAVerifyStore{
		userByID: &model.User{
			ID:            userID,
			OrgID:         orgID,
			Username:      "testuser",
			Email:         "test@rampart.local",
			EmailVerified: true,
			Enabled:       true,
			GivenName:     "Test",
			FamilyName:    "User",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		verifiedDevice: &model.MFADevice{
			ID:       uuid.New(),
			UserID:   userID,
			Secret:   testTOTPSecret,
			Verified: true,
		},
		consumeBackupCode: true, // backup code is valid
	}
	h := newTestMFAVerifyHandler(s)

	// Use a wrong TOTP code that will fail validation, but backup code will succeed
	body := fmt.Appendf(nil, `{"mfa_token": "%s", "code": "abcd-efgh"}`, mfaToken)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyTOTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp LoginResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
}

func TestMFAVerifyTOTPBackupCodeCheckError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	mfaToken := generateMFAToken(userID)

	s := &mockMFAVerifyStore{
		userByID: &model.User{
			ID:        userID,
			OrgID:     orgID,
			Username:  "testuser",
			Email:     "test@rampart.local",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		verifiedDevice: &model.MFADevice{
			ID:       uuid.New(),
			UserID:   userID,
			Secret:   testTOTPSecret,
			Verified: true,
		},
		consumeBackupErr: fmt.Errorf("db error"),
	}
	h := newTestMFAVerifyHandler(s)

	// Use wrong TOTP code so it falls through to backup code path
	body := fmt.Appendf(nil, `{"mfa_token": "%s", "code": "abcd-efgh"}`, mfaToken)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyTOTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMFAVerifyTOTPInvalidJSON(t *testing.T) {
	s := &mockMFAVerifyStore{}
	h := newTestMFAVerifyHandler(s)

	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	h.VerifyTOTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMFAVerifyTOTPWithOrgSettings(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	secret := testTOTPSecret
	code := validTOTPCode()
	mfaToken := generateMFAToken(userID)

	s := &mockMFAVerifyStore{
		userByID: &model.User{
			ID:            userID,
			OrgID:         orgID,
			Username:      "testuser",
			Email:         "test@rampart.local",
			EmailVerified: true,
			Enabled:       true,
			GivenName:     "Test",
			FamilyName:    "User",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		verifiedDevice: &model.MFADevice{
			ID:       uuid.New(),
			UserID:   userID,
			Secret:   secret,
			Verified: true,
		},
		orgSettings: &model.OrgSettings{
			AccessTokenTTL:  30 * time.Minute,
			RefreshTokenTTL: 14 * 24 * time.Hour,
		},
	}
	h := newTestMFAVerifyHandler(s)

	body := fmt.Appendf(nil, `{"mfa_token": "%s", "code": "%s"}`, mfaToken, code)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyTOTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp LoginResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	// With 30 min org setting, expires_in should be 1800
	if resp.ExpiresIn != 1800 {
		t.Errorf("expires_in = %d, want 1800", resp.ExpiresIn)
	}
}
