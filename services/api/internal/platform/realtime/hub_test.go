package realtime

import "testing"

func TestHubPublishesOnlyToTargetUser(t *testing.T) {
	hub := NewHub()
	a := hub.Register("a", "s1")
	b := hub.Register("b", "s2")

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

func TestHubDisconnectSessionOnlyClosesThatSession(t *testing.T) {
	hub := NewHub()
	phone := hub.Register("u", "phone")
	laptop := hub.Register("u", "laptop")
	hub.DisconnectSession("u", "phone")
	if _, open := <-phone.Send; open {
		t.Fatal("phone connection must be closed")
	}
	hub.Publish("u", Event{Type: "ping"})
	if _, open := <-laptop.Send; !open {
		t.Fatal("other session must stay connected")
	}
}
