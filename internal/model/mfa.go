package model

import (
	"time"

	"github.com/google/uuid"
)

// MFADevice represents a registered MFA device (e.g. TOTP authenticator).
type MFADevice struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DeviceType string
	Name       string
	Secret     string
	Verified   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TOTPEnrollResponse is returned when initiating TOTP enrollment.
type TOTPEnrollResponse struct {
	Secret          string    `json:"secret"`
	ProvisioningURI string    `json:"provisioning_uri"`
	DeviceID        uuid.UUID `json:"device_id"`
}

// TOTPVerifySetupResponse is returned when completing TOTP setup.
type TOTPVerifySetupResponse struct {
	Message     string   `json:"message"`
	BackupCodes []string `json:"backup_codes"`
}

// TOTPDisableResponse is returned when disabling TOTP.
type TOTPDisableResponse struct {
	Message string `json:"message"`
}
