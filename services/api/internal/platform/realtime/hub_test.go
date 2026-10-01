package realtime

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("realtime-test-secret-0123456789")

func TestTicketRoundTrip(t *testing.T) {
	ticket, err := IssueTicket(secret, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := parseTicket(secret, ticket); !ok || id != "user-1" {
		t.Fatalf("valid ticket rejected: %q %v", id, ok)
	}
	if _, ok := parseTicket([]byte("different-secret-0123456789abcd"), ticket); ok {
		t.Fatal("ticket signed with another secret must be rejected")
	}
}

func TestTicketRejectsAccessTokensAndExpired(t *testing.T) {
	// an access token (no audience) must not open a socket
	access, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": "user-1", "sid": "s", "sub": "user-1", "exp": time.Now().Add(time.Minute).Unix(),
	}).SignedString(secret)
	if _, ok := parseTicket(secret, access); ok {
		t.Fatal("access token accepted as ticket")
	}
	expired, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: "user-1", Audience: jwt.ClaimStrings{ticketAudience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Second)),
	}).SignedString(secret)
	if _, ok := parseTicket(secret, expired); ok {
		t.Fatal("expired ticket accepted")
	}
	noExp, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: "user-1", Audience: jwt.ClaimStrings{ticketAudience},
	}).SignedString(secret)
	if _, ok := parseTicket(secret, noExp); ok {
		t.Fatal("ticket without expiry accepted")
	}
}

func TestPublishToUnknownUserIsHarmless(t *testing.T) {
	NewHub().Publish("nobody", Event{Type: "x"})
	NewHub().Disconnect("nobody")
}
