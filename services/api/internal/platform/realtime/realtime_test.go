package realtime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHubDeliversOnlyToTheTargetUser(t *testing.T) {
	h := NewHub()
	a1, a2, b := h.register("a"), h.register("a"), h.register("b")

	h.Publish("a", Event{Type: "ping", Data: map[string]string{"k": "v"}})

	for _, c := range []*client{a1, a2} {
		select {
		case raw := <-c.send:
			var ev Event
			if err := json.Unmarshal(raw, &ev); err != nil || ev.Type != "ping" {
				t.Fatalf("bad frame %s (%v)", raw, err)
			}
		default:
			t.Fatal("user a connection did not receive the event")
		}
	}
	select {
	case raw := <-b.send:
		t.Fatalf("user b must not receive a's event: %s", raw)
	default:
	}
}

func TestHubDropsSlowConsumersAndCleansUp(t *testing.T) {
	h := NewHub()
	slow := h.register("a")
	for i := 0; i < sendBuffer+5; i++ {
		h.Publish("a", Event{Type: "x"})
	}
	if h.ConnectionCount("a") != 0 {
		t.Fatal("a consumer with a full buffer must be dropped")
	}
	// The dropped client's channel is closed once drained.
	for range slow.send {
	}
	h.Publish("nobody", Event{Type: "x"}) // no panic
}

func TestDisconnectUserClosesAllConnections(t *testing.T) {
	h := NewHub()
	c1, c2 := h.register("a"), h.register("a")
	h.DisconnectUser("a")
	for _, c := range []*client{c1, c2} {
		if _, ok := <-c.send; ok {
			t.Fatal("channel should be closed")
		}
	}
	h.unregister(c1) // must not double close
	if h.ConnectionCount("a") != 0 {
		t.Fatal("connections remain")
	}
}

func TestTicketsAreSingleUseAndExpire(t *testing.T) {
	tk := NewTickets()
	now := time.Now()
	tk.now = func() time.Time { return now }

	v, err := tk.Issue("u1")
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := tk.Redeem(v); !ok || id != "u1" {
		t.Fatal("ticket should redeem once")
	}
	if _, ok := tk.Redeem(v); ok {
		t.Fatal("ticket must be single use")
	}
	if _, ok := tk.Redeem("unknown"); ok {
		t.Fatal("unknown ticket accepted")
	}

	v, _ = tk.Issue("u2")
	now = now.Add(ticketTTL + time.Second)
	if _, ok := tk.Redeem(v); ok {
		t.Fatal("expired ticket accepted")
	}
}
