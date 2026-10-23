package middleware

import (
	"context"
	"errors"
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

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{"uid": "u1", "sid": "s1", "iss": TokenIssuer, "exp": time.Now().Add(time.Minute).Unix()}
}

func TestParseAccessToken(t *testing.T) {
	ctx := context.Background()
	ok := func(context.Context, string, string) error { return nil }

	id, err := ParseAccessToken(ctx, testSecret, sign(t, jwt.SigningMethodHS256, testSecret, validClaims()), ok)
	if err != nil || id.UserID != "u1" || id.SessionID != "s1" {
		t.Fatalf("valid token rejected: %v %+v", err, id)
	}

	noExp := validClaims()
	delete(noExp, "exp")
	expired := validClaims()
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	wrongIss := validClaims()
	wrongIss["iss"] = "someone-else"
	noUID := validClaims()
	delete(noUID, "uid")

	cases := map[string]string{
		"no expiry":   sign(t, jwt.SigningMethodHS256, testSecret, noExp),
		"expired":     sign(t, jwt.SigningMethodHS256, testSecret, expired),
		"wrong iss":   sign(t, jwt.SigningMethodHS256, testSecret, wrongIss),
		"no uid":      sign(t, jwt.SigningMethodHS256, testSecret, noUID),
		"wrong key":   sign(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-123"), validClaims()),
		"hs512":       sign(t, jwt.SigningMethodHS512, testSecret, validClaims()),
		"alg none":    sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()),
		"garbage":     "not.a.jwt",
		"empty parts": "..",
	}
	for name, tok := range cases {
		if _, err := ParseAccessToken(ctx, testSecret, tok, ok); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%s: expected ErrInvalidToken, got %v", name, err)
		}
	}

	suspended := func(context.Context, string, string) error { return ErrAccountSuspended }
	if _, err := ParseAccessToken(ctx, testSecret, sign(t, jwt.SigningMethodHS256, testSecret, validClaims()), suspended); !errors.Is(err, ErrAccountSuspended) {
		t.Fatalf("expected suspended, got %v", err)
	}
}
