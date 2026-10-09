package auth

import (
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	cases := map[string]bool{
		"  Alice@Example.COM ":             true,
		"a@b.co":                           true,
		"":                                 false,
		"no-at-sign":                       false,
		"a@b":                              false,
		"Alice <a@b.co>":                   false,
		"a@b.co\nBcc: x@y.z":               false,
		strings.Repeat("a", 250) + "@b.co": false,
	}
	for in, valid := range cases {
		got, err := normalizeEmail(in)
		if (err == nil) != valid {
			t.Errorf("normalizeEmail(%q) err=%v, want valid=%v", in, err, valid)
		}
		if valid && got != strings.ToLower(strings.TrimSpace(in)) {
			t.Errorf("normalizeEmail(%q) = %q, not normalized", in, got)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if err := validatePassword("short", "a@b.co"); err == nil {
		t.Error("short password accepted")
	}
	if err := validatePassword(strings.Repeat("x", 73), "a@b.co"); err == nil {
		t.Error("password longer than bcrypt limit accepted")
	}
	if err := validatePassword("a@b.co.long", "a@b.co.long"); err == nil {
		t.Error("password equal to email accepted")
	}
	if err := validatePassword("correct horse battery", "a@b.co"); err != nil {
		t.Errorf("good password rejected: %v", err)
	}
}

func TestResetCodeShape(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		code, err := generateResetCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != resetCodeLength || strings.Trim(code, resetCodeAlphabet) != "" {
			t.Fatalf("unexpected code %q", code)
		}
		seen[code] = true
	}
	if len(seen) < 49 {
		t.Fatal("reset codes are not random enough")
	}
}

func TestHashRefreshTokenIsDeterministicAndEmptySafe(t *testing.T) {
	if hashRefreshToken("  ") != "" {
		t.Error("blank token must hash to empty string")
	}
	if hashRefreshToken("abc") != hashRefreshToken("abc") || hashRefreshToken("abc") == hashRefreshToken("abd") {
		t.Error("hash must be deterministic and collision free for distinct inputs")
	}
}
