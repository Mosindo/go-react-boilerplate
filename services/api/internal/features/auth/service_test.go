package auth

import "testing"

func TestValidEmail(t *testing.T) {
	for _, ok := range []string{"a.b@example.org", "x+tag@sub.domain.fr"} {
		if !ValidEmail(ok) {
			t.Fatalf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "plain", "a@b", "a@@b.com", "a b@c.com", "Name <a@b.com>", "a@.com", "a@com."} {
		if ValidEmail(bad) {
			t.Fatalf("%q should be invalid", bad)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if validatePassword("Password123") != nil {
		t.Fatal("expected valid password")
	}
	for _, bad := range []string{"short1", "onlyletters", "1234567890"} {
		if validatePassword(bad) == nil {
			t.Fatalf("%q should be rejected", bad)
		}
	}
}

func TestResetCodeAlphabetAndLength(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		code, err := generateResetCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != resetCodeLength {
			t.Fatalf("unexpected length %d", len(code))
		}
		for _, r := range code {
			found := false
			for _, a := range resetCodeAlphabet {
				if a == r {
					found = true
				}
			}
			if !found {
				t.Fatalf("unexpected symbol %q", r)
			}
		}
		seen[code] = true
	}
	if len(seen) < 199 {
		t.Fatal("codes should be random")
	}
}
