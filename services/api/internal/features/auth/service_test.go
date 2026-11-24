package auth

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	cases := map[string]bool{
		"short1":                 false,
		"onlyletters":            false,
		"12345678":               false,
		"letters123":             true,
		"Pässwörd9":              true,
		string(make([]byte, 73)): false,
	}
	for password, ok := range cases {
		err := ValidatePassword(password)
		if ok && err != nil {
			t.Errorf("%q: expected valid, got %v", password, err)
		}
		if !ok && !errors.Is(err, ErrWeakPassword) {
			t.Errorf("%q: expected weak password error, got %v", password, err)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM "); got != "alice@example.com" {
		t.Fatalf("unexpected normalisation %q", got)
	}
}

func TestRandomDigits(t *testing.T) {
	code, err := randomDigits(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 8 {
		t.Fatalf("expected 8 digits, got %q", code)
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			t.Fatalf("non digit in %q", code)
		}
	}
}

func TestResetCodeHashIsBoundToUser(t *testing.T) {
	if hashResetCode("u1", "12345678") == hashResetCode("u2", "12345678") {
		t.Fatal("reset hash must depend on user id")
	}
}

func TestParseEmail(t *testing.T) {
	if got, err := ParseEmail("  Bob@Mail.FR "); err != nil || got != "bob@mail.fr" {
		t.Fatalf("unexpected %q %v", got, err)
	}
	for _, bad := range []string{"bob", "bob@", "@mail.fr", "bob@localhost", "Bob <bob@mail.fr>", "a b@mail.fr"} {
		if _, err := ParseEmail(bad); !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestTruncateKeepsValidUTF8(t *testing.T) {
	value := string(make([]byte, 254)) + "é" + "x"
	got := truncate(value, 255)
	if len(got) != 254 {
		t.Fatalf("expected cut before the multibyte rune, got %d bytes", len(got))
	}
	if got := truncate("ok\xff", 255); got != "ok" {
		t.Fatalf("invalid bytes must be dropped, got %q", got)
	}
}
