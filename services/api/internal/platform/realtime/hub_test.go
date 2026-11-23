package realtime

import (
	"encoding/json"
	"testing"
)

func TestHubPublishAndLimits(t *testing.T) {
	h := NewHub()
	clients := make([]*client, 0, MaxConnectionsPerUser)
	for i := 0; i < MaxConnectionsPerUser; i++ {
		c := &client{userID: "u1", send: make(chan []byte, 1), close: func() {}}
		if !h.register(c) {
			t.Fatalf("connection %d should be accepted", i)
		}
		clients = append(clients, c)
	}
	if h.register(&client{userID: "u1", send: make(chan []byte, 1), close: func() {}}) {
		t.Fatal("connections beyond the cap must be refused")
	}
	other := &client{userID: "u2", send: make(chan []byte, 1), close: func() {}}
	h.register(other)

	h.Publish([]string{"u1"}, Event{Type: "message.created", Data: map[string]string{"id": "m1"}})
	for _, c := range clients {
		var ev Event
		if err := json.Unmarshal(<-c.send, &ev); err != nil || ev.Type != "message.created" {
			t.Fatalf("unexpected event %v %v", ev, err)
		}
	}
	select {
	case <-other.send:
		t.Fatal("events must only reach the targeted users")
	default:
	}

	h.unregister(clients[0])
	if h.ConnectionCount("u1") != MaxConnectionsPerUser-1 {
		t.Fatal("unregister must free a slot")
	}
}
