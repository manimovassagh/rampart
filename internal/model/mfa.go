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
