package realtime

import (
	"encoding/json"
	"testing"
)

func TestHubDeliversOnlyToTargetUser(t *testing.T) {
	hub := NewHub()
	alice := hub.Subscribe("alice")
	bob := hub.Subscribe("bob")
	defer hub.Unsubscribe(alice)
	defer hub.Unsubscribe(bob)

	hub.Publish("alice", "message.created", map[string]string{"id": "m1"})

	select {
	case raw := <-alice.C:
		var evt struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &evt); err != nil || evt.Type != "message.created" {
			t.Fatalf("unexpected event %s (%v)", raw, err)
		}
	default:
		t.Fatalf("alice should have received the event")
	}
	select {
	case raw := <-bob.C:
		t.Fatalf("bob must not receive alice's events, got %s", raw)
	default:
	}
}

func TestHubUnsubscribeIsIdempotentAndBounded(t *testing.T) {
	hub := NewHub()
	subs := make([]*Subscription, 0, 7)
	for i := 0; i < 7; i++ {
		subs = append(subs, hub.Subscribe("u"))
	}
	if got := hub.Connections("u"); got != 5 {
		t.Fatalf("expected per-user cap of 5, got %d", got)
	}
	for _, s := range subs {
		hub.Unsubscribe(s)
		hub.Unsubscribe(s)
	}
	if got := hub.Connections("u"); got != 0 {
		t.Fatalf("expected no connections, got %d", got)
	}
	// Publishing with a full buffer must not block.
	s := hub.Subscribe("u")
	for i := 0; i < 100; i++ {
		hub.Publish("u", "x", i)
	}
	hub.Unsubscribe(s)
}
