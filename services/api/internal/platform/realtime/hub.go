// Package realtime fans server events out to connected clients over WebSocket.
//
// Design: one in-memory Hub per API process. Clients authenticate with a one-time short-lived
// ticket (POST /ws/ticket, JWT-bound) so long-lived access tokens never appear in URLs.
// Events are best-effort: clients must also fetch state over REST on (re)connect.
// To scale horizontally, back Publish with Redis pub/sub or Postgres LISTEN/NOTIFY.
package realtime

import (
	"encoding/json"
	"sync"
)

// Event is the envelope sent to clients: {"type":"message.new","data":{...}}.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher is what feature services depend on.
type Publisher interface {
	Publish(userID string, event Event)
}

// NopPublisher is used in tests or when realtime is disabled.
type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}

const sendBuffer = 32

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

func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[c.userID]
	if !ok {
		set = make(map[*client]struct{})
		h.clients[c.userID] = set
	}
	set[c] = struct{}{}
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.userID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Publish delivers to every connection of userID. Slow consumers are dropped, never awaited.
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
			// buffer full: the client will resync over REST after reconnecting
		}
	}
}

// Online reports whether the user has at least one open connection.
func (h *Hub) Online(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}
