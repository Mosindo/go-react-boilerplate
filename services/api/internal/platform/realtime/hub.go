// Package realtime fans server events out to connected WebSocket clients.
// The hub is in-process: it serves a single API instance. To scale horizontally, back Publish with a
// shared bus (Postgres LISTEN/NOTIFY or Redis pub/sub) without changing callers.
package realtime

import (
	"encoding/json"
	"sync"
)

// Event is the wire envelope pushed to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher is what features depend on.
type Publisher interface {
	Publish(userID string, event Event)
}

// NopPublisher discards events (tests, CLI tools).
type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}

const sendBuffer = 64

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

func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[c.userID]
	if !ok {
		set = make(map[*client]struct{})
		h.clients[c.userID] = set
	}
	set[c] = struct{}{}
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.userID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Publish delivers the event to every live connection of the user. Slow consumers are dropped
// (their write loop then closes the socket) instead of blocking the publisher.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	var slow []*client
	h.mu.RLock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
			slow = append(slow, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range slow {
		h.drop(c)
	}
}

// drop unregisters and closes the send channel under the write lock, so no Publish can send on it afterwards.
func (h *Hub) drop(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[c.userID]
	if !ok {
		return
	}
	if _, registered := set[c]; !registered {
		return
	}
	delete(set, c)
	if len(set) == 0 {
		delete(h.clients, c.userID)
	}
	close(c.send)
}

// Connected reports whether the user has at least one live connection (used by tests).
func (h *Hub) Connected(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}
