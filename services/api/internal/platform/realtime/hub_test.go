package realtime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHubDeliversOnlyToTargetUser(t *testing.T) {
	h := NewHub()
	a1, a2, b := h.Register("a"), h.Register("a"), h.Register("b")
	h.Publish("a", Event{Type: "message", Data: "hi"})

	for _, c := range []*Client{a1, a2} {
		select {
		case raw := <-c.Send:
			var ev Event
			if err := json.Unmarshal(raw, &ev); err != nil || ev.Type != "message" {
				t.Fatalf("bad event: %s %v", raw, err)
			}
		case <-time.After(time.Second):
			t.Fatal("all connections of the target user must receive the event")
		}
	}
	select {
	case <-b.Send:
		t.Fatal("other users must not receive the event")
	default:
	}

	h.Unregister(a1)
	h.Unregister(a2)
	h.Publish("a", Event{Type: "x"}) // no clients: must not panic or block
}

func TestHubDropsEventsForSlowConsumers(t *testing.T) {
	h := NewHub()
	c := h.Register("a")
	for i := 0; i < 100; i++ {
		h.Publish("a", Event{Type: "spam"}) // never blocks even though nobody reads
	}
	if len(c.Send) != cap(c.Send) {
		t.Fatalf("buffer should be full, got %d", len(c.Send))
	}
}
