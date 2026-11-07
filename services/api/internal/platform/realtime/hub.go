// Package realtime is an in-memory fan-out hub for WebSocket clients.
// It is single-instance by design; to scale horizontally, back Publish with Postgres LISTEN/NOTIFY or Redis.
package realtime

import (
	"encoding/json"
	"sync"
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Client struct {
	UserID string
	Send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

func NewHub() *Hub { return &Hub{clients: make(map[string]map[*Client]struct{})} }

func (h *Hub) Register(userID string) *Client {
	c := &Client{UserID: userID, Send: make(chan []byte, 32)}
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]struct{})
	}
	h.clients[userID][c] = struct{}{}
	h.mu.Unlock()
	return c
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if set := h.clients[c.UserID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.UserID)
		}
	}
	h.mu.Unlock()
}

// Publish delivers an event to every live connection of the user. Slow consumers drop events
// (clients re-sync through the REST API on reconnect).
func (h *Hub) Publish(userID string, ev Event) {
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.Send <- payload:
		default:
		}
	}
}

// Publisher is the narrow interface features depend on.
type Publisher interface {
	Publish(userID string, ev Event)
}

type Noop struct{}

func (Noop) Publish(string, Event) {}
