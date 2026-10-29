// Package realtime pushes events to connected clients over WebSocket.
//
// The hub is in-memory: it serves a single API instance. To scale horizontally,
// replace Hub with an implementation of Publisher backed by Redis or Postgres
// LISTEN/NOTIFY; feature code only depends on the Publisher interface.
package realtime

import (
	"encoding/json"
	"sync"
)

// Event is the envelope sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type Publisher interface {
	Publish(userID string, event Event)
}

// NopPublisher discards events.
type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: map[string]map[*Client]struct{}{}}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[c.userID]
	if set == nil {
		set = map[*Client]struct{}{}
		h.clients[c.userID] = set
	}
	set[c] = struct{}{}
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.clients[c.userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Publish delivers an event to every open connection of userID. Slow consumers
// are dropped rather than blocking the caller.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		select {
		case c.send <- payload:
		default:
			c.close()
		}
	}
}

// Online reports whether the user has at least one open connection.
func (h *Hub) Online(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

// CloseUser drops all connections of a user (account deletion, logout).
func (h *Hub) CloseUser(userID string) {
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		c.close()
	}
}
