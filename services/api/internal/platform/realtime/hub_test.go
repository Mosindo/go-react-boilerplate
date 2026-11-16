package realtime

import (
	"testing"
	"time"
)

func TestTicketsAreSingleUseAndExpire(t *testing.T) {
	h := NewHub()
	now := time.Unix(0, 0)
	h.now = func() time.Time { return now }

	tk, err := h.IssueTicket("u1")
	if err != nil {
		t.Fatal(err)
	}
	if uid, ok := h.RedeemTicket(tk); !ok || uid != "u1" {
		t.Fatalf("expected ticket to redeem for u1, got %q %v", uid, ok)
	}
	if _, ok := h.RedeemTicket(tk); ok {
		t.Fatal("ticket must be single-use")
	}

	tk2, _ := h.IssueTicket("u2")
	now = now.Add(time.Minute)
	if _, ok := h.RedeemTicket(tk2); ok {
		t.Fatal("expired ticket must be rejected")
	}
}

func TestPublishDeliversToUserConnections(t *testing.T) {
	h := NewHub()
	a := &client{userID: "a", send: make(chan []byte, 1)}
	b := &client{userID: "b", send: make(chan []byte, 1)}
	h.register(a)
	h.register(b)

	h.Publish("a", Event{Type: "ping", Data: 1})
	select {
	case msg := <-a.send:
		if string(msg) != `{"type":"ping","data":1}` {
			t.Fatalf("unexpected payload %s", msg)
		}
	default:
		t.Fatal("expected event for a")
	}
	if len(b.send) != 0 {
		t.Fatal("b must not receive a's event")
	}

	// Fill the buffer: the slow client gets disconnected rather than blocking.
	h.Publish("a", Event{Type: "x"})
	h.Publish("a", Event{Type: "y"})
	if h.ConnectionCount("a") != 0 {
		t.Fatal("expected slow client to be dropped")
	}
	h.unregister(b)
	if h.ConnectionCount("b") != 0 {
		t.Fatal("expected b to be unregistered")
	}
}
