// Package realtime is an in-process WebSocket fan-out hub keyed by user id.
// It is single-instance by design; swap Publisher's implementation (e.g. Redis pub/sub) to scale out.
package realtime

import (
	"encoding/json"
	"sync"
)

// Publisher delivers a server event to every live connection of a user.
type Publisher interface {
	Publish(userID, eventType string, data any)
}

type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Client is one live connection; Send must never block.
type Client struct {
	UserID string
	Send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

func NewHub() *Hub { return &Hub{clients: make(map[string]map[*Client]struct{})} }

const maxConnectionsPerUser = 5

func (h *Hub) Register(c *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[c.UserID]
	if set == nil {
		set = make(map[*Client]struct{})
		h.clients[c.UserID] = set
	}
	if len(set) >= maxConnectionsPerUser {
		return false
	}
	set[c] = struct{}{}
	return true
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.clients[c.UserID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.UserID)
		}
	}
}

func (h *Hub) Publish(userID, eventType string, data any) {
	payload, err := json.Marshal(map[string]any{"type": eventType, "data": data})
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.Send <- payload:
		default: // slow consumer: drop; the client refetches on reconnect
		}
	}
}

// Nop is a Publisher that drops everything (used when realtime is not wired, e.g. unit tests).
type Nop struct{}

func (Nop) Publish(string, string, any) {}
