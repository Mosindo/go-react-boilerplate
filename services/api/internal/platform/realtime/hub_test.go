package realtime

import "testing"

func TestHubDeliversOnlyToTheTargetUser(t *testing.T) {
	h := NewHub()
	a1, a2, b := NewClient("a"), NewClient("a"), NewClient("b")
	h.Register(a1)
	h.Register(a2)
	h.Register(b)

	h.Publish("a", Event{Type: "x"})
	if len(a1.Send) != 1 || len(a2.Send) != 1 || len(b.Send) != 0 {
		t.Fatalf("fan-out wrong: %d %d %d", len(a1.Send), len(a2.Send), len(b.Send))
	}
}

func TestHubDropsSlowConsumersWithoutBlocking(t *testing.T) {
	h := NewHub()
	c := NewClient("a")
	h.Register(c)
	for i := 0; i < cap(c.Send)+10; i++ {
		h.Publish("a", Event{Type: "x"}) // must never block
	}
	if len(c.Send) != cap(c.Send) {
		t.Fatalf("buffer should be full, got %d", len(c.Send))
	}
}

func TestHubUnregisterAndDisconnectAreIdempotent(t *testing.T) {
	h := NewHub()
	c := NewClient("a")
	h.Register(c)
	h.Disconnect("a")
	h.Unregister(c) // after Disconnect: must not double close
	h.Publish("a", Event{Type: "x"})
	if _, ok := <-c.Send; ok {
		t.Fatal("channel should be closed")
	}
}
