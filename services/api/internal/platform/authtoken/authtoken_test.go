package authtoken

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func TestAccessTokenRoundTrip(t *testing.T) {
	m := NewManager(secret)
	token, err := m.SignAccess("user-1", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := m.Parse(token, TypeAccess)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != "user-1" || claims.SessionID != "session-1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestTicketCannotBeUsedAsAccessToken(t *testing.T) {
	m := NewManager(secret)
	ticket, _ := m.SignTicket("user-1")
	if _, err := m.Parse(ticket, TypeAccess); err == nil {
		t.Fatal("ticket must not be accepted as access token")
	}
	if _, err := m.Parse(ticket, TypeTicket); err != nil {
		t.Fatalf("ticket should parse as ticket: %v", err)
	}
}

func TestExpiredAndForeignTokensRejected(t *testing.T) {
	m := NewManager(secret)
	m.now = func() time.Time { return time.Now().Add(-time.Hour) }
	old, _ := m.SignAccess("u", "s")
	m.now = time.Now
	if _, err := m.Parse(old, TypeAccess); err == nil {
		t.Fatal("expired token must be rejected")
	}

	other := NewManager([]byte("another-secret-another-secret-123"))
	foreign, _ := other.SignAccess("u", "s")
	if _, err := m.Parse(foreign, TypeAccess); err == nil {
		t.Fatal("token signed with another secret must be rejected")
	}

	noExp := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u", "sid": "s", "typ": TypeAccess})
	raw, _ := noExp.SignedString(secret)
	if _, err := m.Parse(raw, TypeAccess); err == nil {
		t.Fatal("token without expiry must be rejected")
	}

	hs512 := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{UserID: "u", SessionID: "s", Type: TypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}})
	raw, _ = hs512.SignedString(secret)
	if _, err := m.Parse(raw, TypeAccess); err == nil {
		t.Fatal("non-HS256 token must be rejected")
	}
}
