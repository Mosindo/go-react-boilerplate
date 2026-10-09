// Package realtime pushes server events to connected clients over WebSocket.
//
// The hub is in-memory (single API instance). To scale horizontally, publish
// through Postgres LISTEN/NOTIFY or Redis and keep the Publisher interface.
package realtime

import (
	"encoding/json"
	"log"
	"sync"
)

// Event is the frame sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher is what feature services depend on.
type Publisher interface {
	Publish(userID string, event Event)
}

// NopPublisher is used by tests and tools that do not need push delivery.
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

func (h *Hub) register(userID string) *client {
	c := &client{userID: userID, send: make(chan []byte, sendBuffer)}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*client]struct{})
	}
	h.clients[userID][c] = struct{}{}
	return c
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.userID]; ok {
		if _, ok := set[c]; ok {
			delete(set, c)
			close(c.send)
		}
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Publish delivers an event to every connection of userID. Slow consumers whose
// buffer is full are dropped; they reconnect and refetch.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("realtime: marshal event %s: %v", event.Type, err)
		return
	}
	h.mu.RLock()
	var slow []*client
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
			slow = append(slow, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range slow {
		h.unregister(c)
	}
}

// DisconnectUser closes every connection of a user (account deletion).
func (h *Hub) DisconnectUser(userID string) {
	h.mu.Lock()
	set := h.clients[userID]
	delete(h.clients, userID)
	h.mu.Unlock()
	for c := range set {
		close(c.send)
	}
}

// ConnectionCount is used by tests.
func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}
