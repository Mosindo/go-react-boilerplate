package realtime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTicketsAreSingleUseAndExpire(t *testing.T) {
	h := NewHub()
	now := time.Unix(1_700_000_000, 0)
	h.now = func() time.Time { return now }

	tk, err := h.IssueTicket("u1")
	if err != nil {
		t.Fatal(err)
	}
	if uid, ok := h.RedeemTicket(tk); !ok || uid != "u1" {
		t.Fatal("ticket should redeem once")
	}
	if _, ok := h.RedeemTicket(tk); ok {
		t.Fatal("ticket must be single use")
	}

	tk2, _ := h.IssueTicket("u1")
	now = now.Add(time.Minute)
	if _, ok := h.RedeemTicket(tk2); ok {
		t.Fatal("expired ticket must be rejected")
	}
}

func TestPublishOnlyReachesTargetUser(t *testing.T) {
	h := NewHub()
	a := h.Register("a")
	b := h.Register("b")
	h.Publish("a", Event{Type: "ping", Payload: map[string]string{"x": "y"}})

	select {
	case msg := <-a.Messages():
		var ev Event
		if err := json.Unmarshal(msg, &ev); err != nil || ev.Type != "ping" {
			t.Fatalf("unexpected event %s", msg)
		}
	default:
		t.Fatal("a should have received the event")
	}
	select {
	case <-b.Messages():
		t.Fatal("b must not receive a's events")
	default:
	}

	h.Unregister(a)
	if h.ConnectedCount("a") != 0 {
		t.Fatal("a should be unregistered")
	}
	h.Unregister(a) // idempotent
	h.Disconnect("b")
	if _, open := <-b.Messages(); open {
		t.Fatal("b channel should be closed")
	}
}
