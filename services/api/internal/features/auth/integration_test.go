package auth

import (
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/golang-jwt/jwt/v5"
)

func TestNormalizeEmail(t *testing.T) {
	good := map[string]string{
		"  Foo.Bar@Example.COM ": "foo.bar@example.com",
		"a+tag@sub.domain.io":    "a+tag@sub.domain.io",
	}
	for in, want := range good {
		got, err := normalizeEmail(in)
		if err != nil || got != want {
			t.Errorf("normalizeEmail(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "plain", "a@b", "Name <a@b.co>", "a@b.co, c@d.co", strings.Repeat("a", 250) + "@b.co", "a b@c.co"} {
		if _, err := normalizeEmail(in); err == nil {
			t.Errorf("normalizeEmail(%q) should fail", in)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if validatePassword("1234567", "p") == nil {
		t.Error("7 chars accepted")
	}
	if validatePassword("12345678", "p") != nil {
		t.Error("8 chars rejected")
	}
	if validatePassword(strings.Repeat("a", 73), "p") == nil {
		t.Error("73 bytes accepted (bcrypt would truncate)")
	}
}

func TestAccessTokenRoundTripAndTamper(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	token, err := signAccessToken(secret, "user-1", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	uid, sid, err := middleware.ParseAccessToken(secret, token)
	if err != nil || uid != "user-1" || sid != "session-1" {
		t.Fatalf("parse: %q %q %v", uid, sid, err)
	}
	if _, _, err := middleware.ParseAccessToken([]byte("another-secret-another-secret-123"), token); err == nil {
		t.Error("token accepted with a different secret")
	}

	// Tokens without expiry or signed with another HMAC alg must be refused.
	noExp := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u", "sid": "s"})
	raw, _ := noExp.SignedString(secret)
	if _, _, err := middleware.ParseAccessToken(secret, raw); err == nil {
		t.Error("token without exp accepted")
	}
	hs512 := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{"uid": "u", "sid": "s", "exp": time.Now().Add(time.Hour).Unix()})
	raw, _ = hs512.SignedString(secret)
	if _, _, err := middleware.ParseAccessToken(secret, raw); err == nil {
		t.Error("HS512 token accepted")
	}
	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u", "sid": "s", "exp": time.Now().Add(-time.Minute).Unix()})
	raw, _ = expired.SignedString(secret)
	if _, _, err := middleware.ParseAccessToken(secret, raw); err == nil {
		t.Error("expired token accepted")
	}
}

func TestTokenHashing(t *testing.T) {
	a, b := hashToken("abc"), hashToken("abc")
	if a != b || a == "abc" || len(a) != 64 {
		t.Fatalf("hash: %q", a)
	}
	if hashToken("") != "" {
		t.Error("empty token must hash to empty (rejected)")
	}
	t1, _ := generateToken()
	t2, _ := generateToken()
	if t1 == t2 || len(t1) < 40 {
		t.Error("tokens must be random and long")
	}
}
