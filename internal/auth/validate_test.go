package auth

import (
	"strings"
	"testing"
)

func TestValidateEmailValid(t *testing.T) {
	cases := []string{
		"user@example.com",
		"first.last@domain.org",
		"user+tag@example.co.uk",
	}
	for _, email := range cases {
		if fe := ValidateEmail(email); fe != nil {
			t.Errorf("ValidateEmail(%q) = %q, want nil", email, fe.Message)
		}
	}
}

func TestValidateEmailInvalid(t *testing.T) {
	cases := []struct {
		email   string
		wantMsg string
	}{
		{"", "email is required"},
		{"   ", "email is required"},
		{"notanemail", "not a valid email"},
		{"@missing-local.com", "not a valid email"},
		{strings.Repeat("a", 250) + "@b.com", "254 characters or fewer"},
	}
	for _, tc := range cases {
		fe := ValidateEmail(tc.email)
		if fe == nil {
			t.Errorf("ValidateEmail(%q) = nil, want error containing %q", tc.email, tc.wantMsg)
			continue
		}
		if fe.Field != "email" {
			t.Errorf("ValidateEmail(%q).Field = %q, want email", tc.email, fe.Field)
		}
		if !strings.Contains(fe.Message, tc.wantMsg) {
			t.Errorf("ValidateEmail(%q).Message = %q, want containing %q", tc.email, fe.Message, tc.wantMsg)
		}
	}
}

func TestValidatePasswordValid(t *testing.T) {
	cases := []string{
		"Str0ng!Pass",
		"MyP@ssw0rd",
		"Abcdefg1!",
	}
	for _, pw := range cases {
		if fe := ValidatePassword(pw); fe != nil {
			t.Errorf("ValidatePassword(%q) = %q, want nil", pw, fe.Message)
		}
	}
}

func TestValidatePasswordInvalid(t *testing.T) {
	cases := []struct {
		password string
		wantMsg  string
	}{
		{"", "password is required"},
		{"Sh0rt!", "at least 8 characters"},
		{strings.Repeat("A", 129) + "a1!", "128 characters or fewer"},
		{"alllowercase1!", "uppercase"},
		{"ALLUPPERCASE1!", "lowercase"},
		{"NoDigitsHere!", "digit"},
		{"NoSpecial1abc", "special character"},
	}
	for _, tc := range cases {
		fe := ValidatePassword(tc.password)
		if fe == nil {
			t.Errorf("ValidatePassword(%q) = nil, want error containing %q", tc.password, tc.wantMsg)
			continue
		}
		if fe.Field != "password" {
			t.Errorf("ValidatePassword(%q).Field = %q, want password", tc.password, fe.Field)
		}
		if !strings.Contains(fe.Message, tc.wantMsg) {
			t.Errorf("ValidatePassword(%q).Message = %q, want containing %q", tc.password, fe.Message, tc.wantMsg)
		}
	}
}

func TestValidateUsernameValid(t *testing.T) {
	cases := []string{
		"john",
		"john.doe",
		"john-doe",
		"john_doe",
		"j0hn",
		"abc",
	}
	for _, u := range cases {
		if fe := ValidateUsername(u); fe != nil {
			t.Errorf("ValidateUsername(%q) = %q, want nil", u, fe.Message)
		}
	}
}

func TestValidateUsernameInvalid(t *testing.T) {
	cases := []struct {
		username string
		wantMsg  string
	}{
		{"", "username is required"},
		{"ab", "at least 3 characters"},
		{strings.Repeat("a", 65), "64 characters or fewer"},
		{".startdot", "start and end with"},
		{"enddot.", "start and end with"},
		{"-startdash", "start and end with"},
		{"has spaces", "start and end with"},
	}
	for _, tc := range cases {
		fe := ValidateUsername(tc.username)
		if fe == nil {
			t.Errorf("ValidateUsername(%q) = nil, want error containing %q", tc.username, tc.wantMsg)
			continue
		}
		if fe.Field != "username" {
			t.Errorf("ValidateUsername(%q).Field = %q, want username", tc.username, fe.Field)
		}
		if !strings.Contains(fe.Message, tc.wantMsg) {
			t.Errorf("ValidateUsername(%q).Message = %q, want containing %q", tc.username, fe.Message, tc.wantMsg)
		}
	}
}

func TestValidateRegistrationAllValid(t *testing.T) {
	errs := ValidateRegistration("user@example.com", "Str0ng!Pass", "johndoe")
	if len(errs) != 0 {
		t.Errorf("ValidateRegistration() returned %d errors, want 0: %v", len(errs), errs)
	}
}

func TestValidateRegistrationMultipleErrors(t *testing.T) {
	errs := ValidateRegistration("", "", "")
	if len(errs) != 3 {
		t.Errorf("ValidateRegistration() returned %d errors, want 3", len(errs))
	}

	fields := map[string]bool{}
	for _, e := range errs {
		fields[e.Field] = true
	}
	for _, f := range []string{"email", "password", "username"} {
		if !fields[f] {
			t.Errorf("missing error for field %q", f)
		}
	}
}
