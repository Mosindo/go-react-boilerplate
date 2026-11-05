// Package realtime pushes server events to connected clients over WebSocket.
// The hub is in-memory (single API instance). The database stays the source of
// truth: clients refetch on reconnect, so a lost event never loses data. To run
// several replicas, replace Publish with a Postgres LISTEN/NOTIFY or Redis fan-out.
package realtime

import (
	"encoding/json"
	"sync"
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

func NewHub() *Hub { return &Hub{clients: make(map[string]map[*client]struct{})} }

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.clients[c.userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Publish delivers an event to every open connection of userID. Slow
// consumers are dropped instead of blocking the caller.
func (h *Hub) Publish(userID string, ev Event) {
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
			close(c.send)
			delete(h.clients[userID], c)
		}
	}
}

// Connected reports whether the user has an open connection.
func (h *Hub) Connected(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}
