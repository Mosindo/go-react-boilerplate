package auth

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	for _, ok := range []string{"Password123", "abcdefg1", "mot de passe 2 lueur"} {
		if err := ValidatePassword(ok); err != nil {
			t.Errorf("%q should be accepted: %v", ok, err)
		}
	}
	for _, bad := range []string{"short1", "onlyletters", "12345678", ""} {
		if err := ValidatePassword(bad); !errors.Is(err, ErrWeakPassword) {
			t.Errorf("%q should be rejected, got %v", bad, err)
		}
	}
}

func TestResetCodeFormatAndBinding(t *testing.T) {
	code, err := generateResetCode()
	if err != nil || len(code) != 6 {
		t.Fatalf("unexpected code %q (%v)", code, err)
	}
	if hashResetCode("user-a", code) == hashResetCode("user-b", code) {
		t.Fatalf("code hashes must be bound to the user")
	}
}

func TestValidEmail(t *testing.T) {
	for _, ok := range []string{"ana@lueur.app", "a.b+tag@sub.domain.fr"} {
		if !ValidEmail(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "ana", "ana@localhost", "Ana <ana@lueur.app>", "a@@b.c", "ana@lueur.app\n"} {
		if ValidEmail(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
