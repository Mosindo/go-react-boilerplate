package auth

import (
	"strings"
	"testing"
)

func TestValidatePasswordStrength(t *testing.T) {
	for _, ok := range []string{"Password123", "abc12345", "motdepasse9"} {
		if err := validatePasswordStrength(ok); err != nil {
			t.Fatalf("expected %q to be accepted: %v", ok, err)
		}
	}
	for _, bad := range []string{"password", "12345678", "a1"} {
		if err := validatePasswordStrength(bad); err == nil {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}

func TestGenerateResetCode(t *testing.T) {
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
			if !strings.ContainsRune(resetCodeAlphabet, r) {
				t.Fatalf("unexpected character %q", r)
			}
		}
		seen[code] = true
	}
	if len(seen) < 50 {
		t.Fatal("reset codes should be unique")
	}
}

func TestNormalizeResetCode(t *testing.T) {
	if got := normalizeResetCode(" abcd-efgh 23 "); got != "ABCDEFGH23" {
		t.Fatalf("unexpected normalization %q", got)
	}
}

func TestNormalizeEmailAndHash(t *testing.T) {
	if normalizeEmail("  Alice@Mail.TEST ") != "alice@mail.test" {
		t.Fatal("email should be trimmed and lowercased")
	}
	if hashSecret("") != "" || hashSecret("x") == hashSecret("y") {
		t.Fatal("unexpected hash behaviour")
	}
}

func TestValidateEmail(t *testing.T) {
	got, err := ValidateEmail("  Alice.Martin@Mail.Example  ")
	if err != nil || got != "alice.martin@mail.example" {
		t.Fatalf("unexpected %q %v", got, err)
	}
	for _, bad := range []string{"", "alice", "alice@", "@mail.test", "alice@localhost", "Alice <alice@mail.test>", "a b@mail.test"} {
		if _, err := ValidateEmail(bad); err == nil {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}
