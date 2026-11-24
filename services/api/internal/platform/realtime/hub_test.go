package realtime

import "testing"

func TestHubPublishesOnlyToTargetUser(t *testing.T) {
	hub := NewHub()
	a := hub.Register("a")
	b := hub.Register("b")

	hub.Publish("a", Event{Type: "ping", Data: map[string]string{"x": "y"}})

	select {
	case msg := <-a.Send:
		if string(msg) != `{"type":"ping","data":{"x":"y"}}` {
			t.Fatalf("unexpected payload %s", msg)
		}
	default:
		t.Fatal("expected message for a")
	}
	select {
	case <-b.Send:
		t.Fatal("b must not receive a's events")
	default:
	}

	hub.Unregister(a)
	if _, open := <-a.Send; open {
		t.Fatal("send channel must be closed after unregister")
	}
	hub.DisconnectUser("b")
	if _, open := <-b.Send; open {
		t.Fatal("send channel must be closed after disconnect")
	}
}
