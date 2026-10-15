package realtime

import (
	"testing"
	"time"
)

func TestTicketIsSingleUse(t *testing.T) {
	h := NewHub()
	ticket, err := h.IssueTicket("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := h.redeemTicket(ticket); !ok || id != "user-1" {
		t.Fatalf("first redemption must succeed, got %q %v", id, ok)
	}
	if _, ok := h.redeemTicket(ticket); ok {
		t.Fatal("a ticket must not be redeemable twice")
	}
	if _, ok := h.redeemTicket("forged"); ok {
		t.Fatal("unknown tickets must be refused")
	}
}

func TestTicketExpires(t *testing.T) {
	h := NewHub()
	now := time.Unix(1_700_000_000, 0)
	h.now = func() time.Time { return now }
	ticket, _ := h.IssueTicket("user-1")
	now = now.Add(ticketTTL + time.Second)
	if _, ok := h.redeemTicket(ticket); ok {
		t.Fatal("an expired ticket must be refused")
	}
}

func TestPublishReachesOnlyTheTargetUser(t *testing.T) {
	h := NewHub()
	a1, a2, b := h.register("a"), h.register("a"), h.register("b")

	h.Publish("a", Event{Type: "message.new"})
	for _, c := range []*client{a1, a2} {
		select {
		case e := <-c.send:
			if e.Type != "message.new" {
				t.Fatalf("unexpected event %v", e)
			}
		default:
			t.Fatal("every connection of the user must receive the event")
		}
	}
	select {
	case <-b.send:
		t.Fatal("other users must not receive the event")
	default:
	}
}

func TestSlowClientsNeverBlockPublishers(t *testing.T) {
	h := NewHub()
	c := h.register("a")
	done := make(chan struct{})
	go func() {
		for i := 0; i < sendBuffer*3; i++ {
			h.Publish("a", Event{Type: "x"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a full client buffer")
	}
	if len(c.send) != sendBuffer {
		t.Fatalf("buffer should be full, has %d", len(c.send))
	}
}

func TestConnectionCapAndDisconnect(t *testing.T) {
	h := NewHub()
	first := h.register("a")
	for i := 0; i < maxConnectionsPerUser; i++ {
		h.register("a")
	}
	if _, open := <-first.send; open {
		t.Fatal("the oldest connection must be closed once the cap is exceeded")
	}

	h.DisconnectUser("a")
	if h.IsOnline("a") {
		t.Fatal("DisconnectUser must drop every connection")
	}
	// Unregistering after a forced disconnect must not panic (double close).
	h.unregister(first)
}
