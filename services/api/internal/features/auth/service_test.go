package auth

import "testing"

func TestValidatePassword(t *testing.T) {
	ok := []string{"Password1", "abcdefg1", "12345678a", "пароль123"}
	bad := []string{"", "short1", "alllettersonly", "123456789", "a1" + string(make([]byte, 80))}
	for _, p := range ok {
		if err := ValidatePassword(p); err != nil {
			t.Errorf("%q should be accepted: %v", p, err)
		}
	}
	for _, p := range bad {
		if err := ValidatePassword(p); err == nil {
			t.Errorf("%q should be refused", p)
		}
	}
}

func TestValidEmail(t *testing.T) {
	for _, e := range []string{"a@b.co", "first.last+tag@example.org"} {
		if !ValidEmail(e) {
			t.Errorf("%q should be valid", e)
		}
	}
	for _, e := range []string{"", "plain", "a@b", "Name <a@b.co>", "a b@c.de", "@b.co", "a@@b.co"} {
		if ValidEmail(e) {
			t.Errorf("%q should be invalid", e)
		}
	}
}

func TestResetCodeIsEightDigits(t *testing.T) {
	for i := 0; i < 50; i++ {
		code, err := generateResetCode()
		if err != nil || len(code) != 8 {
			t.Fatalf("bad code %q (%v)", code, err)
		}
	}
	if hashResetCode("u1", "12345678") == hashResetCode("u2", "12345678") {
		t.Fatal("reset code hash must be bound to the user")
	}
}
