package realtime

import "testing"

func TestHubPublishSubscribe(t *testing.T) {
	hub := NewHub()
	events, unsubscribe := hub.Subscribe("u1")
	hub.Publish("u2", Event{Type: "ignored"})
	hub.Publish("u1", Event{Type: "message.new"})

	got := <-events
	if got.Type != "message.new" {
		t.Fatalf("unexpected event %q", got.Type)
	}
	if hub.Connections("u1") != 1 {
		t.Fatal("expected one connection")
	}
	unsubscribe()
	unsubscribe() // idempotent
	if hub.Connections("u1") != 0 {
		t.Fatal("expected no connections after unsubscribe")
	}
	hub.Publish("u1", Event{Type: "after-close"}) // must not panic
}

func TestHubDropsWhenSubscriberIsSlow(t *testing.T) {
	hub := NewHub()
	_, unsubscribe := hub.Subscribe("u1")
	defer unsubscribe()
	for i := 0; i < subscriberBuffer*2; i++ {
		hub.Publish("u1", Event{Type: "spam"}) // must never block
	}
}
