package realtime

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHubDeliversOnlyToTheTargetUser(t *testing.T) {
	h := NewHub()
	a1, a2, b := h.register("a"), h.register("a"), h.register("b")
	h.Publish("a", Event{Type: "x"})
	for name, c := range map[string]*client{"a1": a1, "a2": a2} {
		select {
		case ev := <-c.send:
			if ev.Type != "x" {
				t.Fatalf("%s got %v", name, ev)
			}
		default:
			t.Fatalf("%s should receive the event", name)
		}
	}
	select {
	case <-b.send:
		t.Fatal("b must not receive a's events")
	default:
	}

	h.unregister("a", a1)
	h.unregister("a", a2)
	if h.Connected("a") {
		t.Fatal("a should be disconnected")
	}
	h.Publish("a", Event{Type: "y"}) // no clients: must not panic or block
}

func TestHubNeverBlocksOnSlowClient(t *testing.T) {
	h := NewHub()
	h.register("a")
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			h.Publish("a", Event{Type: "spam"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a full client buffer")
	}
}

func TestTicketValidation(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	g := NewGateway(NewHub(), secret)
	sign := func(c ticketClaims, method jwt.SigningMethod) string {
		s, _ := jwt.NewWithClaims(method, c).SignedString(secret)
		return s
	}
	good := ticketClaims{UserID: "u1", RegisteredClaims: jwt.RegisteredClaims{
		Audience: jwt.ClaimStrings{ticketAudience}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}}
	if uid, err := g.userFromTicket(sign(good, jwt.SigningMethodHS256)); err != nil || uid != "u1" {
		t.Fatalf("good ticket rejected: %v", err)
	}

	expired := good
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
	wrongAud := good
	wrongAud.Audience = jwt.ClaimStrings{"access"}
	noExp := good
	noExp.ExpiresAt = nil
	for name, tok := range map[string]string{
		"expired":   sign(expired, jwt.SigningMethodHS256),
		"wrong aud": sign(wrongAud, jwt.SigningMethodHS256),
		"no exp":    sign(noExp, jwt.SigningMethodHS256),
		"hs512":     sign(good, jwt.SigningMethodHS512),
		"garbage":   "not.a.token",
		"empty":     "",
	} {
		if _, err := g.userFromTicket(tok); err == nil {
			t.Errorf("%s ticket must be rejected", name)
		}
	}
}
