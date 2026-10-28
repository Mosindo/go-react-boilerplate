package auth

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	for _, ok := range []string{"a@b.co", "first.last+tag@sub.example.com"} {
		if err := validateEmail(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "plain", "a@b", "@b.co", "a@", "Name <a@b.co>", "a b@c.co", strings.Repeat("a", 250) + "@b.co"} {
		if err := validateEmail(bad); err == nil {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if validatePassword("1234567") == nil || validatePassword(strings.Repeat("x", 73)) == nil {
		t.Error("length bounds not enforced")
	}
	if validatePassword("12345678") != nil || validatePassword(strings.Repeat("x", 72)) != nil {
		t.Error("boundary values should pass")
	}
}

func TestResetCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		code, err := generateResetCode()
		if err != nil || len(code) != resetCodeLen {
			t.Fatalf("bad code %q %v", code, err)
		}
		for _, r := range code {
			if !strings.ContainsRune(resetAlphabet, r) {
				t.Fatalf("char %q outside alphabet", r)
			}
		}
		seen[code] = true
	}
	if len(seen) < 45 {
		t.Error("codes should be random")
	}
	if normalizeCode(" ab-cd2345 ") != "ABCD2345" {
		t.Error("normalizeCode failed")
	}
	s := &Service{jwtSecret: []byte("k")}
	if s.hashResetCode("a@b.co", "X") == s.hashResetCode("c@d.co", "X") {
		t.Error("hash must bind the email")
	}
}
