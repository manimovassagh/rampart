package auth

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
)

// FieldError represents a validation error on a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

const (
	minPasswordLen = 8
	maxPasswordLen = 128
	minUsernameLen = 3
	maxUsernameLen = 64
	maxEmailLen    = 254
	maxNameLen     = 255
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*[a-zA-Z0-9]$`)

// ValidateEmail checks that the email is well-formed and within length limits.
func ValidateEmail(email string) *FieldError {
	if strings.TrimSpace(email) == "" {
		return &FieldError{Field: "email", Message: "email is required"}
	}
	if len(email) > maxEmailLen {
		return &FieldError{Field: "email", Message: "email must be 254 characters or fewer"}
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return &FieldError{Field: "email", Message: "email is not a valid email address"}
	}
	return nil
}

// PasswordPolicy defines per-organization password requirements.
type PasswordPolicy struct {
	MinLength        int
	RequireUppercase bool
	RequireLowercase bool
	RequireNumbers   bool
	RequireSymbols   bool
}

// DefaultPasswordPolicy returns the built-in password policy.
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:        minPasswordLen,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumbers:   true,
		RequireSymbols:   true,
	}
}

// ValidatePassword enforces the default password policy.
func ValidatePassword(password string) *FieldError {
	return ValidatePasswordWithPolicy(password, DefaultPasswordPolicy())
}

// ValidatePasswordWithPolicy enforces a custom password policy.
func ValidatePasswordWithPolicy(password string, policy PasswordPolicy) *FieldError {
	if password == "" {
		return &FieldError{Field: "password", Message: "password is required"}
	}

	minLen := policy.MinLength
	if minLen < 1 {
		minLen = minPasswordLen
	}

	if len(password) < minLen {
		return &FieldError{
			Field:   "password",
			Message: fmt.Sprintf("password must be at least %d characters", minLen),
		}
	}
	if len(password) > maxPasswordLen {
		return &FieldError{Field: "password", Message: "password must be 128 characters or fewer"}
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	var missing []string
	if policy.RequireUppercase && !hasUpper {
		missing = append(missing, "one uppercase letter")
	}
	if policy.RequireLowercase && !hasLower {
		missing = append(missing, "one lowercase letter")
	}
	if policy.RequireNumbers && !hasDigit {
		missing = append(missing, "one digit")
	}
	if policy.RequireSymbols && !hasSpecial {
		missing = append(missing, "one special character")
	}

	if len(missing) > 0 {
		return &FieldError{
			Field:   "password",
			Message: "password must contain at least " + strings.Join(missing, ", "),
		}
	}
	return nil
}

// ValidateUsername checks username format: 3-64 chars, alphanumeric with dots/hyphens/underscores,
// must start and end with alphanumeric.
func ValidateUsername(username string) *FieldError {
	if strings.TrimSpace(username) == "" {
		return &FieldError{Field: "username", Message: "username is required"}
	}
	if len(username) < minUsernameLen {
		return &FieldError{Field: "username", Message: "username must be at least 3 characters"}
	}
	if len(username) > maxUsernameLen {
		return &FieldError{Field: "username", Message: "username must be 64 characters or fewer"}
	}
	if !usernameRegex.MatchString(username) {
		return &FieldError{
			Field:   "username",
			Message: "username must start and end with a letter or digit and can contain dots, hyphens, or underscores",
		}
	}
	return nil
}

// ValidateName checks that a name field (given_name or family_name) is within
// length limits and does not contain HTML-unsafe characters. Returns nil for
// empty strings because names are optional.
func ValidateName(field, name string) *FieldError {
	if name == "" {
		return nil
	}
	if len(name) > maxNameLen {
		return &FieldError{
			Field:   field,
			Message: fmt.Sprintf("%s must be %d characters or fewer", field, maxNameLen),
		}
	}
	if strings.ContainsAny(name, "<>&") {
		return &FieldError{
			Field:   field,
			Message: fmt.Sprintf("%s contains invalid characters", field),
		}
	}
	return nil
}

// ValidateRegistration validates all fields of a registration request.
// Returns a slice of field errors (empty if all valid).
func ValidateRegistration(email, password, username string) []FieldError {
	var errs []FieldError

	if fe := ValidateEmail(email); fe != nil {
		errs = append(errs, *fe)
	}
	if fe := ValidatePassword(password); fe != nil {
		errs = append(errs, *fe)
	}
	if fe := ValidateUsername(username); fe != nil {
		errs = append(errs, *fe)
	}

	return errs
}
