package middleware

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("0123456789abcdef0123456789abcdef")

func sign(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims, key any) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestParseAccessToken(t *testing.T) {
	valid := jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(time.Minute).Unix()}

	if c, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, valid, testSecret)); err != nil || c.UserID != "u1" || c.SessionID != "s1" {
		t.Fatalf("valid token rejected: %v %+v", err, c)
	}

	cases := map[string]string{
		"expired":       sign(t, jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(-time.Minute).Unix()}, testSecret),
		"no expiration": sign(t, jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u1", "sid": "s1"}, testSecret),
		"wrong secret":  sign(t, jwt.SigningMethodHS256, valid, []byte("another-secret-another-secret-123")),
		"hs512":         sign(t, jwt.SigningMethodHS512, valid, testSecret),
		"missing uid":   sign(t, jwt.SigningMethodHS256, jwt.MapClaims{"sid": "s1", "exp": time.Now().Add(time.Minute).Unix()}, testSecret),
		"missing sid":   sign(t, jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u1", "exp": time.Now().Add(time.Minute).Unix()}, testSecret),
		"garbage":       "not-a-token",
		"alg none":      sign(t, jwt.SigningMethodNone, valid, jwt.UnsafeAllowNoneSignatureType),
	}
	for name, token := range cases {
		if _, err := ParseAccessToken(testSecret, token); err == nil {
			t.Errorf("%s: token must be rejected", name)
		}
	}
}
