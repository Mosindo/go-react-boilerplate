package auth

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	cases := map[string]bool{
		"short1":            false,
		"onlyletters":       false,
		"12345678":          false,
		"Password123":       true,
		"motdepasseé1":      true,
		"averyveryverylong": false,
		"Pass1":             false,
	}
	for input, ok := range cases {
		err := ValidatePassword(input)
		if ok && err != nil {
			t.Errorf("expected %q to be accepted, got %v", input, err)
		}
		if !ok && !errors.Is(err, ErrWeakPassword) {
			t.Errorf("expected %q to be rejected", input)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM "); got != "alice@example.com" {
		t.Fatalf("unexpected normalized email %q", got)
	}
}

func TestGenerateNumericCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		code, err := generateNumericCode(6)
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != 6 {
			t.Fatalf("unexpected code length %q", code)
		}
		for _, r := range code {
			if r < '0' || r > '9' {
				t.Fatalf("non numeric code %q", code)
			}
		}
		seen[code] = true
	}
	if len(seen) < 45 {
		t.Fatal("codes are not random enough")
	}
}

func TestValidateEmail(t *testing.T) {
	if got, err := ValidateEmail("  Alice@Example.COM "); err != nil || got != "alice@example.com" {
		t.Fatalf("got %q %v", got, err)
	}
	for _, bad := range []string{"", "alice", "alice@", "@example.com", "Alice <alice@example.com>", "alice@localhost", "a b@example.com"} {
		if _, err := ValidateEmail(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}
