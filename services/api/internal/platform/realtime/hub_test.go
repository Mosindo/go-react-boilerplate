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

func TestOriginHosts(t *testing.T) {
	got := originHosts([]string{"http://localhost:8081", "https://app.example.com", "bare.example.org"})
	want := []string{"localhost:8081", "app.example.com", "bare.example.org"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("originHosts = %v, want %v", got, want)
		}
	}
}
