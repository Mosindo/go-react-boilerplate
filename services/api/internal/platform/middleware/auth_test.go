package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type fakeSessions struct{ active bool }

func (f fakeSessions) IsSessionActive(context.Context, string, string) (bool, error) {
	return f.active, nil
}

var secret = []byte("0123456789abcdef0123456789abcdef")

func sign(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{"uid": "u1", "oid": "o1", "sid": "s1", "exp": time.Now().Add(time.Minute).Unix()}
}

func TestAuthenticateAcceptsValidToken(t *testing.T) {
	a := NewAuthenticator(secret, fakeSessions{active: true})
	id, err := a.Authenticate(context.Background(), sign(t, jwt.SigningMethodHS256, validClaims()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.UserID != "u1" || id.SessionID != "s1" {
		t.Fatalf("unexpected identity %#v", id)
	}
}

func TestAuthenticateRejectsInvalidTokens(t *testing.T) {
	a := NewAuthenticator(secret, fakeSessions{active: true})

	noExp := validClaims()
	delete(noExp, "exp")
	expired := validClaims()
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	missingSID := validClaims()
	delete(missingSID, "sid")

	cases := map[string]string{
		"no exp":      sign(t, jwt.SigningMethodHS256, noExp),
		"expired":     sign(t, jwt.SigningMethodHS256, expired),
		"missing sid": sign(t, jwt.SigningMethodHS256, missingSID),
		"wrong alg":   sign(t, jwt.SigningMethodHS512, validClaims()),
		"garbage":     "not-a-token",
		"empty":       "",
		"alg none":    "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1aWQiOiJ1MSIsIm9pZCI6Im8xIiwic2lkIjoiczEifQ.",
	}
	for name, token := range cases {
		if _, err := a.Authenticate(context.Background(), token); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%s: expected ErrInvalidToken, got %v", name, err)
		}
	}
}

func TestAuthenticateRejectsRevokedSession(t *testing.T) {
	a := NewAuthenticator(secret, fakeSessions{active: false})
	if _, err := a.Authenticate(context.Background(), sign(t, jwt.SigningMethodHS256, validClaims())); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected revoked session to be rejected, got %v", err)
	}
}
