// Package realtime fans server events out to connected WebSocket clients.
// The hub is in-process: with several API instances, back it with Postgres
// LISTEN/NOTIFY or Redis pub/sub behind the same Publisher interface.
package realtime

import "sync"

// Event is the envelope sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher delivers an event to every live connection of a user.
type Publisher interface {
	Publish(userID string, event Event)
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*Client]struct{})}
}

// Client is one live connection. Send is buffered; slow consumers are dropped.
type Client struct {
	UserID string
	Send   chan Event
}

func NewClient(userID string) *Client {
	return &Client{UserID: userID, Send: make(chan Event, 32)}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.UserID] == nil {
		h.clients[c.UserID] = make(map[*Client]struct{})
	}
	h.clients[c.UserID][c] = struct{}{}
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.UserID]; ok {
		if _, ok := set[c]; ok {
			delete(set, c)
			close(c.Send)
		}
		if len(set) == 0 {
			delete(h.clients, c.UserID)
		}
	}
}

func (h *Hub) Publish(userID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.Send <- event:
		default: // slow consumer: it will resync through REST on reconnect
		}
	}
}

// Disconnect closes every connection of a user (used on logout/account deletion).
func (h *Hub) Disconnect(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients[userID] {
		close(c.Send)
	}
	delete(h.clients, userID)
}

// Noop discards events (tests, jobs).
type Noop struct{}

func (Noop) Publish(string, Event) {}
