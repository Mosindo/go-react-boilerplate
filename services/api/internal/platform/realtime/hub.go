// Package realtime pushes server events to connected clients over WebSocket.
// The hub is in-process: with several API replicas, back it with Postgres
// LISTEN/NOTIFY or a message broker (see README, "Scaling").
package realtime

import (
	"encoding/json"
	"sync"
)

// Event is the envelope every WebSocket frame sent by the server uses.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher delivers an event to every live connection of a user.
type Publisher interface {
	Publish(userID, eventType string, data any)
}

const (
	sendBuffer         = 32
	maxConnsPerUser    = 5
	maxOutgoingPayload = 64 << 10
)

type client struct {
	userID string
	send   chan []byte
	once   sync.Once
	done   chan struct{}
}

func (c *client) close() { c.once.Do(func() { close(c.done) }) }

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

// register adds a client, evicting the oldest-registered one if the user has too many.
func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[c.userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.clients[c.userID] = set
	}
	if len(set) >= maxConnsPerUser {
		for old := range set {
			old.close()
			delete(set, old)
			break
		}
	}
	set[c] = struct{}{}
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
	c.close()
}

// Publish never blocks: a client whose buffer is full is disconnected and must resync.
func (h *Hub) Publish(userID, eventType string, data any) {
	payload, err := json.Marshal(Event{Type: eventType, Data: data})
	if err != nil || len(payload) > maxOutgoingPayload {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
			c.close()
		}
	}
}

// ConnectionCount is exposed for tests and diagnostics.
func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}
