package auth

import "testing"

func TestValidatePassword(t *testing.T) {
	for _, ok := range []string{"Password1", "correct horse 9", "motdepasse2026"} {
		if err := ValidatePassword(ok); err != nil {
			t.Errorf("%q should be accepted: %v", ok, err)
		}
	}
	long := make([]byte, 73)
	for i := range long {
		long[i] = 'a'
	}
	long[0] = '1'
	for _, bad := range []string{"short1", "onlyletters", "12345678", string(long)} {
		if err := ValidatePassword(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM "); got != "alice@example.com" {
		t.Fatalf("got %q", got)
	}
}
