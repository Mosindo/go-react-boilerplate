package authtoken

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func TestSignAndParseAccessToken(t *testing.T) {
	token, err := Sign(secret, TypeAccess, "user-1", "session-1", AccessTTL, time.Now())
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := Parse(secret, token, TypeAccess)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != "user-1" || claims.SessionID != "session-1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestParseRejectsWrongType(t *testing.T) {
	token, _ := Sign(secret, TypeRealtime, "user-1", "session-1", RealtimeTTL, time.Now())
	if _, err := Parse(secret, token, TypeAccess); err == nil {
		t.Fatal("realtime ticket must not be accepted as access token")
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	token, _ := Sign(secret, TypeAccess, "user-1", "session-1", time.Minute, time.Now().Add(-2*time.Hour))
	if _, err := Parse(secret, token, TypeAccess); err == nil {
		t.Fatal("expired token must be rejected")
	}
}

func TestParseRejectsWrongSecretAndAlgorithms(t *testing.T) {
	token, _ := Sign(secret, TypeAccess, "user-1", "session-1", AccessTTL, time.Now())
	if _, err := Parse([]byte("another-secret-another-secret-123"), token, TypeAccess); err == nil {
		t.Fatal("token signed with another secret must be rejected")
	}

	claims := Claims{UserID: "user-1", SessionID: "s", Type: TypeAccess, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: Issuer, Subject: "user-1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString(secret)
	if _, err := Parse(secret, hs512, TypeAccess); err == nil {
		t.Fatal("HS512 token must be rejected")
	}
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := Parse(secret, none, TypeAccess); err == nil {
		t.Fatal("alg=none token must be rejected")
	}
}

func TestParseRequiresExpiration(t *testing.T) {
	claims := Claims{UserID: "user-1", SessionID: "s", Type: TypeAccess, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: Issuer, Subject: "user-1",
	}}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if _, err := Parse(secret, token, TypeAccess); err == nil {
		t.Fatal("token without exp must be rejected")
	}
}
