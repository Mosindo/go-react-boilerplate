package realtime

import (
	"encoding/json"
	"testing"
)

func TestHubDeliversOnlyToTargetUser(t *testing.T) {
	h := NewHub()
	a := h.Register("a")
	b := h.Register("b")
	h.Publish("a", "ping", map[string]string{"x": "1"})

	select {
	case raw := <-a.Send:
		var ev Event
		if err := json.Unmarshal(raw, &ev); err != nil || ev.Type != "ping" {
			t.Fatalf("bad event %s %v", raw, err)
		}
	default:
		t.Fatal("a should receive the event")
	}
	select {
	case <-b.Send:
		t.Fatal("b must not receive a's event")
	default:
	}
}

func TestHubDropsSlowClientsWithoutBlocking(t *testing.T) {
	h := NewHub()
	c := h.Register("a")
	for i := 0; i < 100; i++ {
		h.Publish("a", "ping", i)
	}
	// Channel was closed when the client fell behind.
	drained := 0
	for range c.Send {
		drained++
	}
	if drained == 0 || drained > 32 {
		t.Fatalf("unexpected drained count %d", drained)
	}
}
