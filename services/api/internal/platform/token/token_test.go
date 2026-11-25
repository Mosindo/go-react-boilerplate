package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func TestSignAndParse(t *testing.T) {
	m := NewManager(secret, time.Minute)
	raw, err := m.Sign("user-1", "session-1", time.Now())
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	userID, sessionID, err := m.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if userID != "user-1" || sessionID != "session-1" {
		t.Fatalf("unexpected claims %q %q", userID, sessionID)
	}
}

func TestParseRejectsExpired(t *testing.T) {
	m := NewManager(secret, time.Minute)
	raw, _ := m.Sign("user-1", "session-1", time.Now().Add(-2*time.Hour))
	if _, _, err := m.Parse(raw); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	raw, _ := NewManager([]byte("another-secret-another-secret-123"), time.Minute).Sign("u", "s", time.Now())
	if _, _, err := NewManager(secret, time.Minute).Parse(raw); err == nil {
		t.Fatal("expected token signed with another secret to be rejected")
	}
}

func TestParseRejectsOtherAlgorithmsAndMissingClaims(t *testing.T) {
	m := NewManager(secret, time.Minute)

	none := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "u", "sid": "s", "exp": time.Now().Add(time.Minute).Unix()})
	rawNone, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, _, err := m.Parse(rawNone); err == nil {
		t.Fatal("expected alg=none to be rejected")
	}

	hs512 := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{"sub": "u", "sid": "s", "iss": Issuer, "aud": Audience, "exp": time.Now().Add(time.Minute).Unix()})
	raw512, _ := hs512.SignedString(secret)
	if _, _, err := m.Parse(raw512); err == nil {
		t.Fatal("expected HS512 to be rejected")
	}

	noExp := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "u", "sid": "s", "iss": Issuer, "aud": Audience})
	rawNoExp, _ := noExp.SignedString(secret)
	if _, _, err := m.Parse(rawNoExp); err == nil {
		t.Fatal("expected token without exp to be rejected")
	}

	wrongAud := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "u", "sid": "s", "iss": Issuer, "aud": "other", "exp": time.Now().Add(time.Minute).Unix()})
	rawAud, _ := wrongAud.SignedString(secret)
	if _, _, err := m.Parse(rawAud); err == nil {
		t.Fatal("expected wrong audience to be rejected")
	}

	noSid := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "u", "iss": Issuer, "aud": Audience, "exp": time.Now().Add(time.Minute).Unix()})
	rawNoSid, _ := noSid.SignedString(secret)
	if _, _, err := m.Parse(rawNoSid); err == nil {
		t.Fatal("expected missing sid to be rejected")
	}
}
