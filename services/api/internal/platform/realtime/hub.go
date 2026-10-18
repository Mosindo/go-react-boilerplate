// Package realtime pushes server events to connected clients over WebSocket.
// The hub is in-memory (single API instance). To scale horizontally, back
// Publish with Postgres LISTEN/NOTIFY or Redis pub/sub; callers do not change.
package realtime

import "sync"

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type Publisher interface {
	Publish(userID string, event Event)
}

type client struct {
	send chan Event
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

func (h *Hub) register(userID string) *client {
	c := &client{send: make(chan Event, 64)}
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
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
}

// Publish never blocks: a slow client drops events and is expected to resync
// through the REST API on reconnect.
func (h *Hub) Publish(userID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- event:
		default:
		}
	}
}

// Connected reports whether the user has at least one open socket.
func (h *Hub) Connected(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

// Noop discards events; used by tests and the seed command.
type Noop struct{}

func (Noop) Publish(string, Event) {}
