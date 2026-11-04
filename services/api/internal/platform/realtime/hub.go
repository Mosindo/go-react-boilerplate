// Package realtime fans events out to the WebSocket connections of a user.
package realtime

import (
	"encoding/json"
	"sync"
)

// Event is a server-to-client push message.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher delivers an event to every live connection of a user.
type Publisher interface {
	Publish(userID string, ev Event)
	IsOnline(userID string) bool
}

// Client is one live connection. Send must never block.
type Client struct {
	Out chan []byte
}

// Hub is an in-memory Publisher. It is per-process: for multiple API instances,
// back it with Redis/NATS pub-sub behind the same interface.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

func NewHub() *Hub { return &Hub{clients: make(map[string]map[*Client]struct{})} }

// Register adds a connection and returns it.
func (h *Hub) Register(userID string) *Client {
	c := &Client{Out: make(chan []byte, 32)}
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]struct{})
	}
	h.clients[userID][c] = struct{}{}
	h.mu.Unlock()
	return c
}

// Unregister removes a connection.
func (h *Hub) Unregister(userID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.clients[userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
}

// Publish serialises ev once and enqueues it; slow consumers drop events and
// recover through the REST API.
func (h *Hub) Publish(userID string, ev Event) {
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.Out <- payload:
		default:
		}
	}
}

// IsOnline reports whether the user has at least one live connection.
func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}
