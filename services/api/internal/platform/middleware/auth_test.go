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
		t.Fatalf("sign: %v", err)
	}
	return token
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"uid": "user-1",
		"oid": "org-1",
		"sid": "session-1",
		"exp": time.Now().Add(time.Minute).Unix(),
	}
}

func TestParseAccessTokenAcceptsValidToken(t *testing.T) {
	identity, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, testSecret, validClaims()))
	if err != nil {
		t.Fatalf("expected valid token: %v", err)
	}
	if identity.UserID != "user-1" || identity.SessionID != "session-1" || identity.OrganizationID != "org-1" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
}

func TestParseAccessTokenRejectsInvalidTokens(t *testing.T) {
	expired := validClaims()
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	noExp := validClaims()
	delete(noExp, "exp")
	noSession := validClaims()
	delete(noSession, "sid")

	cases := map[string]string{
		"expired":      sign(t, jwt.SigningMethodHS256, testSecret, expired),
		"missing exp":  sign(t, jwt.SigningMethodHS256, testSecret, noExp),
		"missing sid":  sign(t, jwt.SigningMethodHS256, testSecret, noSession),
		"wrong secret": sign(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-123"), validClaims()),
		"hs512":        sign(t, jwt.SigningMethodHS512, testSecret, validClaims()),
		"none":         sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()),
		"garbage":      "not-a-token",
	}
	for name, token := range cases {
		if _, err := ParseAccessToken(testSecret, token); err == nil {
			t.Fatalf("%s: expected rejection", name)
		}
	}
}
