// Package realtime pushes server events to connected clients over WebSocket.
//
// The hub is in-memory, which is correct for a single API instance. To scale
// horizontally, implement Publisher on top of Redis pub/sub or Postgres
// LISTEN/NOTIFY; callers only depend on the Publisher interface.
package realtime

import (
	"encoding/json"
	"sync"
)

// Event is the JSON envelope sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher delivers an event to every live connection of a user.
type Publisher interface {
	Publish(userID string, e Event)
}

// NopPublisher discards events (tests, tools).
type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}

type client struct {
	send chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub { return &Hub{clients: map[string]map[*client]struct{}{}} }

func (h *Hub) add(userID string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = map[*client]struct{}{}
	}
	h.clients[userID][c] = struct{}{}
}

func (h *Hub) remove(userID string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.clients[userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
}

// Publish never blocks: a client whose buffer is full is dropped and will
// resynchronise through the REST API when it reconnects.
func (h *Hub) Publish(userID string, e Event) {
	payload, err := json.Marshal(e)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
			close(c.send)
			delete(h.clients[userID], c)
		}
	}
}

// Online reports whether the user has at least one live connection.
func (h *Hub) Online(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}
