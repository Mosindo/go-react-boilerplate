package realtime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHubPublishesOnlyToTargetUser(t *testing.T) {
	h := NewHub()
	alice := h.Register("alice")
	bob := h.Register("bob")

	h.Publish("alice", Event{Type: "message.new", Data: map[string]string{"body": "hi"}})

	select {
	case raw := <-alice.Send:
		var ev Event
		if err := json.Unmarshal(raw, &ev); err != nil || ev.Type != "message.new" {
			t.Fatalf("unexpected event %s (%v)", raw, err)
		}
	default:
		t.Fatal("alice should have received the event")
	}
	select {
	case <-bob.Send:
		t.Fatal("bob must not receive alice's events")
	default:
	}

	h.Unregister(alice)
	if h.ConnectionCount("alice") != 0 {
		t.Fatal("expected alice to be unregistered")
	}
	h.Publish("alice", Event{Type: "noop"}) // must not panic on closed client
}

func TestTicketsAreSingleUseAndExpire(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	h := NewHub()
	h.now = func() time.Time { return now }

	tk, err := h.IssueTicket("alice")
	if err != nil {
		t.Fatal(err)
	}
	if user, ok := h.RedeemTicket(tk); !ok || user != "alice" {
		t.Fatalf("expected ticket to redeem for alice, got %q %v", user, ok)
	}
	if _, ok := h.RedeemTicket(tk); ok {
		t.Fatal("ticket must be single use")
	}

	tk2, _ := h.IssueTicket("alice")
	now = now.Add(2 * time.Minute)
	if _, ok := h.RedeemTicket(tk2); ok {
		t.Fatal("expired ticket must be rejected")
	}
}
