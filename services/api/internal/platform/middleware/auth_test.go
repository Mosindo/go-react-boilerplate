package middleware

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func sign(t *testing.T, m jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(m, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestParseAccessToken(t *testing.T) {
	exp := time.Now().Add(time.Hour).Unix()
	good := sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": exp})
	claims, err := ParseAccessToken(secret, good)
	if err != nil || claims.UserID != "u1" || claims.SessionID != "s1" {
		t.Fatalf("valid token rejected: %v %+v", err, claims)
	}

	bad := map[string]string{
		"wrong secret": sign(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-0000"), jwt.MapClaims{"uid": "u", "sid": "s", "exp": exp}),
		"hs512":        sign(t, jwt.SigningMethodHS512, secret, jwt.MapClaims{"uid": "u", "sid": "s", "exp": exp}),
		"alg none":     sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, jwt.MapClaims{"uid": "u", "sid": "s", "exp": exp}),
		"no exp":       sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u", "sid": "s"}),
		"expired":      sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u", "sid": "s", "exp": time.Now().Add(-time.Minute).Unix()}),
		"no uid":       sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"sid": "s", "exp": exp}),
		"no sid":       sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u", "exp": exp}),
		"garbage":      "not.a.token",
	}
	for name, tok := range bad {
		if _, err := ParseAccessToken(secret, tok); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}
