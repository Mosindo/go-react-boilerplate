package middleware

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func sign(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestParseAccessTokenStrict(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	exp := time.Now().Add(time.Minute).Unix()
	good := sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u", "sid": "s", "exp": exp})
	if uid, sid, err := ParseAccessToken(secret, good); err != nil || uid != "u" || sid != "s" {
		t.Fatalf("valid token rejected: %v", err)
	}
	bad := map[string]string{
		"no exp":    sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u", "sid": "s"}),
		"expired":   sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u", "sid": "s", "exp": time.Now().Add(-time.Minute).Unix()}),
		"no uid":    sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"sid": "s", "exp": exp}),
		"no sid":    sign(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u", "exp": exp}),
		"wrong key": sign(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-123"), jwt.MapClaims{"uid": "u", "sid": "s", "exp": exp}),
		"hs512":     sign(t, jwt.SigningMethodHS512, secret, jwt.MapClaims{"uid": "u", "sid": "s", "exp": exp}),
		"alg none":  sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, jwt.MapClaims{"uid": "u", "sid": "s", "exp": exp}),
		"garbage":   "not.a.jwt",
	}
	for name, tok := range bad {
		if _, _, err := ParseAccessToken(secret, tok); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}
