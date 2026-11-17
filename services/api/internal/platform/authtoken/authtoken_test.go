package authtoken

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func TestSignAndParse(t *testing.T) {
	token, err := Sign(secret, "user-1", "session-1", time.Minute, time.Now())
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := Parse(secret, token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != "user-1" || claims.SessionID != "session-1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	token, err := Sign(secret, "user-1", "session-1", time.Minute, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := Parse(secret, token); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestParseRejectsWrongSecretAndAlgorithm(t *testing.T) {
	token, _ := Sign(secret, "user-1", "session-1", time.Minute, time.Now())
	if _, err := Parse([]byte("another-secret-another-secret-0000"), token); err == nil {
		t.Fatal("expected wrong secret to be rejected")
	}

	claims := Claims{UserID: "user-1", SessionID: "s", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: Issuer, Subject: "user-1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	}}
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString(secret)
	if _, err := Parse(secret, hs512); err == nil {
		t.Fatal("expected HS512 token to be rejected")
	}

	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := Parse(secret, none); err == nil {
		t.Fatal("expected alg=none token to be rejected")
	}
}

func TestParseRejectsMissingExpirationAndClaims(t *testing.T) {
	noExp, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "u", SessionID: "s", RegisteredClaims: jwt.RegisteredClaims{Issuer: Issuer, Subject: "u"}}).SignedString(secret)
	if _, err := Parse(secret, noExp); err == nil {
		t.Fatal("expected token without exp to be rejected")
	}
	noSession, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "u", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: Issuer, Subject: "u", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	}}).SignedString(secret)
	if _, err := Parse(secret, noSession); err == nil {
		t.Fatal("expected token without session to be rejected")
	}
	wrongIssuer, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "u", SessionID: "s", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: "someone-else", Subject: "u", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	}}).SignedString(secret)
	if _, err := Parse(secret, wrongIssuer); err == nil {
		t.Fatal("expected token with wrong issuer to be rejected")
	}
}
