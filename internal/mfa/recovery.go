package mfa

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// RecoveryCodeLength is the number of alphanumeric characters per half of a recovery code.
	RecoveryCodeLength = 4

	// BcryptCost is the bcrypt cost factor for hashing recovery codes.
	BcryptCost = 10
)

// recoveryAlphabet contains uppercase alphanumeric characters, excluding ambiguous ones.
const recoveryAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// GenerateRecoveryCodes generates the specified number of human-readable recovery codes
// in the format XXXX-XXXX (8 alphanumeric characters with a dash separator).
func GenerateRecoveryCodes(count int) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive, got %d", count)
	}

	codes := make([]string, 0, count)
	for i := 0; i < count; i++ {
		code, err := generateOneCode()
		if err != nil {
			return nil, fmt.Errorf("generating recovery code: %w", err)
		}
		codes = append(codes, code)
	}
	return codes, nil
}

// HashRecoveryCode returns a bcrypt hash of the given recovery code.
func HashRecoveryCode(code string) (string, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	hash, err := bcrypt.GenerateFromPassword([]byte(normalized), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("hashing recovery code: %w", err)
	}
	return string(hash), nil
}

// VerifyRecoveryCode checks whether a plaintext code matches a bcrypt hash.
func VerifyRecoveryCode(code, hash string) bool {
	normalized := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(normalized)) == nil
}

func generateOneCode() (string, error) {
	alphabetLen := big.NewInt(int64(len(recoveryAlphabet)))
	var sb strings.Builder
	sb.Grow(RecoveryCodeLength*2 + 1)

	for i := 0; i < RecoveryCodeLength*2; i++ {
		if i == RecoveryCodeLength {
			sb.WriteByte('-')
		}
		idx, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", fmt.Errorf("reading random bytes: %w", err)
		}
		sb.WriteByte(recoveryAlphabet[idx.Int64()])
	}
	return sb.String(), nil
}
