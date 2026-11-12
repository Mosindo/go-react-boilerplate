package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("middleware-unit-key-0123456789abcdef")

func sign(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims, key any) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func validClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{"uid": "u1", "oid": "o1", "sid": "s1", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix()}
}

func TestVerifierAcceptsValidToken(t *testing.T) {
	v := NewTokenVerifier(secret, nil)
	id, err := v.Verify(context.Background(), sign(t, jwt.SigningMethodHS256, validClaims(), secret))
	if err != nil || id.UserID != "u1" || id.SessionID != "s1" {
		t.Fatalf("unexpected: %+v %v", id, err)
	}
}

func TestVerifierRejectsBadTokens(t *testing.T) {
	v := NewTokenVerifier(secret, nil)
	noExp := validClaims()
	delete(noExp, "exp")
	expired := validClaims()
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	noSID := validClaims()
	delete(noSID, "sid")

	cases := map[string]string{
		"hs512":     sign(t, jwt.SigningMethodHS512, validClaims(), secret),
		"none":      sign(t, jwt.SigningMethodNone, validClaims(), jwt.UnsafeAllowNoneSignatureType),
		"wrong key": sign(t, jwt.SigningMethodHS256, validClaims(), []byte("another-key-0123456789abcdef0123456")),
		"no exp":    sign(t, jwt.SigningMethodHS256, noExp, secret),
		"expired":   sign(t, jwt.SigningMethodHS256, expired, secret),
		"no sid":    sign(t, jwt.SigningMethodHS256, noSID, secret),
		"garbage":   "a.b.c",
	}
	for name, tok := range cases {
		if _, err := v.Verify(context.Background(), tok); err == nil {
			t.Fatalf("%s token must be rejected", name)
		}
	}
}

func TestVerifierChecksSession(t *testing.T) {
	v := NewTokenVerifier(secret, func(context.Context, string, string) (bool, error) { return false, nil })
	if _, err := v.Verify(context.Background(), sign(t, jwt.SigningMethodHS256, validClaims(), secret)); err != ErrInvalidToken {
		t.Fatalf("revoked session must be rejected, got %v", err)
	}
}

func TestStillActive(t *testing.T) {
	active := true
	v := NewTokenVerifier(secret, func(context.Context, string, string) (bool, error) { return active, nil })
	id := Identity{UserID: "u1", SessionID: "s1"}
	if !v.StillActive(context.Background(), id) {
		t.Fatal("expected active session")
	}
	active = false
	if v.StillActive(context.Background(), id) {
		t.Fatal("expected revoked session")
	}
	if !NewTokenVerifier(secret, nil).StillActive(context.Background(), id) {
		t.Fatal("no checker means no revocation source")
	}
}
