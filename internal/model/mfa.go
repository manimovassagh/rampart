package model

import (
	"time"

	"github.com/google/uuid"
)

// TOTPDevice represents a TOTP authenticator device registered to a user.
type TOTPDevice struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Secret     string     `json:"-"`
	Name       string     `json:"name"`
	Verified   bool       `json:"verified"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// RecoveryCode represents a one-time-use recovery code for MFA bypass.
type RecoveryCode struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	CodeHash  string     `json:"-"`
	Used      bool       `json:"used"`
	CreatedAt time.Time  `json:"created_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}
