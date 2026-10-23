package auth

import (
	"strings"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	cases := map[string]bool{
		"Password123":                   true,
		"short1":                        false,
		"alllettersnodigits":            false,
		"12345678901":                   false,
		strings.Repeat("a1", 37):        false, // 74 bytes
		strings.Repeat("a", 71) + "1":   true,
		"mot de passe 2024 très solide": true,
	}
	for pw, ok := range cases {
		if err := validatePassword(pw); (err == nil) != ok {
			t.Errorf("validatePassword(%q) ok=%v, want %v", pw, err == nil, ok)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	good, err := validateEmail("  Alice@Example.COM ")
	if err != nil || good != "alice@example.com" {
		t.Fatalf("expected normalized email, got %q %v", good, err)
	}
	for _, bad := range []string{"", "nope", "a@b", "Alice <a@example.com>", "a@example.com, b@example.com", "a b@example.com"} {
		if _, err := validateEmail(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}

func TestResetCodeFormat(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		code, err := generateResetCode()
		if err != nil || len(code) != resetCodeLength {
			t.Fatalf("bad code %q: %v", code, err)
		}
		for _, r := range code {
			if !strings.ContainsRune(resetCodeAlphabet, r) {
				t.Fatalf("unexpected char %q in %q", r, code)
			}
		}
		seen[code] = true
	}
	if len(seen) < 49 {
		t.Fatal("reset codes are not random enough")
	}
}

func TestHashRefreshTokenIgnoresWhitespaceAndEmpty(t *testing.T) {
	if hashRefreshToken("   ") != "" {
		t.Fatal("blank token must hash to empty")
	}
	if hashRefreshToken(" abc ") != hashRefreshToken("abc") {
		t.Fatal("token should be trimmed before hashing")
	}
}
