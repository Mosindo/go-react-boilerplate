// Package realtime fans server events out to connected WebSocket clients.
// The hub is in-memory: with several API replicas, put a shared bus
// (Postgres LISTEN/NOTIFY or Redis) behind Publisher.
package realtime

import (
	"encoding/json"
	"sync"
)

type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// Publisher is the narrow interface features depend on.
type Publisher interface {
	Publish(userID string, e Event)
}

type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}

type Client struct {
	Send chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

func NewHub() *Hub { return &Hub{clients: make(map[string]map[*Client]struct{})} }

func (h *Hub) Register(userID string) *Client {
	c := &Client{Send: make(chan []byte, 32)}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]struct{})
	}
	h.clients[userID][c] = struct{}{}
	return c
}

func (h *Hub) Unregister(userID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[userID]; ok {
		if _, ok := set[c]; ok {
			delete(set, c)
			close(c.Send)
		}
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
}

// Publish never blocks: a client whose buffer is full is dropped and will
// resynchronise through REST when it reconnects.
func (h *Hub) Publish(userID string, e Event) {
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	h.mu.RLock()
	var slow []*Client
	for c := range h.clients[userID] {
		select {
		case c.Send <- data:
		default:
			slow = append(slow, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range slow {
		h.Unregister(userID, c)
	}
}

// DisconnectUser drops every connection of a user (account deleted, blocked...).
func (h *Hub) DisconnectUser(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients[userID] {
		close(c.Send)
	}
	delete(h.clients, userID)
}
