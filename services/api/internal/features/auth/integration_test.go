package auth

import (
	"strings"
	"testing"

	platformerrors "example.com/api/internal/platform/errors"
	"github.com/golang-jwt/jwt/v5"
)

func TestValidateEmail(t *testing.T) {
	good := map[string]string{"  Bob@Example.ORG ": "bob@example.org", "a.b+c@sub.example.fr": "a.b+c@sub.example.fr"}
	for in, want := range good {
		got, err := validateEmail(in)
		if err != nil || got != want {
			t.Fatalf("validateEmail(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "plain", "a@b", "a@@b.com", "Bob <bob@example.org>", strings.Repeat("a", 250) + "@example.org"} {
		if _, err := validateEmail(in); err == nil {
			t.Fatalf("validateEmail(%q) should fail", in)
		} else if _, ok := err.(*platformerrors.ValidationError); !ok {
			t.Fatalf("expected a validation error, got %T", err)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if validatePassword("short") == nil {
		t.Fatal("short password accepted")
	}
	if validatePassword(strings.Repeat("a", 73)) == nil {
		t.Fatal("password beyond the bcrypt limit accepted")
	}
	if err := validatePassword("long-enough-password"); err != nil {
		t.Fatal(err)
	}
}

func TestHashTokenIsStableAndEmptySafe(t *testing.T) {
	if hashToken("") != "" || hashToken("   ") != "" {
		t.Fatal("empty tokens must hash to empty")
	}
	if hashToken("abc") != hashToken("abc") || hashToken("abc") == hashToken("abd") {
		t.Fatal("hash must be deterministic and collision free on distinct inputs")
	}
}

func TestAccessTokenIsHS256WithExpiry(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	raw, err := signAccessToken(secret, User{ID: "u1"}, "s1")
	if err != nil {
		t.Fatal(err)
	}
	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) { return secret, nil })
	if err != nil || !tok.Valid || tok.Method.Alg() != "HS256" {
		t.Fatalf("invalid token: %v", err)
	}
	exp, _ := claims.GetExpirationTime()
	if exp == nil || claims["uid"] != "u1" || claims["sid"] != "s1" {
		t.Fatalf("unexpected claims %v", claims)
	}
}
