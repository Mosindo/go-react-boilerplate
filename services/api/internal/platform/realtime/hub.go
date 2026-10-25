// Package realtime is the in-process WebSocket fan-out used to push events to connected users.
// It is intentionally single-node: running several API replicas requires a shared bus (for example
// PostgreSQL LISTEN/NOTIFY) behind the Publisher interface.
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

// Publisher is what features depend on to push events.
type Publisher interface {
	Publish(userID string, evt Event)
	Disconnect(userID string)
}

// Client is one connected socket.
type Client struct {
	UserID string
	send   chan []byte
	closed chan struct{}
	once   sync.Once
}

const sendBuffer = 64

func NewClient(userID string) *Client {
	return &Client{UserID: userID, send: make(chan []byte, sendBuffer), closed: make(chan struct{})}
}

// Outbound returns the channel the socket writer drains.
func (c *Client) Outbound() <-chan []byte { return c.send }

// Done is closed when the hub (or the connection) closes the client.
func (c *Client) Done() <-chan struct{} { return c.closed }

// Close is idempotent.
func (c *Client) Close() { c.once.Do(func() { close(c.closed) }) }

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
	set := h.clients[c.UserID]
	if set == nil {
		set = make(map[*Client]struct{})
		h.clients[c.UserID] = set
	}
	set[c] = struct{}{}
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
	c.Close()
}

// Publish delivers the event to every socket of the user. Slow consumers are dropped, never blocked on.
func (h *Hub) Publish(userID string, evt Event) {
	payload, err := json.Marshal(evt)
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
			c.Close()
		}
	}
}

// Disconnect closes every socket of the user (account deleted, password reset...).
func (h *Hub) Disconnect(userID string) {
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		c.Close()
	}
}

// Noop is a Publisher that drops everything (tests, tools).
type Noop struct{}

func (Noop) Publish(string, Event) {}
func (Noop) Disconnect(string)     {}
