package authtoken

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func TestSignAndParse(t *testing.T) {
	raw, err := Sign(secret, "user-1", "session-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Parse(secret, raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != "user-1" || claims.SessionID != "session-1" {
		t.Fatalf("unexpected claims %+v", claims)
	}
}

func TestParseRejectsExpiredWrongSecretAndAlg(t *testing.T) {
	expired, _ := Sign(secret, "u", "s", time.Now().Add(-time.Hour))
	if _, err := Parse(secret, expired); err == nil {
		t.Fatal("expired token must be rejected")
	}

	valid, _ := Sign(secret, "u", "s", time.Now())
	if _, err := Parse([]byte("another-secret-another-secret-xx"), valid); err == nil {
		t.Fatal("wrong secret must be rejected")
	}

	none := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: "u", SessionID: "s"})
	unsafe, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := Parse(secret, unsafe); err == nil {
		t.Fatal("alg=none must be rejected")
	}

	hs512 := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{
		UserID: "u", SessionID: "s",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: Issuer, Subject: "u", Audience: jwt.ClaimStrings{Audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)), IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	})
	other, _ := hs512.SignedString(secret)
	if _, err := Parse(secret, other); err == nil {
		t.Fatal("HS512 must be rejected")
	}

	noExp := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: "u", SessionID: "s",
		RegisteredClaims: jwt.RegisteredClaims{Issuer: Issuer, Subject: "u", Audience: jwt.ClaimStrings{Audience}},
	})
	noExpRaw, _ := noExp.SignedString(secret)
	if _, err := Parse(secret, noExpRaw); err == nil {
		t.Fatal("token without exp must be rejected")
	}
}
