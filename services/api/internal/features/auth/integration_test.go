package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	for _, ok := range []string{"12345678", strings.Repeat("a", 72)} {
		if err := validatePassword(ok); err != nil {
			t.Fatalf("%q should be accepted: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "1234567", strings.Repeat("a", 73)} {
		if err := validatePassword(bad); !errors.Is(err, ErrInvalidPassword) {
			t.Fatalf("%d bytes should be rejected, got %v", len(bad), err)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := normalizeEmail("  Jane.Doe@Example.COM "); got != "jane.doe@example.com" {
		t.Fatalf("unexpected normalization: %q", got)
	}
}

func TestHashRefreshTokenIsStableAndNotPlaintext(t *testing.T) {
	a, b := hashRefreshToken("token"), hashRefreshToken("token")
	if a != b || a == "token" || len(a) != 64 {
		t.Fatalf("unexpected hash %q", a)
	}
	if hashRefreshToken("   ") != "" {
		t.Fatal("blank tokens must hash to empty")
	}
}
