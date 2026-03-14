package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // SHA-1 required by TOTP RFC 6238
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
	"github.com/manimovassagh/rampart/internal/model"
)

// mockMFAStore implements MFAHandlerStore for testing.
type mockMFAStore struct {
	userByID            *model.User
	userByIDErr         error
	verifiedDevice      *model.MFADevice
	verifiedDeviceErr   error
	pendingDevice       *model.MFADevice
	pendingDeviceErr    error
	createDevice        *model.MFADevice
	createDeviceErr     error
	verifyDeviceErr     error
	disableMFAErr       error
	storeBackupCodesErr error
}

func (m *mockMFAStore) GetUserByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return m.userByID, m.userByIDErr
}
func (m *mockMFAStore) GetUserByEmail(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockMFAStore) GetUserByUsername(_ context.Context, _ string, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (m *mockMFAStore) FindUserByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}
func (m *mockMFAStore) GetVerifiedMFADevice(_ context.Context, _ uuid.UUID) (*model.MFADevice, error) {
	return m.verifiedDevice, m.verifiedDeviceErr
}
func (m *mockMFAStore) GetPendingMFADevice(_ context.Context, _ uuid.UUID) (*model.MFADevice, error) {
	return m.pendingDevice, m.pendingDeviceErr
}
func (m *mockMFAStore) CreateMFADevice(_ context.Context, _ uuid.UUID, _, _, _ string) (*model.MFADevice, error) {
	return m.createDevice, m.createDeviceErr
}
func (m *mockMFAStore) VerifyMFADevice(_ context.Context, _, _ uuid.UUID) error {
	return m.verifyDeviceErr
}
func (m *mockMFAStore) DeleteUnverifiedMFADevices(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockMFAStore) DisableMFA(_ context.Context, _ uuid.UUID) error {
	return m.disableMFAErr
}
func (m *mockMFAStore) StoreBackupCodes(_ context.Context, _ uuid.UUID, _ [][]byte) error {
	return m.storeBackupCodesErr
}
func (m *mockMFAStore) ConsumeBackupCode(_ context.Context, _ uuid.UUID, _ []byte) (bool, error) {
	return false, nil
}

const testTOTPSecret = "JBSWY3DPEHPK3PXP"

// validTOTPCode generates a valid TOTP code for the test secret at the current time.
func validTOTPCode() string {
	secretBytes, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(testTOTPSecret))
	counter := time.Now().Unix() / 30

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter)) //nolint:gosec // counter is always non-negative

	mac := hmac.New(sha1.New, secretBytes) //nolint:gosec // SHA-1 required by RFC 6238
	mac.Write(buf)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	otp := truncated % uint32(math.Pow10(6))

	return fmt.Sprintf("%06d", otp)
}

func newAuthenticatedRequest(target string, body []byte, userID, orgID uuid.UUID) *http.Request {
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

func newTestMFAHandler(s *mockMFAStore) *MFAHandler {
	return NewMFAHandler(s, noopLogger(), nil, testIssuer)
}

// ── EnrollTOTP tests ─────────────────────────────────────────────────────

func TestMFAEnrollTOTPSuccess(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	deviceID := uuid.New()

	s := &mockMFAStore{
		verifiedDevice: nil,
		userByID: &model.User{
			ID:       userID,
			OrgID:    orgID,
			Email:    "test@rampart.local",
			Username: "testuser",
			Enabled:  true,
		},
		createDevice: &model.MFADevice{
			ID:         deviceID,
			UserID:     userID,
			DeviceType: "totp",
			Name:       "Authenticator",
			Secret:     testTOTPSecret,
		},
	}

	r := newAuthenticatedRequest("/mfa/totp/enroll", nil, userID, orgID)
	w := httptest.NewRecorder()

	h := newTestMFAHandler(s)
	h.EnrollTOTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp model.TOTPEnrollResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Secret == "" {
		t.Error("expected non-empty secret")
	}
	if resp.ProvisioningURI == "" {
		t.Error("expected non-empty provisioning_uri")
	}
	if resp.DeviceID != deviceID {
		t.Errorf("device_id = %q, want %q", resp.DeviceID, deviceID)
	}
}

func TestMFAEnrollTOTPUnauthenticated(t *testing.T) {
	s := &mockMFAStore{}
	h := newTestMFAHandler(s)

	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/enroll", http.NoBody)
	w := httptest.NewRecorder()

	h.EnrollTOTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMFAEnrollTOTPAlreadyEnabled(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{
		verifiedDevice: &model.MFADevice{
			ID:       uuid.New(),
			UserID:   userID,
			Verified: true,
		},
	}

	r := newAuthenticatedRequest("/mfa/totp/enroll", nil, userID, orgID)
	w := httptest.NewRecorder()

	h := newTestMFAHandler(s)
	h.EnrollTOTP(w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestMFAEnrollTOTPGetVerifiedDeviceError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{
		verifiedDeviceErr: fmt.Errorf("db error"),
	}

	r := newAuthenticatedRequest("/mfa/totp/enroll", nil, userID, orgID)
	w := httptest.NewRecorder()

	h := newTestMFAHandler(s)
	h.EnrollTOTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMFAEnrollTOTPCreateDeviceError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{
		verifiedDevice: nil,
		userByID: &model.User{
			ID:       userID,
			OrgID:    orgID,
			Email:    "test@rampart.local",
			Username: "testuser",
			Enabled:  true,
		},
		createDeviceErr: fmt.Errorf("db error"),
	}

	r := newAuthenticatedRequest("/mfa/totp/enroll", nil, userID, orgID)
	w := httptest.NewRecorder()

	h := newTestMFAHandler(s)
	h.EnrollTOTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMFAEnrollTOTPGetUserError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{
		verifiedDevice: nil,
		userByIDErr:    fmt.Errorf("db error"),
	}

	r := newAuthenticatedRequest("/mfa/totp/enroll", nil, userID, orgID)
	w := httptest.NewRecorder()

	h := newTestMFAHandler(s)
	h.EnrollTOTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ── VerifyTOTPSetup tests ────────────────────────────────────────────────

func TestMFAVerifyTOTPSetupSuccess(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	deviceID := uuid.New()
	// Use a known secret for generating a valid code
	secret := testTOTPSecret
	code := validTOTPCode()

	s := &mockMFAStore{
		pendingDevice: &model.MFADevice{
			ID:       deviceID,
			UserID:   userID,
			Secret:   secret,
			Verified: false,
		},
	}

	body := fmt.Appendf(nil, `{"code": "%s"}`, code)
	r := newAuthenticatedRequest("/mfa/totp/verify-setup", body, userID, orgID)
	w := httptest.NewRecorder()

	h := newTestMFAHandler(s)
	h.VerifyTOTPSetup(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp model.TOTPVerifySetupResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
	if len(resp.BackupCodes) == 0 {
		t.Error("expected backup codes in response")
	}
}

func TestMFAVerifyTOTPSetupUnauthenticated(t *testing.T) {
	s := &mockMFAStore{}
	h := newTestMFAHandler(s)

	body := []byte(`{"code": "123456"}`)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/verify-setup", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyTOTPSetup(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMFAVerifyTOTPSetupMissingCode(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{}
	h := newTestMFAHandler(s)

	body := []byte(`{"code": ""}`)
	r := newAuthenticatedRequest("/mfa/totp/verify-setup", body, userID, orgID)
	w := httptest.NewRecorder()

	h.VerifyTOTPSetup(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMFAVerifyTOTPSetupNoPendingEnrollment(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{
		pendingDevice: nil,
	}
	h := newTestMFAHandler(s)

	body := []byte(`{"code": "123456"}`)
	r := newAuthenticatedRequest("/mfa/totp/verify-setup", body, userID, orgID)
	w := httptest.NewRecorder()

	h.VerifyTOTPSetup(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMFAVerifyTOTPSetupInvalidCode(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{
		pendingDevice: &model.MFADevice{
			ID:       uuid.New(),
			UserID:   userID,
			Secret:   testTOTPSecret,
			Verified: false,
		},
	}
	h := newTestMFAHandler(s)

	body := []byte(`{"code": "000000"}`)
	r := newAuthenticatedRequest("/mfa/totp/verify-setup", body, userID, orgID)
	w := httptest.NewRecorder()

	h.VerifyTOTPSetup(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMFAVerifyTOTPSetupGetPendingDeviceError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{
		pendingDeviceErr: fmt.Errorf("db error"),
	}
	h := newTestMFAHandler(s)

	body := []byte(`{"code": "123456"}`)
	r := newAuthenticatedRequest("/mfa/totp/verify-setup", body, userID, orgID)
	w := httptest.NewRecorder()

	h.VerifyTOTPSetup(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMFAVerifyTOTPSetupVerifyDeviceError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	secret := testTOTPSecret
	code := validTOTPCode()

	s := &mockMFAStore{
		pendingDevice: &model.MFADevice{
			ID:       uuid.New(),
			UserID:   userID,
			Secret:   secret,
			Verified: false,
		},
		verifyDeviceErr: fmt.Errorf("db error"),
	}
	h := newTestMFAHandler(s)

	body := fmt.Appendf(nil, `{"code": "%s"}`, code)
	r := newAuthenticatedRequest("/mfa/totp/verify-setup", body, userID, orgID)
	w := httptest.NewRecorder()

	h.VerifyTOTPSetup(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMFAVerifyTOTPSetupInvalidJSON(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{}
	h := newTestMFAHandler(s)

	body := []byte(`not json`)
	r := newAuthenticatedRequest("/mfa/totp/verify-setup", body, userID, orgID)
	w := httptest.NewRecorder()

	h.VerifyTOTPSetup(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ── DisableTOTP tests ────────────────────────────────────────────────────

func TestMFADisableTOTPSuccess(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	secret := testTOTPSecret
	code := validTOTPCode()

	s := &mockMFAStore{
		verifiedDevice: &model.MFADevice{
			ID:       uuid.New(),
			UserID:   userID,
			Secret:   secret,
			Verified: true,
		},
	}
	h := newTestMFAHandler(s)

	body := fmt.Appendf(nil, `{"code": "%s"}`, code)
	r := newAuthenticatedRequest("/mfa/totp/disable", body, userID, orgID)
	w := httptest.NewRecorder()

	h.DisableTOTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp model.TOTPDisableResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestMFADisableTOTPUnauthenticated(t *testing.T) {
	s := &mockMFAStore{}
	h := newTestMFAHandler(s)

	body := []byte(`{"code": "123456"}`)
	r := httptest.NewRequest(http.MethodPost, "/mfa/totp/disable", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.DisableTOTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMFADisableTOTPMissingCode(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{}
	h := newTestMFAHandler(s)

	body := []byte(`{"code": ""}`)
	r := newAuthenticatedRequest("/mfa/totp/disable", body, userID, orgID)
	w := httptest.NewRecorder()

	h.DisableTOTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMFADisableTOTPNotEnabled(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{
		verifiedDevice: nil,
	}
	h := newTestMFAHandler(s)

	body := []byte(`{"code": "123456"}`)
	r := newAuthenticatedRequest("/mfa/totp/disable", body, userID, orgID)
	w := httptest.NewRecorder()

	h.DisableTOTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMFADisableTOTPInvalidCode(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{
		verifiedDevice: &model.MFADevice{
			ID:       uuid.New(),
			UserID:   userID,
			Secret:   testTOTPSecret,
			Verified: true,
		},
	}
	h := newTestMFAHandler(s)

	body := []byte(`{"code": "000000"}`)
	r := newAuthenticatedRequest("/mfa/totp/disable", body, userID, orgID)
	w := httptest.NewRecorder()

	h.DisableTOTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMFADisableTOTPGetDeviceError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{
		verifiedDeviceErr: fmt.Errorf("db error"),
	}
	h := newTestMFAHandler(s)

	body := []byte(`{"code": "123456"}`)
	r := newAuthenticatedRequest("/mfa/totp/disable", body, userID, orgID)
	w := httptest.NewRecorder()

	h.DisableTOTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMFADisableTOTPDisableMFAError(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	secret := testTOTPSecret
	code := validTOTPCode()

	s := &mockMFAStore{
		verifiedDevice: &model.MFADevice{
			ID:       uuid.New(),
			UserID:   userID,
			Secret:   secret,
			Verified: true,
		},
		disableMFAErr: fmt.Errorf("db error"),
	}
	h := newTestMFAHandler(s)

	body := fmt.Appendf(nil, `{"code": "%s"}`, code)
	r := newAuthenticatedRequest("/mfa/totp/disable", body, userID, orgID)
	w := httptest.NewRecorder()

	h.DisableTOTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMFADisableTOTPInvalidJSON(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	s := &mockMFAStore{}
	h := newTestMFAHandler(s)

	body := []byte(`not json`)
	r := newAuthenticatedRequest("/mfa/totp/disable", body, userID, orgID)
	w := httptest.NewRecorder()

	h.DisableTOTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
