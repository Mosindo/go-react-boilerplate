package auth

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	for _, ok := range []string{"Password123", "abcdefg1", "mot de passe 2026"} {
		if err := ValidatePassword(ok); err != nil {
			t.Fatalf("%q should be accepted: %v", ok, err)
		}
	}
	for _, bad := range []string{"short1", "onlyletters", "1234567890", ""} {
		if err := ValidatePassword(bad); !errors.Is(err, ErrWeakPassword) {
			t.Fatalf("%q should be rejected", bad)
		}
	}
}

func TestGenerateResetCodeFormat(t *testing.T) {
	for i := 0; i < 50; i++ {
		code, err := generateResetCode()
		if err != nil || len(code) != 6 {
			t.Fatalf("unexpected code %q err=%v", code, err)
		}
	}
}

func TestResetCodeHashIsBoundToUser(t *testing.T) {
	svc := NewService(nil, []byte("0123456789abcdef0123456789abcdef"))
	if svc.resetCodeHash("u1", "123456") == svc.resetCodeHash("u2", "123456") {
		t.Fatal("hash must depend on the user")
	}
}
