package tokens

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	m := NewManager([]byte("0123456789abcdef0123456789abcdef"))
	raw, err := m.SignAccess("user-1", "session-1")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := m.Parse(raw, TypeAccess)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != "user-1" || claims.SessionID != "session-1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestTicketCannotBeUsedAsAccessToken(t *testing.T) {
	m := NewManager([]byte("0123456789abcdef0123456789abcdef"))
	ticket, _ := m.SignRealtimeTicket("user-1", "session-1")
	if _, err := m.Parse(ticket, TypeAccess); err == nil {
		t.Fatal("realtime ticket must not validate as access token")
	}
}

func TestRejectsExpiredWrongSecretAndAlgNone(t *testing.T) {
	m := NewManager([]byte("0123456789abcdef0123456789abcdef"))
	m.now = func() time.Time { return time.Now().Add(-time.Hour) }
	expired, _ := m.SignAccess("u", "s")
	m.now = time.Now
	if _, err := m.Parse(expired, TypeAccess); err == nil {
		t.Fatal("expired token accepted")
	}

	other := NewManager([]byte("ffffffffffffffffffffffffffffffff"))
	foreign, _ := other.SignAccess("u", "s")
	if _, err := m.Parse(foreign, TypeAccess); err == nil {
		t.Fatal("token signed with another secret accepted")
	}

	none := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: "u", SessionID: "s", Type: TypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}})
	raw, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := m.Parse(raw, TypeAccess); err == nil {
		t.Fatal("alg=none token accepted")
	}

	noExp := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "u", SessionID: "s", Type: TypeAccess})
	rawNoExp, _ := noExp.SignedString([]byte("0123456789abcdef0123456789abcdef"))
	if _, err := m.Parse(rawNoExp, TypeAccess); err == nil {
		t.Fatal("token without expiration accepted")
	}
}
