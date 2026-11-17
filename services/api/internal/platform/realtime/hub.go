// Package realtime pushes events to connected clients over WebSocket.
package realtime

import (
	"encoding/json"
	"sync"
	"time"
)

// Event is the envelope sent to clients.
type Event struct {
	Type string    `json:"type"`
	Data any       `json:"data"`
	At   time.Time `json:"at"`
}

// Publisher is what features depend on. The in-memory Hub serves a single API
// instance; a multi-instance deployment can implement it on top of Postgres
// LISTEN/NOTIFY or Redis pub/sub without changing feature code.
type Publisher interface {
	Publish(userID string, eventType string, data any)
}

type client struct {
	send chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

func (h *Hub) register(userID string) *client {
	c := &client{send: make(chan []byte, 64)}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*client]struct{})
	}
	h.clients[userID][c] = struct{}{}
	return c
}

func (h *Hub) unregister(userID string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[userID]; ok {
		if _, ok := set[c]; ok {
			delete(set, c)
			close(c.send)
		}
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
}

// Publish delivers an event to every connection of userID. Slow clients drop events
// rather than blocking the publisher; clients resync over REST on reconnect.
func (h *Hub) Publish(userID string, eventType string, data any) {
	payload, err := json.Marshal(Event{Type: eventType, Data: data, At: time.Now().UTC()})
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
		}
	}
}

// Disconnect closes every connection of userID (used on account deletion).
func (h *Hub) Disconnect(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients[userID] {
		close(c.send)
	}
	delete(h.clients, userID)
}

// ConnectionCount is used by tests and health diagnostics.
func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}
