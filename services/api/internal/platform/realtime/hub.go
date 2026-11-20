// Package realtime pushes server events to connected clients over WebSocket.
//
// The hub lives in-process: with several API replicas, Publish must be fanned
// out through a shared bus (for example PostgreSQL LISTEN/NOTIFY); the
// Publisher interface is the seam for that evolution.
package realtime

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Event is the envelope sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher is what features depend on to emit events.
type Publisher interface {
	Publish(userID string, event Event)
}

type client struct {
	userID    string
	sessionID string
	send      chan []byte
	once      sync.Once
	closed    chan struct{}
}

func (c *client) close() {
	c.once.Do(func() { close(c.closed) })
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

const (
	sendBuffer   = 32
	maxPerUser   = 5
	writeTimeout = 10 * time.Second
)

func (h *Hub) register(userID, sessionID string) *client {
	c := &client{userID: userID, sessionID: sessionID, send: make(chan []byte, sendBuffer), closed: make(chan struct{})}
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.clients[userID] = set
	}
	// Bound connections per user: drop the oldest arbitrary connection.
	if len(set) >= maxPerUser {
		for existing := range set {
			existing.close()
			delete(set, existing)
			break
		}
	}
	set[c] = struct{}{}
	return c
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.userID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
	c.close()
}

// Publish delivers an event to every connection of a user. Slow consumers
// whose buffer is full are disconnected rather than blocking the caller.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf(`{"event":"realtime_marshal_error","type":%q,"error":%q}`, event.Type, err.Error())
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
			c.close()
		}
	}
}

// DisconnectUser closes every connection of a user (logout, account deletion).
func (h *Hub) DisconnectUser(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients[userID] {
		c.close()
	}
	delete(h.clients, userID)
}

// DisconnectSession closes the connections opened with one session (logout).
func (h *Hub) DisconnectSession(sessionID string) {
	h.disconnectWhere(func(c *client) bool { return c.sessionID == sessionID })
}

// DisconnectUserExcept closes a user's connections except those of keepSessionID
// (password change signs out the other devices).
func (h *Hub) DisconnectUserExcept(userID, keepSessionID string) {
	h.disconnectWhere(func(c *client) bool { return c.userID == userID && c.sessionID != keepSessionID })
}

func (h *Hub) disconnectWhere(match func(*client) bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for userID, set := range h.clients {
		for c := range set {
			if match(c) {
				c.close()
				delete(set, c)
			}
		}
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
}

// ConnectionCount is exposed for tests and diagnostics.
func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}
