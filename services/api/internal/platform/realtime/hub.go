// Package realtime fans server events out to the WebSocket connections of a
// user. The hub is in-process: with several API instances, replace Publish
// with a Postgres LISTEN/NOTIFY or Redis pub/sub relay behind the same
// Publisher interface.
package realtime

import (
	"encoding/json"
	"sync"
	"time"
)

// Event is the envelope pushed to clients.
type Event struct {
	Type    string    `json:"type"`
	At      time.Time `json:"at"`
	Payload any       `json:"payload,omitempty"`
}

// Publisher is what features depend on; it never blocks the caller.
type Publisher interface {
	Publish(userID string, eventType string, payload any)
}

// NopPublisher discards events (used in tests and tools).
type NopPublisher struct{}

func (NopPublisher) Publish(string, string, any) {}

type client struct {
	userID string
	send   chan []byte
	close  func()
}

// Hub tracks live connections per user.
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
	set, ok := h.clients[c.userID]
	if !ok {
		return
	}
	delete(set, c)
	if len(set) == 0 {
		delete(h.clients, c.userID)
	}
}

// IsOnline reports whether a user has at least one live connection.
func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

// Publish serialises the event once and queues it on every connection of the
// user. Slow consumers are disconnected rather than blocking the publisher;
// clients resynchronise over REST when they reconnect.
func (h *Hub) Publish(userID string, eventType string, payload any) {
	data, err := json.Marshal(Event{Type: eventType, At: time.Now().UTC(), Payload: payload})
	if err != nil {
		return
	}

	h.mu.RLock()
	targets := make([]*client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		select {
		case c.send <- data:
		default:
			c.close()
		}
	}
}

// DisconnectUser closes every live connection of a user (used when all of
// their sessions are revoked).
func (h *Hub) DisconnectUser(userID string) {
	h.mu.RLock()
	targets := make([]*client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		c.close()
	}
}
