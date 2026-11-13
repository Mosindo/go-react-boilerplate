package auth

import (
	"errors"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	ok := map[string]string{
		"  Alice@Example.COM ":           "alice@example.com",
		"bob.smith+tag@mail.example.org": "bob.smith+tag@mail.example.org",
	}
	for in, want := range ok {
		got, err := NormalizeEmail(in)
		if err != nil || got != want {
			t.Errorf("NormalizeEmail(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "not-an-email", "Alice <alice@example.com>", "a@b", "a@@b.com", "x@y.com\r\nBcc: z@y.com"} {
		if _, err := NormalizeEmail(bad); !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("NormalizeEmail(%q) should fail", bad)
		}
	}
}

func TestResetCodeFormat(t *testing.T) {
	for i := 0; i < 50; i++ {
		code, err := generateResetCode()
		if err != nil || len(code) != 6 {
			t.Fatalf("invalid code %q (%v)", code, err)
		}
	}
	s := &Service{jwtSecret: []byte("secret-secret-secret-secret-secret")}
	if s.hashResetCode("u1", "123456") == s.hashResetCode("u2", "123456") {
		t.Fatal("reset code hash must be bound to the user")
	}
}
