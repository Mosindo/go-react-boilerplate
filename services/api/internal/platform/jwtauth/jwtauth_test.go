package jwtauth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func TestSignParseRoundTrip(t *testing.T) {
	tok, err := Sign(secret, "u1", "s1", time.Minute, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse(secret, tok)
	if err != nil || c.UserID != "u1" || c.SessionID != "s1" || c.ExpiresAt == nil {
		t.Fatalf("parse: %+v %v", c, err)
	}
}

func TestParseRejects(t *testing.T) {
	now := time.Now()
	mk := func(method jwt.SigningMethod, claims jwt.Claims, key any) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	good := Claims{UserID: "u", SessionID: "s", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute))}}
	expired := good
	expired.ExpiresAt = jwt.NewNumericDate(now.Add(-time.Minute))
	noExp := Claims{UserID: "u", SessionID: "s"}
	noSid := Claims{UserID: "u", RegisteredClaims: good.RegisteredClaims}
	noUid := Claims{SessionID: "s", RegisteredClaims: good.RegisteredClaims}

	cases := map[string]string{
		"alg none":       mk(jwt.SigningMethodNone, good, jwt.UnsafeAllowNoneSignatureType),
		"HS512":          mk(jwt.SigningMethodHS512, good, secret),
		"HS384":          mk(jwt.SigningMethodHS384, good, secret),
		"wrong secret":   mk(jwt.SigningMethodHS256, good, []byte("another-secret-another-secret-123")),
		"expired":        mk(jwt.SigningMethodHS256, expired, secret),
		"no exp":         mk(jwt.SigningMethodHS256, noExp, secret),
		"no sid":         mk(jwt.SigningMethodHS256, noSid, secret),
		"no uid":         mk(jwt.SigningMethodHS256, noUid, secret),
		"garbage":        "not.a.token",
		"empty":          "",
		"tampered claim": mk(jwt.SigningMethodHS256, good, secret) + "x",
	}
	for name, tok := range cases {
		if _, err := Parse(secret, tok); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	if _, err := Parse(secret, mk(jwt.SigningMethodHS256, good, secret)); err != nil {
		t.Errorf("control token must parse: %v", err)
	}
}
