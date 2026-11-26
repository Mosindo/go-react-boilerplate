package realtime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHubPublishesToEveryConnectionOfAUser(t *testing.T) {
	hub := NewHub()
	a1 := &client{userID: "a", send: make(chan []byte, 1)}
	a2 := &client{userID: "a", send: make(chan []byte, 1)}
	b := &client{userID: "b", send: make(chan []byte, 1)}
	for _, c := range []*client{a1, a2, b} {
		hub.register(c)
	}

	hub.Publish("a", Event{Type: "message.created", Data: map[string]string{"body": "hi"}})

	for _, c := range []*client{a1, a2} {
		select {
		case raw := <-c.send:
			var ev Event
			if err := json.Unmarshal(raw, &ev); err != nil || ev.Type != "message.created" {
				t.Fatalf("unexpected payload %s", raw)
			}
		default:
			t.Fatal("every connection of the user should receive the event")
		}
	}
	select {
	case <-b.send:
		t.Fatal("other users must not receive the event")
	default:
	}

	// A full buffer never blocks the publisher.
	hub.Publish("b", Event{Type: "x"})
	hub.Publish("b", Event{Type: "y"})
}

func TestHubDisconnectClosesConnections(t *testing.T) {
	hub := NewHub()
	c := &client{userID: "a", send: make(chan []byte, 1)}
	hub.register(c)
	hub.Disconnect("a")
	if _, open := <-c.send; open {
		t.Fatal("send channel should be closed")
	}
	hub.unregister(c) // must not panic on an already closed client
	hub.Publish("a", Event{Type: "ignored"})
}

func TestTicketsAreSingleUseAndExpire(t *testing.T) {
	tickets := NewTickets(50 * time.Millisecond)
	value, err := tickets.Issue("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if userID, ok := tickets.Redeem(value); !ok || userID != "user-1" {
		t.Fatal("ticket should be redeemable once")
	}
	if _, ok := tickets.Redeem(value); ok {
		t.Fatal("ticket must be single use")
	}

	expiring, _ := tickets.Issue("user-2")
	time.Sleep(80 * time.Millisecond)
	if _, ok := tickets.Redeem(expiring); ok {
		t.Fatal("expired ticket must be rejected")
	}
	if _, ok := tickets.Redeem("forged"); ok {
		t.Fatal("unknown ticket must be rejected")
	}
}
