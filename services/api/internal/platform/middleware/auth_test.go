package middleware

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("0123456789abcdef0123456789abcdef")

func sign(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestParseAccessTokenValid(t *testing.T) {
	tok := sign(t, jwt.SigningMethodHS256, testSecret, jwt.MapClaims{
		"uid": "u1", "sid": "s1", "exp": time.Now().Add(time.Minute).Unix(),
	})
	claims, err := ParseAccessToken(testSecret, tok)
	if err != nil {
		t.Fatalf("expected valid token: %v", err)
	}
	if claims.UserID != "u1" || claims.SessionID != "s1" {
		t.Fatalf("unexpected claims %+v", claims)
	}
}

func TestParseAccessTokenRejections(t *testing.T) {
	cases := map[string]string{
		"missing exp": sign(t, jwt.SigningMethodHS256, testSecret, jwt.MapClaims{"uid": "u1", "sid": "s1"}),
		"expired":     sign(t, jwt.SigningMethodHS256, testSecret, jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(-time.Minute).Unix()}),
		"wrong alg":   sign(t, jwt.SigningMethodHS512, testSecret, jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(time.Minute).Unix()}),
		"wrong key":   sign(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-xx"), jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(time.Minute).Unix()}),
		"no uid":      sign(t, jwt.SigningMethodHS256, testSecret, jwt.MapClaims{"sid": "s1", "exp": time.Now().Add(time.Minute).Unix()}),
		"no sid":      sign(t, jwt.SigningMethodHS256, testSecret, jwt.MapClaims{"uid": "u1", "exp": time.Now().Add(time.Minute).Unix()}),
		"none alg":    sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(time.Minute).Unix()}),
		"garbage":     "not-a-token",
	}
	for name, tok := range cases {
		if _, err := ParseAccessToken(testSecret, tok); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}
