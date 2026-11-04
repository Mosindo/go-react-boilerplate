package auth

import "testing"

func TestNormalizeEmail(t *testing.T) {
	got, err := NormalizeEmail("  Jane.Doe@Example.COM ")
	if err != nil || got != "jane.doe@example.com" {
		t.Fatalf("got %q %v", got, err)
	}
	for _, bad := range []string{"", "plain", "a@b", "Jane <j@x.com>", "a b@c.com", "@x.com"} {
		if _, err := NormalizeEmail(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	for _, ok := range []string{"abcdefg1", "Sup3rSecretPass", "pässwörd9"} {
		if err := ValidatePassword(ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"short1", "alllettersonly", "123456789", "", string(make([]byte, 80))} {
		if err := ValidatePassword(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestHashTokenIsStableAndNeverEmptyForInput(t *testing.T) {
	if hashToken("") != "" {
		t.Fatal("empty token must hash to empty string so it can never match a session")
	}
	if hashToken("abc") != hashToken("abc") || hashToken("abc") == hashToken("abd") {
		t.Fatal("hash must be deterministic and collision-free on distinct inputs")
	}
}

func TestNumericCode(t *testing.T) {
	c, err := numericCode(8)
	if err != nil || len(c) != 8 {
		t.Fatalf("got %q %v", c, err)
	}
}
