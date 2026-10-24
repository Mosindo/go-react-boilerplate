package middleware

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("unit-test-secret-unit-test-secret")

func sign(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestParseAccessToken(t *testing.T) {
	good := jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(time.Minute).Unix()}

	if c, err := ParseAccessToken(secret, sign(t, jwt.SigningMethodHS256, secret, good)); err != nil || c.UserID != "u1" || c.SessionID != "s1" {
		t.Fatalf("valid token rejected: %v %+v", err, c)
	}

	cases := map[string]string{
		"expired":       sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(-time.Minute).Unix()}),
		"no expiration": sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u1", "sid": "s1"}),
		"wrong secret":  sign(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-123"), good),
		"hs512":         sign(t, jwt.SigningMethodHS512, secret, good),
		"missing uid":   sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"sid": "s1", "exp": time.Now().Add(time.Minute).Unix()}),
		"missing sid":   sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u1", "exp": time.Now().Add(time.Minute).Unix()}),
		"alg none":      sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, good),
		"garbage":       "a.b.c",
		"empty":         "",
	}
	for name, token := range cases {
		if _, err := ParseAccessToken(secret, token); err == nil {
			t.Errorf("%s: token must be rejected", name)
		}
	}
}
