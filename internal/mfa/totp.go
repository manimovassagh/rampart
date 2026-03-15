// Package mfa implements TOTP-based multi-factor authentication per RFC 6238.
package mfa

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // SHA-1 is required by TOTP RFC 6238
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

const (
	// SecretSize is the number of random bytes for the TOTP secret.
	SecretSize = 20
	// Digits is the number of digits in a TOTP code.
	Digits = 6
	// Period is the time step in seconds.
	Period = 30
	// Skew is the number of periods to check before/after current time.
	Skew = 1
	// BackupCodeCount is the number of backup codes to generate.
	BackupCodeCount = 10
	// BackupCodeLength is the number of characters in a backup code.
	BackupCodeLength = 8
)

// GenerateSecret generates a random TOTP secret and returns it as a base32 string.
func GenerateSecret() (string, error) {
	secret := make([]byte, SecretSize)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generating TOTP secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// ProvisioningURI builds an otpauth:// URI for QR code generation.
func ProvisioningURI(secret, email, issuer string) string {
	params := url.Values{}
	params.Set("secret", secret)
	params.Set("issuer", issuer)
	params.Set("algorithm", "SHA1")
	params.Set("digits", fmt.Sprintf("%d", Digits))
	params.Set("period", fmt.Sprintf("%d", Period))

	label := url.PathEscape(fmt.Sprintf("%s:%s", issuer, email))
	return fmt.Sprintf("otpauth://totp/%s?%s", label, params.Encode())
}

// ValidateCode checks a TOTP code against the secret, allowing for clock skew.
// It accepts the last used time step to prevent replay attacks within the same
// TOTP window. On success it returns the matched time step; on failure it returns 0.
func ValidateCode(secret, code string, lastUsedAt int64) (ok bool, timeStep int64) {
	if len(code) != Digits {
		return false, 0
	}

	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false, 0
	}

	now := time.Now().Unix()
	counter := now / Period

	for i := -int64(Skew); i <= int64(Skew); i++ {
		step := counter + i
		expected := generateTOTP(secretBytes, step)
		if hmac.Equal([]byte(code), []byte(expected)) {
			if step <= lastUsedAt {
				return false, 0
			}
			return true, step
		}
	}
	return false, 0
}

func generateTOTP(secret []byte, counter int64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter)) //nolint:gosec // counter is always non-negative (unix timestamp / period)

	mac := hmac.New(sha1.New, secret) //nolint:gosec // SHA-1 required by RFC 6238
	mac.Write(buf)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	otp := truncated % uint32(math.Pow10(Digits))

	return fmt.Sprintf("%0*d", Digits, otp)
}

// GenerateBackupCodes generates a set of single-use backup codes.
func GenerateBackupCodes() ([]string, error) {
	codes := make([]string, BackupCodeCount)
	for i := range codes {
		b := make([]byte, BackupCodeLength)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("generating backup code: %w", err)
		}
		// Use hex-like alphanumeric codes (easy to type)
		const charset = "abcdefghjkmnpqrstuvwxyz23456789" // no ambiguous chars
		for j := range b {
			b[j] = charset[b[j]%byte(len(charset))]
		}
		codes[i] = string(b[:4]) + "-" + string(b[4:])
	}
	return codes, nil
}

// HashBackupCode returns the SHA-256 hash of a backup code.
func HashBackupCode(code string) []byte {
	normalized := strings.ToLower(strings.ReplaceAll(code, "-", ""))
	h := sha256.Sum256([]byte(normalized))
	return h[:]
}
