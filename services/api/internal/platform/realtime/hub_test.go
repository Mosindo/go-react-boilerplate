package realtime

import (
	"encoding/json"
	"testing"
)

func TestHubPublishDisconnectAndCap(t *testing.T) {
	h := NewHub()
	a := &client{userID: "u1", send: make(chan []byte, 4)}
	b := &client{userID: "u2", send: make(chan []byte, 4)}
	if !h.register(a) || !h.register(b) {
		t.Fatal("register should succeed")
	}

	h.Publish("u1", Event{Type: "message.new", Data: map[string]string{"body": "hi"}})
	select {
	case raw := <-a.send:
		var ev Event
		if err := json.Unmarshal(raw, &ev); err != nil || ev.Type != "message.new" {
			t.Fatalf("unexpected event %s", raw)
		}
	default:
		t.Fatal("u1 should receive its event")
	}
	select {
	case <-b.send:
		t.Fatal("u2 must not receive u1's events")
	default:
	}

	h.Disconnect("u1")
	if _, open := <-a.send; open {
		t.Fatal("disconnect must close the channel")
	}
	h.unregister(a) // must not panic after Disconnect
	if h.ConnectionCount("u1") != 0 {
		t.Fatal("u1 should have no connection")
	}

	for i := 0; i < MaxConnectionsPerUser-1; i++ {
		if !h.register(&client{userID: "u2", send: make(chan []byte, 1)}) {
			t.Fatalf("connection %d should be accepted", i)
		}
	}
	if h.register(&client{userID: "u2", send: make(chan []byte, 1)}) {
		t.Fatal("connections beyond the cap must be refused")
	}
}
