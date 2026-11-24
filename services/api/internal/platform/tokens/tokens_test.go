package tokens

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("unit-test-signing-key-0123456789abcdef")

func TestSignAndParse(t *testing.T) {
	m := NewManager(secret)
	token, err := m.Sign("u1", "s1", AudienceAccess, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := m.Parse(token, AudienceAccess)
	if err != nil || claims.UserID != "u1" || claims.SessionID != "s1" {
		t.Fatalf("unexpected %+v %v", claims, err)
	}
	if _, err := m.Parse(token, AudienceRealtime); err == nil {
		t.Fatal("audience must be enforced")
	}
	if _, err := NewManager([]byte("another-secret-0123456789abcdefghij")).Parse(token, AudienceAccess); err == nil {
		t.Fatal("signature must be enforced")
	}
}

func TestParseRejectsExpiredAndForeignAlgorithms(t *testing.T) {
	m := NewManager(secret)
	past := time.Now().Add(-2 * time.Hour)
	m.now = func() time.Time { return past }
	expired, _ := m.Sign("u1", "s1", AudienceAccess, time.Minute)
	m.now = time.Now
	if _, err := m.Parse(expired, AudienceAccess); err == nil {
		t.Fatal("expired token must be rejected")
	}

	claims := Claims{UserID: "u1", SessionID: "s1", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: Issuer, Subject: "u1", Audience: jwt.ClaimStrings{AudienceAccess},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now()),
	}}
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := m.Parse(none, AudienceAccess); err == nil {
		t.Fatal("alg=none must be rejected")
	}
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString(secret)
	if _, err := m.Parse(hs512, AudienceAccess); err == nil {
		t.Fatal("only HS256 is accepted")
	}

	noExp := claims
	noExp.ExpiresAt = nil
	withoutExp, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, noExp).SignedString(secret)
	if _, err := m.Parse(withoutExp, AudienceAccess); err == nil {
		t.Fatal("expiration is mandatory")
	}
}
