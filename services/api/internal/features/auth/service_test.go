package auth

import (
	"testing"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/golang-jwt/jwt/v5"
)

func TestAgeOn(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // 18th birthday today
		{"2008-06-16", 17}, // turns 18 tomorrow
		{"2008-12-31", 17},
		{"1990-01-01", 36},
		{"2000-02-29", 26},
	}
	for _, c := range cases {
		birth, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeOn(birth, now); got != c.want {
			t.Errorf("AgeOn(%s) = %d, want %d", c.birth, got, c.want)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	good, err := NormalizeEmail("  Jane.Doe@Example.COM ")
	if err != nil || good != "jane.doe@example.com" {
		t.Fatalf("got %q %v", good, err)
	}
	for _, bad := range []string{"", "plain", "a@b", "Jane <jane@example.com>", "a@@b.com", "x@y.com\nBcc: z@z.com"} {
		if _, err := NormalizeEmail(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if err := validatePassword("1234567"); err == nil {
		t.Error("7 chars must be rejected")
	}
	if err := validatePassword("12345678"); err != nil {
		t.Errorf("8 chars must pass: %v", err)
	}
	long := make([]byte, 73)
	for i := range long {
		long[i] = 'a'
	}
	if err := validatePassword(string(long)); err == nil {
		t.Error("passwords beyond bcrypt's 72 bytes must be rejected, not silently truncated")
	}
}

func TestAccessTokenValidation(t *testing.T) {
	secret := []byte("a-very-long-secret-for-unit-tests-0123456789")
	token, err := signAccessToken(secret, "user-1", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	uid, sid, err := middleware.ParseAccessToken(secret, token)
	if err != nil || uid != "user-1" || sid != "session-1" {
		t.Fatalf("valid token rejected: %v", err)
	}
	if _, _, err := middleware.ParseAccessToken([]byte("another-secret-another-secret-123456"), token); err == nil {
		t.Error("token signed with another secret must be rejected")
	}

	// alg=none and HS512 must be refused, and expiry is mandatory.
	none := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"uid": "u", "sid": "s", "exp": time.Now().Add(time.Hour).Unix()})
	noneStr, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, _, err := middleware.ParseAccessToken(secret, noneStr); err == nil {
		t.Error("alg=none must be rejected")
	}
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{"uid": "u", "sid": "s", "exp": time.Now().Add(time.Hour).Unix()}).SignedString(secret)
	if _, _, err := middleware.ParseAccessToken(secret, hs512); err == nil {
		t.Error("HS512 must be rejected")
	}
	noExp, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u", "sid": "s"}).SignedString(secret)
	if _, _, err := middleware.ParseAccessToken(secret, noExp); err == nil {
		t.Error("tokens without expiry must be rejected")
	}
	expired, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u", "sid": "s", "exp": time.Now().Add(-time.Minute).Unix()}).SignedString(secret)
	if _, _, err := middleware.ParseAccessToken(secret, expired); err == nil {
		t.Error("expired tokens must be rejected")
	}
}
