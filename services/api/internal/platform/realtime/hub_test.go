package realtime

import (
	"encoding/json"
	"testing"
)

func TestHubPublishAndDisconnect(t *testing.T) {
	h := NewHub()
	a := NewClient("user-a")
	b := NewClient("user-b")
	h.Register(a)
	h.Register(b)

	h.Publish("user-a", "message.created", map[string]string{"id": "m1"})
	select {
	case raw := <-a.Send:
		var ev Event
		if err := json.Unmarshal(raw, &ev); err != nil || ev.Type != "message.created" {
			t.Fatalf("unexpected event %s (%v)", raw, err)
		}
	default:
		t.Fatal("user-a should receive the event")
	}
	select {
	case <-b.Send:
		t.Fatal("user-b must not receive user-a events")
	default:
	}

	h.Disconnect("user-a")
	select {
	case <-a.Closed():
	default:
		t.Fatal("client should be closed after disconnect")
	}
	if h.ConnectionCount("user-a") != 0 {
		t.Fatal("expected no connection left")
	}
}
