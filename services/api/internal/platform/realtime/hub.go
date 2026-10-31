// Package realtime fans events out to connected clients (WebSocket sessions).
//
// The hub is in-process: it delivers to clients connected to this API
// instance. To scale horizontally, back Publish with Postgres LISTEN/NOTIFY
// or Redis pub/sub; callers only depend on the Publisher interface.
package realtime

import (
	"encoding/json"
	"sync"
)

// Event is the envelope pushed to clients.
type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Publisher delivers an event to every live connection of a user.
type Publisher interface {
	Publish(userID string, eventType string, payload any)
}

// Client is a connection endpoint owned by one user.
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
		if _, ok := set[c]; ok {
			delete(set, c)
			close(c.Send)
		}
		if len(set) == 0 {
			delete(h.clients, c.UserID)
		}
	}
	h.mu.Unlock()
}

// Publish never blocks: a client whose buffer is full is dropped (it will
// reconnect and resync through the REST API).
func (h *Hub) Publish(userID, eventType string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	msg, err := json.Marshal(Event{Type: eventType, Data: data})
	if err != nil {
		return
	}
	h.mu.RLock()
	var slow []*Client
	for c := range h.clients[userID] {
		select {
		case c.Send <- msg:
		default:
			slow = append(slow, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range slow {
		h.Unregister(c)
	}
}

// Disconnect closes every connection of a user (e.g. after a block).
func (h *Hub) Disconnect(userID string) {
	h.mu.RLock()
	var all []*Client
	for c := range h.clients[userID] {
		all = append(all, c)
	}
	h.mu.RUnlock()
	for _, c := range all {
		h.Unregister(c)
	}
}

// Noop is a Publisher that drops everything (tests, tools).
type Noop struct{}

func (Noop) Publish(string, string, any) {}
