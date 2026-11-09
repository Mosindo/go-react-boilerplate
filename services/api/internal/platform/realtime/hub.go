// Package realtime delivers server events to connected clients over WebSocket.
// The hub is in-process: with several API replicas, back it with Postgres
// LISTEN/NOTIFY or Redis pub/sub (the Publisher interface stays the same).
package realtime

import (
	"encoding/json"
	"sync"
)

// Event is the JSON envelope sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher is what feature services depend on.
type Publisher interface {
	Publish(userID string, event Event)
}

type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}

type client struct {
	send chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: map[string]map[*client]struct{}{}}
}

func (h *Hub) register(userID string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = map[*client]struct{}{}
	}
	h.clients[userID][c] = struct{}{}
}

func (h *Hub) unregister(userID string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.clients[userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
}

// Publish sends the event to all connections of the user. Slow consumers drop events
// (clients also re-sync through the REST API, so drops are safe).
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
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
