package middleware

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("unit-test-secret-0123456789abcdef-xyz")

func sign(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims, key any) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(time.Minute).Unix()}
}

func TestParseAccessTokenAcceptsValidHS256(t *testing.T) {
	c, err := ParseAccessToken(secret, sign(t, jwt.SigningMethodHS256, validClaims(), secret))
	if err != nil || c.UserID != "u1" || c.SessionID != "s1" || c.ExpiresAt.IsZero() {
		t.Fatalf("unexpected result: %+v %v", c, err)
	}
}

func TestParseAccessTokenRejects(t *testing.T) {
	expired := validClaims()
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	noExp := validClaims()
	delete(noExp, "exp")
	noUID := validClaims()
	delete(noUID, "uid")
	noSID := validClaims()
	delete(noSID, "sid")
	emptyUID := validClaims()
	emptyUID["uid"] = "  "

	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims()).SignedString(jwt.UnsafeAllowNoneSignatureType)

	cases := map[string]string{
		"expired":         sign(t, jwt.SigningMethodHS256, expired, secret),
		"no expiry":       sign(t, jwt.SigningMethodHS256, noExp, secret),
		"no uid":          sign(t, jwt.SigningMethodHS256, noUID, secret),
		"no sid":          sign(t, jwt.SigningMethodHS256, noSID, secret),
		"blank uid":       sign(t, jwt.SigningMethodHS256, emptyUID, secret),
		"wrong secret":    sign(t, jwt.SigningMethodHS256, validClaims(), []byte("another-secret-another-secret-0000")),
		"HS512 not HS256": sign(t, jwt.SigningMethodHS512, validClaims(), secret),
		"alg none":        none,
		"garbage":         "a.b.c",
		"empty":           "",
	}
	for name, token := range cases {
		if _, err := ParseAccessToken(secret, token); err == nil {
			t.Errorf("%s: token must be rejected", name)
		}
	}
}

func TestBearerToken(t *testing.T) {
	for header, want := range map[string]string{
		"Bearer abc":  "abc",
		"bearer  abc": "abc",
		"Basic abc":   "",
		"Bearer":      "",
		"":            "",
	} {
		if got := bearerToken(header); got != want {
			t.Errorf("bearerToken(%q) = %q, want %q", header, got, want)
		}
	}
}
