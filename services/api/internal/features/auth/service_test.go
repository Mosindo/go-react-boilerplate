package auth

import "testing"

func TestValidatePassword(t *testing.T) {
	for _, p := range []string{"short1", "allletters", "12345678", ""} {
		if ValidatePassword(p) == nil {
			t.Errorf("%q should be rejected", p)
		}
	}
	if ValidatePassword("Secret123") != nil {
		t.Error("valid password rejected")
	}
}

func TestValidateEmail(t *testing.T) {
	got, err := ValidateEmail("  Alice@Example.COM ")
	if err != nil || got != "alice@example.com" {
		t.Fatalf("unexpected %q %v", got, err)
	}
	for _, e := range []string{"nope", "a@b", "Alice <a@b.com>", "a@b.com.", "@b.com"} {
		if _, err := ValidateEmail(e); err == nil {
			t.Errorf("%q should be rejected", e)
		}
	}
}

func TestRandomDigits(t *testing.T) {
	code, err := randomDigits(6)
	if err != nil || len(code) != 6 {
		t.Fatalf("unexpected code %q %v", code, err)
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Fatalf("non digit in %q", code)
		}
	}
}
