package auth

import "testing"

func TestPasswordStrength(t *testing.T) {
	good := []string{"Passw0rd!", "abcdefg1", "motdepasse2026"}
	bad := []string{"short1", "onlyletters", "12345678", ""}
	for _, p := range good {
		if !passwordIsStrongEnough(p) {
			t.Errorf("%q should be accepted", p)
		}
	}
	for _, p := range bad {
		if passwordIsStrongEnough(p) {
			t.Errorf("%q should be rejected", p)
		}
	}
}

func TestValidEmail(t *testing.T) {
	for _, e := range []string{"a@b.co", "first.last@example.com"} {
		if !validEmail(e) {
			t.Errorf("%q should be valid", e)
		}
	}
	for _, e := range []string{"", "a", "a@b", "Name <a@b.co>", "a@@b.co"} {
		if validEmail(e) {
			t.Errorf("%q should be invalid", e)
		}
	}
}

func TestHashTokenIsStableAndEmptySafe(t *testing.T) {
	if hashToken("x") != hashToken("x") || hashToken("x") == hashToken("y") {
		t.Fatal("hash must be deterministic and collision-free for distinct inputs")
	}
	if hashToken("  ") != "" {
		t.Fatal("blank token must hash to empty")
	}
}
