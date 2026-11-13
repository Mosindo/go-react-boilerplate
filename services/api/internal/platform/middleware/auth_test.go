package middleware

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("0123456789abcdef0123456789abcdef")

func sign(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestParseAccessToken(t *testing.T) {
	valid := jwt.MapClaims{"uid": "u", "oid": "o", "sid": "s", "exp": time.Now().Add(time.Minute).Unix()}
	if _, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, testSecret, valid)); err != nil {
		t.Fatalf("expected valid token: %v", err)
	}

	noExp := jwt.MapClaims{"uid": "u", "oid": "o", "sid": "s"}
	if _, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, testSecret, noExp)); err == nil {
		t.Fatal("token without exp must be rejected")
	}

	expired := jwt.MapClaims{"uid": "u", "oid": "o", "sid": "s", "exp": time.Now().Add(-time.Minute).Unix()}
	if _, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, testSecret, expired)); err == nil {
		t.Fatal("expired token must be rejected")
	}

	if _, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS512, testSecret, valid)); err == nil {
		t.Fatal("HS512 token must be rejected")
	}

	if _, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-xx"), valid)); err == nil {
		t.Fatal("token signed with another key must be rejected")
	}

	missingSID := jwt.MapClaims{"uid": "u", "oid": "o", "exp": time.Now().Add(time.Minute).Unix()}
	if _, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, testSecret, missingSID)); err == nil {
		t.Fatal("token without session id must be rejected")
	}

	unsigned := sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid)
	if _, err := ParseAccessToken(testSecret, unsigned); err == nil {
		t.Fatal("alg=none token must be rejected")
	}
}
