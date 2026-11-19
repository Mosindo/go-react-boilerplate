package auth

import "testing"

func TestValidatePasswordStrength(t *testing.T) {
	for _, ok := range []string{"abcdefg1", "Motdepasse9", "élan1234"} {
		if err := validatePasswordStrength(ok); err != nil {
			t.Fatalf("%q should be accepted: %v", ok, err)
		}
	}
	for _, bad := range []string{"short1", "abcdefgh", "12345678"} {
		if err := validatePasswordStrength(bad); err == nil {
			t.Fatalf("%q should be rejected", bad)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM "); got != "alice@example.com" {
		t.Fatalf("NormalizeEmail = %q", got)
	}
}

func TestRandomDigits(t *testing.T) {
	code, err := randomDigits(6)
	if err != nil || len(code) != 6 {
		t.Fatalf("randomDigits = %q, %v", code, err)
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			t.Fatalf("non digit in %q", code)
		}
	}
}

func TestResetCodeHashIsBoundToUser(t *testing.T) {
	s := NewService(nil, []byte("0123456789abcdef0123456789abcdef"), nil, nil, WithBcryptCost(4))
	if s.hashResetCode("u1", "123456") == s.hashResetCode("u2", "123456") {
		t.Fatal("reset hash must depend on the user id")
	}
}

func TestValidEmail(t *testing.T) {
	for _, ok := range []string{"a@b.co", "first.last+tag@mail.example.org"} {
		if !ValidEmail(ok) {
			t.Fatalf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "nope", "a@b", "Name <a@b.co>", "a b@c.de", "@b.co"} {
		if ValidEmail(bad) {
			t.Fatalf("%q should be invalid", bad)
		}
	}
}
