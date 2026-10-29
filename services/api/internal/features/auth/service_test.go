package auth

import (
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	if got, err := normalizeEmail("  Jane.Doe@Example.COM "); err != nil || got != "jane.doe@example.com" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{"", "plain", "a@b", "Jane <jane@example.com>", "a@b@c.com", "a b@c.com"} {
		if _, err := normalizeEmail(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if err := validatePassword("12345678"); err != nil {
		t.Errorf("8 chars must be accepted: %v", err)
	}
	for name, pw := range map[string]string{
		"short":      "1234567",
		"blank":      "        ",
		"too long":   strings.Repeat("a", 73),
		"multi-byte": strings.Repeat("é", 37), // 74 bytes: bcrypt would truncate
	} {
		if err := validatePassword(pw); err == nil {
			t.Errorf("%s password must be rejected", name)
		}
	}
}

func TestResetCodesAreUnambiguousAndNormalised(t *testing.T) {
	code, err := randomCode(resetCodeLength)
	if err != nil || len(code) != resetCodeLength {
		t.Fatalf("bad code %q: %v", code, err)
	}
	if strings.ContainsAny(code, "01OIl") {
		t.Fatalf("code uses ambiguous characters: %q", code)
	}
	if normalizeCode(" ab-cd ef ") != "ABCDEF" {
		t.Fatal("codes should be forgiving about case, spaces and dashes")
	}
}
