package auth

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	valid := []string{"Password1", "abcdefg9", "motdepasse2026"}
	invalid := []string{"short1", "onlyletters", "12345678", string(make([]byte, 73))}
	for _, p := range valid {
		if err := ValidatePassword(p); err != nil {
			t.Errorf("%q should be valid: %v", p, err)
		}
	}
	for _, p := range invalid {
		if err := ValidatePassword(p); !errors.Is(err, ErrWeakPassword) {
			t.Errorf("%q should be rejected", p)
		}
	}
}

func TestEmailNormalizationAndValidation(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM "); got != "alice@example.com" {
		t.Fatalf("unexpected normalization %q", got)
	}
	for _, e := range []string{"alice@example.com", "a.b+c@sub.example.fr"} {
		if !validEmail(e) {
			t.Errorf("%q should be valid", e)
		}
	}
	for _, e := range []string{"alice", "alice@localhost", "Alice <alice@example.com>", "a b@example.com"} {
		if validEmail(e) {
			t.Errorf("%q should be invalid", e)
		}
	}
}

func TestRandomCodeFormat(t *testing.T) {
	for i := 0; i < 50; i++ {
		code, err := randomCode()
		if err != nil || len(code) != 6 {
			t.Fatalf("bad code %q (%v)", code, err)
		}
	}
}
