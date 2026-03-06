package mfa

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //#nosec G505 -- SHA1 is required by RFC 6238 TOTP
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

const (
	// SecretSize is the number of random bytes used to generate a TOTP secret.
	SecretSize = 20

	// CodeDigits is the number of digits in a TOTP code.
	CodeDigits = 6

	// Period is the time step in seconds (RFC 6238 default).
	Period = 30

	// Tolerance is the number of time steps to check before/after current.
	Tolerance = 1
)

// GenerateSecret generates a cryptographically random TOTP secret
// and returns it as a base32-encoded string (no padding).
func GenerateSecret() (string, error) {
	secret := make([]byte, SecretSize)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generating TOTP secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GenerateQRCodeURI returns an otpauth:// URI suitable for QR code rendering.
// Format: otpauth://totp/{issuer}:{accountName}?secret={secret}&issuer={issuer}&algorithm=SHA1&digits=6&period=30
func GenerateQRCodeURI(secret, issuer, accountName string) string {
	label := url.PathEscape(issuer) + ":" + url.PathEscape(accountName)
	params := url.Values{}
	params.Set("secret", secret)
	params.Set("issuer", issuer)
	params.Set("algorithm", "SHA1")
	params.Set("digits", fmt.Sprintf("%d", CodeDigits))
	params.Set("period", fmt.Sprintf("%d", Period))
	return fmt.Sprintf("otpauth://totp/%s?%s", label, params.Encode())
}

// ValidateCode checks whether the given 6-digit code is valid for the secret,
// allowing for +/- Tolerance time steps around the current time.
func ValidateCode(secret string, code string) bool {
	return ValidateCodeAt(secret, code, time.Now())
}

// ValidateCodeAt checks a TOTP code against a specific point in time.
func ValidateCodeAt(secret string, code string, t time.Time) bool {
	if len(code) != CodeDigits {
		return false
	}

	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(
		strings.ToUpper(secret),
	)
	if err != nil {
		return false
	}

	counter := t.Unix() / Period

	for i := -int64(Tolerance); i <= int64(Tolerance); i++ {
		expected := generateCode(secretBytes, counter+i)
		if hmac.Equal([]byte(code), []byte(expected)) {
			return true
		}
	}

	return false
}

// generateCode computes a TOTP code for the given secret and counter value
// following RFC 6238 / RFC 4226 (HOTP).
func generateCode(secret []byte, counter int64) string {
	// Step 1: Convert counter to big-endian 8-byte buffer.
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	// Step 2: Compute HMAC-SHA1.
	mac := hmac.New(sha1.New, secret) //#nosec G401 -- SHA1 is required by RFC 6238
	mac.Write(buf)
	sum := mac.Sum(nil)

	// Step 3: Dynamic truncation (RFC 4226 section 5.4).
	offset := sum[len(sum)-1] & 0x0f
	binCode := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		uint32(sum[offset+3])&0xff

	// Step 4: Compute TOTP value = binCode mod 10^digits.
	otp := binCode % uint32(math.Pow10(CodeDigits))

	return fmt.Sprintf("%0*d", CodeDigits, otp)
}
