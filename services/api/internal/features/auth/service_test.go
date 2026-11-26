package auth

import "testing"

func TestValidatePassword(t *testing.T) {
	for _, ok := range []string{"Password1", "motdepasse9", "12345678a"} {
		if err := ValidatePassword(ok); err != nil {
			t.Errorf("%q should be accepted", ok)
		}
	}
	for _, bad := range []string{"short1", "onlyletters", "1234567890", ""} {
		if err := ValidatePassword(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestEmailNormalisationAndValidation(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM "); got != "alice@example.com" {
		t.Fatalf("got %q", got)
	}
	for _, ok := range []string{"a@b.co", "first.last+tag@sub.example.org"} {
		if !ValidEmail(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "nope", "a@b", "Alice <a@b.co>", "a@@b.co", "a b@c.co"} {
		if ValidEmail(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestResetCodeFormat(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		code, err := generateResetCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != resetCodeLength {
			t.Fatalf("unexpected length %d", len(code))
		}
		for _, r := range code {
			if !containsRune(resetCodeAlphabet, r) {
				t.Fatalf("unexpected symbol %q", r)
			}
		}
		if seen[code] {
			t.Fatal("codes must be unique")
		}
		seen[code] = true
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
