// Package realtime fans server events out to connected WebSocket clients.
// The hub is in-process; to scale horizontally put a pub/sub (Redis, NATS) behind Publish.
package realtime

import (
	"encoding/json"
	"sync"
)

const (
	maxConnsPerUser = 5
	sendBuffer      = 32
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type client struct {
	userID string
	send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

func (h *Hub) register(userID string) *client {
	cl := &client{userID: userID, send: make(chan []byte, sendBuffer)}
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.clients[userID] = set
	}
	if len(set) >= maxConnsPerUser {
		for old := range set {
			delete(set, old)
			close(old.send)
			break
		}
	}
	set[cl] = struct{}{}
	return cl
}

func (h *Hub) unregister(cl *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[cl.userID]; ok {
		if _, present := set[cl]; present {
			delete(set, cl)
			close(cl.send)
		}
		if len(set) == 0 {
			delete(h.clients, cl.userID)
		}
	}
}

// Publish delivers an event to every live connection of the user. It never blocks:
// a client whose buffer is full is dropped and will resync on reconnect.
func (h *Hub) Publish(userID, eventType string, data any) {
	payload, err := json.Marshal(Event{Type: eventType, Data: data})
	if err != nil {
		return
	}
	h.mu.RLock()
	var slow []*client
	for cl := range h.clients[userID] {
		select {
		case cl.send <- payload:
		default:
			slow = append(slow, cl)
		}
	}
	h.mu.RUnlock()
	for _, cl := range slow {
		h.unregister(cl)
	}
}

// Online reports whether the user has at least one live connection.
func (h *Hub) Online(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}
