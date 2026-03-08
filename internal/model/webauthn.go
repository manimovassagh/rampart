package model

import (
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// WebAuthnCredential represents a stored WebAuthn/Passkey credential.
type WebAuthnCredential struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	CredentialID    []byte
	PublicKey       []byte
	AttestationType string
	Transport       []string
	FlagsRaw        uint8
	AAGUID          []byte
	SignCount       uint32
	Name            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ToLibCredential converts to the go-webauthn library Credential type.
func (c *WebAuthnCredential) ToLibCredential() webauthn.Credential {
	transports := make([]protocol.AuthenticatorTransport, len(c.Transport))
	for i, t := range c.Transport {
		transports[i] = protocol.AuthenticatorTransport(t)
	}

	return webauthn.Credential{
		ID:              c.CredentialID,
		PublicKey:       c.PublicKey,
		AttestationType: c.AttestationType,
		Transport:       transports,
		Flags:           webauthn.NewCredentialFlags(protocol.AuthenticatorFlags(c.FlagsRaw)),
		Authenticator: webauthn.Authenticator{
			AAGUID:    c.AAGUID,
			SignCount: c.SignCount,
		},
	}
}

// WebAuthnUser wraps a User to satisfy the webauthn.User interface.
type WebAuthnUser struct {
	User        *User
	Credentials []webauthn.Credential
}

// WebAuthnID returns the user ID as bytes.
func (u *WebAuthnUser) WebAuthnID() []byte {
	b, _ := u.User.ID.MarshalBinary()
	return b
}

// WebAuthnName returns the username.
func (u *WebAuthnUser) WebAuthnName() string {
	return u.User.Username
}

// WebAuthnDisplayName returns the display name.
func (u *WebAuthnUser) WebAuthnDisplayName() string {
	if u.User.GivenName != "" {
		name := u.User.GivenName
		if u.User.FamilyName != "" {
			name += " " + u.User.FamilyName
		}
		return name
	}
	return u.User.Username
}

// WebAuthnCredentials returns the user's credentials.
func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}
