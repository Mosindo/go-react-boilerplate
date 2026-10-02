package realtime

import "testing"

func TestHubDeliversOnlyToTargetUser(t *testing.T) {
	h := NewHub()
	a := h.Register("a")
	b := h.Register("b")
	h.Publish("a", Event{Type: "ping"})
	select {
	case msg := <-a.Send:
		if string(msg) != `{"type":"ping"}` {
			t.Fatalf("unexpected payload %s", msg)
		}
	default:
		t.Fatal("a should receive the event")
	}
	select {
	case <-b.Send:
		t.Fatal("b must not receive a's event")
	default:
	}
	h.Unregister("a", a)
	h.Publish("a", Event{Type: "ping"}) // must not panic
}
