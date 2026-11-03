package middleware

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func sign(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims, key any) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestVerifyAccessToken(t *testing.T) {
	valid := jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(time.Minute).Unix()}
	if uid, sid, ok := VerifyAccessToken(secret, sign(t, jwt.SigningMethodHS256, valid, secret)); !ok || uid != "u1" || sid != "s1" {
		t.Fatal("valid token must pass")
	}
	bad := map[string]string{
		"expired":      sign(t, jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u", "sid": "s", "exp": time.Now().Add(-time.Minute).Unix()}, secret),
		"no expiry":    sign(t, jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u", "sid": "s"}, secret),
		"wrong secret": sign(t, jwt.SigningMethodHS256, valid, []byte("another-secret-another-secret-0000")),
		"hs512":        sign(t, jwt.SigningMethodHS512, valid, secret),
		"none":         sign(t, jwt.SigningMethodNone, valid, jwt.UnsafeAllowNoneSignatureType),
		"no uid":       sign(t, jwt.SigningMethodHS256, jwt.MapClaims{"sid": "s", "exp": time.Now().Add(time.Minute).Unix()}, secret),
		"no sid":       sign(t, jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u", "exp": time.Now().Add(time.Minute).Unix()}, secret),
		"garbage":      "not.a.token",
	}
	for name, tok := range bad {
		if _, _, ok := VerifyAccessToken(secret, tok); ok {
			t.Fatalf("%s must be rejected", name)
		}
	}
}
