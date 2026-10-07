package auth

import "testing"

func TestResetCodeFormatting(t *testing.T) {
	code, err := generateResetCode()
	if err != nil || len(code) != resetCodeLength {
		t.Fatalf("code %q err %v", code, err)
	}
	display := formatResetCode(code)
	if normalizeResetCode(display) != code || normalizeResetCode(" "+display+" ") != code {
		t.Fatalf("round trip failed for %q", display)
	}
	if hashResetCode(code) == hashResetCode(code+"x") {
		t.Fatal("hash must depend on the code")
	}
}

func TestValidateEmailAndPassword(t *testing.T) {
	for _, ok := range []string{"a@b.co", "  A.B+tag@Example.com "} {
		if err := validateEmail(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "nope", "a@b", "Name <a@b.co>", "a b@c.co"} {
		if err := validateEmail(bad); err == nil {
			t.Errorf("%q should be invalid", bad)
		}
	}
	if validatePassword("1234567") == nil || validatePassword(string(make([]byte, 73))) == nil || validatePassword("12345678") != nil {
		t.Fatal("password bounds are 8..72 bytes")
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := normalizeEmail("  Foo@Bar.COM "); got != "foo@bar.com" {
		t.Fatalf("got %q", got)
	}
}
