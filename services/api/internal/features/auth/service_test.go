package auth

import "testing"

func TestNormalizeEmail(t *testing.T) {
	good := map[string]string{"  Foo@Example.COM ": "foo@example.com", "a.b+c@sub.domain.io": "a.b+c@sub.domain.io"}
	for in, want := range good {
		got, err := NormalizeEmail(in)
		if err != nil || got != want {
			t.Errorf("NormalizeEmail(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "plain", "a@b", "Name <a@b.co>", "a@b.co, c@d.co", "a b@c.co"} {
		if _, err := NormalizeEmail(in); err == nil {
			t.Errorf("NormalizeEmail(%q) should fail", in)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if ValidatePassword("1234567") == nil {
		t.Error("7 chars must be rejected")
	}
	if ValidatePassword("12345678") != nil {
		t.Error("8 chars must pass")
	}
	long := make([]byte, 73)
	for i := range long {
		long[i] = 'a'
	}
	if ValidatePassword(string(long)) == nil {
		t.Error("73 bytes exceed bcrypt limit")
	}
}
