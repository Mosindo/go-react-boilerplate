// Package realtime delivers server-pushed events to connected clients over WebSocket.
// The hub is in-process: with several API replicas, fan out through Postgres
// LISTEN/NOTIFY or a message broker behind the Publisher interface.
package realtime

import (
	"sync"
)

// Event is the JSON envelope pushed to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher is the dependency feature services use to push events.
type Publisher interface {
	Publish(userID string, event Event)
}

// NopPublisher discards events (tests, tools).
type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}

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

func (h *Hub) register(userID string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.clients[userID] = set
	}
	set[c] = struct{}{}
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

// Publish never blocks: a client whose buffer is full is a slow consumer and drops the event
// (it re-syncs through the REST API on reconnect).
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
