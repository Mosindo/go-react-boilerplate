// Package realtime fans out server events to the WebSocket connections of a
// user. The hub is in-memory: with several API replicas, back it with
// Postgres LISTEN/NOTIFY or Redis pub/sub behind the same Publisher interface.
package realtime

import (
	"encoding/json"
	"sync"
	"time"
)

// Event is the envelope pushed to clients.
type Event struct {
	Type string    `json:"type"`
	Data any       `json:"data,omitempty"`
	At   time.Time `json:"at"`
}

// Publisher is what feature services depend on.
type Publisher interface {
	Publish(userID string, eventType string, data any)
	Disconnect(userID string)
}

// Client is a single connection's outbound queue.
type Client struct {
	UserID string
	Send   chan []byte
	closed chan struct{}
	once   sync.Once
}

// NewClient creates a client with a bounded outbound buffer.
func NewClient(userID string) *Client {
	return &Client{UserID: userID, Send: make(chan []byte, 32), closed: make(chan struct{})}
}

// Closed is signalled when the hub drops the client.
func (c *Client) Closed() <-chan struct{} { return c.closed }

func (c *Client) close() {
	c.once.Do(func() { close(c.closed) })
}

// Hub tracks connected clients per user.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*Client]struct{})}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[c.UserID]
	if !ok {
		set = make(map[*Client]struct{})
		h.clients[c.UserID] = set
	}
	set[c] = struct{}{}
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.UserID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.UserID)
		}
	}
	c.close()
}

// Publish sends an event to every connection of userID. Slow clients whose
// buffer is full are dropped rather than blocking the publisher.
func (h *Hub) Publish(userID string, eventType string, data any) {
	payload, err := json.Marshal(Event{Type: eventType, Data: data, At: time.Now().UTC()})
	if err != nil {
		return
	}
	h.mu.RLock()
	var slow []*Client
	for c := range h.clients[userID] {
		select {
		case c.Send <- payload:
		default:
			slow = append(slow, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range slow {
		h.Unregister(c)
	}
}

// Disconnect drops every connection of a user (logout, account deletion).
func (h *Hub) Disconnect(userID string) {
	h.mu.Lock()
	set := h.clients[userID]
	delete(h.clients, userID)
	h.mu.Unlock()
	for c := range set {
		c.close()
	}
}

// ConnectionCount is used by tests and diagnostics.
func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}

// Nop is a Publisher that drops events (used in tests).
type Nop struct{}

func (Nop) Publish(string, string, any) {}
func (Nop) Disconnect(string)           {}
