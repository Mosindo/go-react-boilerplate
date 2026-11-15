// Package realtime delivers server events (new messages, read receipts,
// matches, notifications) to connected clients over WebSocket.
//
// The hub is in-process. For several API replicas, replace Hub.Publish with a
// broker fan-out (Postgres LISTEN/NOTIFY, Redis pub/sub); callers only depend
// on the Publisher interface.
package realtime

import (
	"encoding/json"
	"log"
	"sync"
)

// Event is the envelope pushed to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Publisher interface {
	Publish(userID string, event Event)
}

type client struct {
	userID string
	send   chan []byte
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
		if _, present := set[c]; present {
			delete(set, c)
			close(c.send)
		}
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Publish sends event to every connection of userID. Slow clients whose
// buffer is full drop the event rather than blocking the publisher; the
// client re-syncs through the REST API on reconnect.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf(`{"event":"realtime_marshal_error","error":%q}`, err.Error())
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

// Disconnect closes every connection of a user (logout, account deletion).
func (h *Hub) Disconnect(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients[userID] {
		close(c.send)
	}
	delete(h.clients, userID)
}

// ConnectionCount is used by tests.
func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}
