package auth

import (
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := NormalizeEmail("  Alice.Doe@Example.COM ")
	if err != nil || got != "alice.doe@example.com" {
		t.Fatalf("unexpected %q %v", got, err)
	}
	for _, bad := range []string{"", "nope", "a@b", "Alice <a@b.com>", "a@@b.com"} {
		if _, err := NormalizeEmail(bad); err == nil {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("Password123"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
	for _, bad := range []string{"short1", "onlyletters", "1234567890", strings.Repeat("a1", 37)} {
		if err := ValidatePassword(bad); err == nil {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}

func TestResetCodeFormat(t *testing.T) {
	code, err := newResetCode()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != resetCodeLength {
		t.Fatalf("unexpected code length %d", len(code))
	}
	if normalizeResetCode(" "+formatResetCode(code)+" ") != code {
		t.Fatal("formatted code must normalize back to the raw code")
	}
	if normalizeResetCode("abcde-fghjk") != "ABCDEFGHJK" {
		t.Fatal("normalization must upper-case and strip separators")
	}
}
