package auth

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	valid := []string{"Password123", "abcdefg1", "çàé12345"}
	for _, p := range valid {
		if err := ValidatePassword(p); err != nil {
			t.Errorf("expected %q to be valid: %v", p, err)
		}
	}
	invalid := []string{"short1", "allletters", "12345678", ""}
	for _, p := range invalid {
		if err := ValidatePassword(p); !errors.Is(err, ErrWeakPassword) {
			t.Errorf("expected %q to be rejected", p)
		}
	}
}

func TestResetCodeFormat(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		code, err := generateResetCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != resetCodeLength {
			t.Fatalf("unexpected code length %d", len(code))
		}
		for _, r := range code {
			if !containsRune(resetCodeAlphabet, r) {
				t.Fatalf("unexpected rune %q in %s", r, code)
			}
		}
		seen[code] = true
	}
	if len(seen) < 45 {
		t.Fatal("reset codes are not random enough")
	}
	if hashResetCode("ABC") == hashResetCode("ABD") {
		t.Fatal("hash collision")
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

func TestNormalizeEmail(t *testing.T) {
	got, err := NormalizeEmail("  Lou.Martin+date@Example.ORG ")
	if err != nil || got != "lou.martin+date@example.org" {
		t.Fatalf("got %q %v", got, err)
	}
	for _, bad := range []string{"", "nope", "a@b", "Lou <lou@x.org>", "a@@b.org", "a b@c.org"} {
		if _, err := NormalizeEmail(bad); !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}
