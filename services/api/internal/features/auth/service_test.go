package auth

import "testing"

func TestValidatePassword(t *testing.T) {
	for _, ok := range []string{"Password123", "abcdefg1", "motdepasse9é"} {
		if ValidatePassword(ok) != nil {
			t.Fatalf("%q should be accepted", ok)
		}
	}
	for _, bad := range []string{"", "short1", "onlyletters", "12345678", string(make([]byte, 129))} {
		if ValidatePassword(bad) == nil {
			t.Fatalf("%q should be rejected", bad)
		}
	}
}

func TestValidEmail(t *testing.T) {
	for _, ok := range []string{"a@b.co", "first.last+tag@example.org"} {
		if !ValidEmail(ok) {
			t.Fatalf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "nope", "a@b", "John <a@b.co>", "a b@c.de", "a@b.co\nBcc: x@y.z"} {
		if ValidEmail(bad) {
			t.Fatalf("%q should be invalid", bad)
		}
	}
}

func TestResetCodeHashIsBoundToUser(t *testing.T) {
	s := NewService(nil, []byte("unit-test-key-0123456789abcdef0123"))
	if s.hashResetCode("u1", "123456") == s.hashResetCode("u2", "123456") {
		t.Fatal("reset code hash must depend on the user")
	}
	code, err := generateResetCode()
	if err != nil || len(code) != 6 {
		t.Fatalf("unexpected code %q (%v)", code, err)
	}
}
