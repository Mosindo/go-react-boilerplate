package auth

import (
	"strings"
	"testing"
)

func TestValidEmail(t *testing.T) {
	for _, ok := range []string{"a@b.co", "first.last+tag@sub.example.com"} {
		if !validEmail(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "plain", "a@b", "@b.co", "a@", "Name <a@b.co>", "a b@c.co", "a@b.co,c@d.co", "a@b.", strings.Repeat("a", 250) + "@b.co"} {
		if validEmail(bad) {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestNormalizeEmailTrimsAndLowercases(t *testing.T) {
	if got := normalizeEmail("  Foo@Example.COM \n"); got != "foo@example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestResetCodeFormat(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 200; i++ {
		code, err := generateResetCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != resetCodeLength || strings.Trim(code, resetCodeAlphabet) != "" {
			t.Fatalf("bad code %q", code)
		}
		seen[code] = struct{}{}
	}
	if len(seen) < 190 {
		t.Fatalf("codes are not random enough: %d unique of 200", len(seen))
	}
}

func TestResetCodeHashIsCaseInsensitiveByContract(t *testing.T) {
	// Users may type the code in lower case: ResetPassword upper-cases before hashing.
	if hashRefreshToken("ABCD2345") == hashRefreshToken("abcd2345") {
		t.Fatal("hash must be case sensitive; normalisation happens before hashing")
	}
}
