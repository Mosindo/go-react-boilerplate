package auth

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	for _, ok := range []string{"Password1", "abc12345", "motdepasse9"} {
		if err := ValidatePassword(ok); err != nil {
			t.Errorf("expected %q to be accepted: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "short1", "onlyletters", "12345678"} {
		if err := ValidatePassword(bad); !errors.Is(err, ErrWeakPassword) {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM "); got != "alice@example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestRandomDigits(t *testing.T) {
	code, err := randomDigits(6)
	if err != nil || len(code) != 6 {
		t.Fatalf("unexpected code %q err=%v", code, err)
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			t.Fatalf("non digit in %q", code)
		}
	}
}

func TestHashTokenEmpty(t *testing.T) {
	if hashToken("   ") != "" {
		t.Fatal("empty token must hash to empty string")
	}
	if hashToken("a") == hashToken("b") {
		t.Fatal("distinct tokens must hash differently")
	}
}

func TestValidateEmail(t *testing.T) {
	for _, ok := range []string{"alice@example.com", "a.b+tag@sub.domain.fr"} {
		if err := ValidateEmail(ok); err != nil {
			t.Errorf("expected %q valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "nope", "a@b", "Alice <a@b.com>", "a@b.com.", "a b@c.com", "@x.com"} {
		if err := ValidateEmail(bad); err == nil {
			t.Errorf("expected %q invalid", bad)
		}
	}
}
