package realtime

import (
	"testing"
	"time"
)

func TestPublishReachesOnlyTheTargetUser(t *testing.T) {
	h := NewHub()
	a, b := NewClient("a"), NewClient("b")
	h.Register(a)
	h.Register(b)

	h.Publish("a", Event{Type: "ping"})

	select {
	case msg := <-a.Messages():
		if string(msg) != `{"type":"ping"}` {
			t.Fatalf("unexpected payload %s", msg)
		}
	default:
		t.Fatal("user a should have received the event")
	}
	select {
	case <-b.Messages():
		t.Fatal("user b must not receive user a's event")
	default:
	}
}

func TestSlowClientIsDropped(t *testing.T) {
	h := NewHub()
	c := NewClient("a")
	h.Register(c)
	for i := 0; i < 100; i++ {
		h.Publish("a", Event{Type: "x"})
	}
	if h.Online("a") {
		t.Fatal("slow client should have been dropped")
	}
	h.Unregister(c) // must be safe after a drop
}

func TestTicketsAreSingleUseAndExpire(t *testing.T) {
	tk := NewTickets(time.Minute)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tk.now = func() time.Time { return now }

	v, err := tk.Issue("u1")
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := tk.Redeem(v); !ok || id != "u1" {
		t.Fatalf("redeem = %q %v", id, ok)
	}
	if _, ok := tk.Redeem(v); ok {
		t.Fatal("ticket must be single use")
	}

	v, _ = tk.Issue("u2")
	now = now.Add(2 * time.Minute)
	if _, ok := tk.Redeem(v); ok {
		t.Fatal("expired ticket accepted")
	}
}
