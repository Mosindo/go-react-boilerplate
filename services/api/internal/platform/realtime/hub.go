// Package realtime pushes events (new messages, read receipts, matches,
// notifications) to connected clients over WebSocket.
//
// The hub is in-process: it fans out to every connection of a user held by
// this API instance. Running several instances requires a shared bus (for
// example Postgres LISTEN/NOTIFY or Redis pub/sub) feeding Publish; the
// Publisher interface keeps features independent from that choice.
package realtime

import (
	"encoding/json"
	"log"
	"sync"
)

// Event is the envelope sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher is what features depend on.
type Publisher interface {
	Publish(userID string, event Event)
}

// Nop discards events (used when realtime is not wired, e.g. unit tests).
type Nop struct{}

func (Nop) Publish(string, Event) {}

type client struct {
	userID string
	send   chan []byte
	close  func()
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[c.userID]
	if !ok {
		set = make(map[*client]struct{})
		h.clients[c.userID] = set
	}
	set[c] = struct{}{}
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.userID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Publish delivers event to every connection of userID. Slow consumers whose
// buffer is full are disconnected rather than blocking the publisher; the
// client reconnects and refetches state.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf(`{"event":"realtime_marshal_error","type":%q}`, event.Type)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
			go c.close()
		}
	}
}

// ConnectionCount returns the number of live connections for userID.
func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}
