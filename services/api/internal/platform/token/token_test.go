package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func TestSignParseRoundTrip(t *testing.T) {
	raw, err := Sign(secret, "u1", "s1", TypeAccess, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse(secret, raw, TypeAccess)
	if err != nil || c.UserID != "u1" || c.SessionID != "s1" {
		t.Fatalf("unexpected result %+v %v", c, err)
	}
}

func TestParseRejects(t *testing.T) {
	ws, _ := Sign(secret, "u1", "s1", TypeWS, time.Minute)
	if _, err := Parse(secret, ws, TypeAccess); err == nil {
		t.Fatal("ws token must not be accepted as access token")
	}
	expired, _ := Sign(secret, "u1", "s1", TypeAccess, -time.Minute)
	if _, err := Parse(secret, expired, TypeAccess); err == nil {
		t.Fatal("expired token must be rejected")
	}
	other, _ := Sign([]byte("another-secret-another-secret-123456"), "u1", "s1", TypeAccess, time.Minute)
	if _, err := Parse(secret, other, TypeAccess); err == nil {
		t.Fatal("wrong signature must be rejected")
	}
	noExp, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "u", SessionID: "s", Type: TypeAccess}).SignedString(secret)
	if _, err := Parse(secret, noExp, TypeAccess); err == nil {
		t.Fatal("token without expiry must be rejected")
	}
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: "u", SessionID: "s", Type: TypeAccess}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := Parse(secret, none, TypeAccess); err == nil {
		t.Fatal("alg none must be rejected")
	}
}
