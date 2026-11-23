// Package realtime pushes server events (new messages, read receipts, matches,
// notifications) to connected clients over WebSocket.
//
// The hub is in-process: with several API replicas, back Publish with a shared
// bus (for example PostgreSQL LISTEN/NOTIFY or Redis) behind the same API.
package realtime

import (
	"encoding/json"
	"sync"
)

// Event is the envelope sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher is what features depend on.
type Publisher interface {
	Publish(userIDs []string, event Event)
}

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

// Publish delivers event to every connection of the given users. Slow
// consumers whose buffer is full are disconnected rather than blocking.
func (h *Hub) Publish(userIDs []string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, userID := range userIDs {
		for c := range h.clients[userID] {
			select {
			case c.send <- payload:
			default:
				go c.close()
			}
		}
	}
}

// Disconnect closes every connection of a user (logout everywhere, deletion).
func (h *Hub) Disconnect(userID string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		go c.close()
	}
}

// ConnectionCount is used by tests and diagnostics.
func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}
