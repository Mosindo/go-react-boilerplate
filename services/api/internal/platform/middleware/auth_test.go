package middleware

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("0123456789abcdef0123456789abcdef")

func sign(t *testing.T, method jwt.SigningMethod, key any, claims AccessClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return token
}

func validClaims() AccessClaims {
	now := time.Now()
	return AccessClaims{
		UserID:    "u",
		SessionID: "s",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
}

func TestParseAccessTokenAcceptsValidHS256(t *testing.T) {
	claims, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, testSecret, validClaims()))
	if err != nil || claims.UserID != "u" || claims.SessionID != "s" {
		t.Fatalf("expected valid token, got %v %+v", err, claims)
	}
}

func TestParseAccessTokenRejectsInvalidTokens(t *testing.T) {
	expired := validClaims()
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))

	noExpiry := validClaims()
	noExpiry.ExpiresAt = nil

	missingSession := validClaims()
	missingSession.SessionID = ""

	cases := map[string]string{
		"expired":         sign(t, jwt.SigningMethodHS256, testSecret, expired),
		"no expiry":       sign(t, jwt.SigningMethodHS256, testSecret, noExpiry),
		"missing session": sign(t, jwt.SigningMethodHS256, testSecret, missingSession),
		"wrong alg":       sign(t, jwt.SigningMethodHS512, testSecret, validClaims()),
		"wrong secret":    sign(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-123"), validClaims()),
		"garbage":         "not-a-token",
	}
	for name, token := range cases {
		if _, err := ParseAccessToken(testSecret, token); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}
