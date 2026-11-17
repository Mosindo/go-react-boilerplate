package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	cases := map[string]bool{
		"short1":                 false,
		"onlyletters":            false,
		"12345678":               false,
		"Password123":            true,
		"motdepasse9":            true,
		strings.Repeat("a1", 40): false, // beyond bcrypt's 72 bytes
		"élégant99":              true,
	}
	for password, ok := range cases {
		err := ValidatePassword(password)
		if ok && err != nil {
			t.Errorf("expected %q to be accepted, got %v", password, err)
		}
		if !ok && !errors.Is(err, ErrWeakPassword) {
			t.Errorf("expected %q to be rejected", password)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM "); got != "alice@example.com" {
		t.Fatalf("unexpected normalized email %q", got)
	}
}

func TestRandomCodeAlphabet(t *testing.T) {
	code, err := randomCode()
	if err != nil {
		t.Fatalf("random code: %v", err)
	}
	if len(code) != resetCodeLength {
		t.Fatalf("unexpected code length %d", len(code))
	}
	for _, r := range code {
		if !strings.ContainsRune(resetCodeAlphabet, r) {
			t.Fatalf("unexpected character %q", r)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	for _, ok := range []string{"a@b.co", "first.last+tag@example.org"} {
		if err := ValidateEmail(ok); err != nil {
			t.Errorf("expected %q valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "plain", "a@b", "Name <a@b.co>", "a@@b.co", "a b@c.co"} {
		if err := ValidateEmail(bad); err == nil {
			t.Errorf("expected %q invalid", bad)
		}
	}
}
