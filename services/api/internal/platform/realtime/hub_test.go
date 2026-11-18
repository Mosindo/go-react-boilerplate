package realtime

import (
	"sync"
	"testing"
	"time"
)

// Concurrent publishes, slow-consumer eviction and disconnects must never
// send on a closed channel (regression test for a panic).
func TestPublishRacesWithUnregister(t *testing.T) {
	h := NewHub()
	for round := 0; round < 200; round++ {
		clients := make([]*client, 0, 3)
		for i := 0; i < 3; i++ {
			c := &client{userID: "u", send: make(chan []byte, 1), createdAt: time.Now()}
			h.register(c)
			clients = append(clients, c)
		}
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				h.Publish("u", Event{Type: "x"})
			}()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.Disconnect("u")
		}()
		wg.Wait()
		for _, c := range clients {
			h.unregister(c)
		}
	}
}

func TestConnectionsPerUserAreCapped(t *testing.T) {
	h := NewHub()
	var first *client
	for i := 0; i < MaxConnectionsPerUser+2; i++ {
		c := &client{userID: "u", send: make(chan []byte, 1), createdAt: time.Now().Add(time.Duration(i) * time.Millisecond)}
		if i == 0 {
			first = c
		}
		h.register(c)
	}
	if got := h.ConnectionCount("u"); got != MaxConnectionsPerUser {
		t.Fatalf("expected %d connections, got %d", MaxConnectionsPerUser, got)
	}
	if !first.closed {
		t.Fatal("the oldest connection must be evicted")
	}
}
