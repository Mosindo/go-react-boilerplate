// Package realtime pushes server events (new message, new match, ...) to
// connected clients over WebSocket. It is in-memory and therefore scoped to a
// single API instance; scaling out requires a shared pub/sub (Redis or
// Postgres LISTEN/NOTIFY) behind the same Publish API.
package realtime

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// Event is the envelope sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

const (
	maxConnectionsPerUser = 5
	sendBuffer            = 32
	ticketTTL             = 30 * time.Second
)

type client struct {
	userID string
	send   chan Event
}

type ticket struct {
	userID  string
	expires time.Time
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string][]*client

	ticketMu sync.Mutex
	tickets  map[string]ticket
	now      func() time.Time
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string][]*client),
		tickets: make(map[string]ticket),
		now:     time.Now,
	}
}

// IssueTicket returns a single-use, short-lived credential that a client
// exchanges for a WebSocket connection. Tickets keep long-lived JWTs out of URLs and logs.
func (h *Hub) IssueTicket(userID string) (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	value := base64.RawURLEncoding.EncodeToString(raw)

	h.ticketMu.Lock()
	defer h.ticketMu.Unlock()
	now := h.now()
	for k, t := range h.tickets {
		if now.After(t.expires) {
			delete(h.tickets, k)
		}
	}
	h.tickets[value] = ticket{userID: userID, expires: now.Add(ticketTTL)}
	return value, nil
}

// redeemTicket consumes a ticket and returns its owner.
func (h *Hub) redeemTicket(value string) (string, bool) {
	h.ticketMu.Lock()
	defer h.ticketMu.Unlock()
	t, ok := h.tickets[value]
	if !ok {
		return "", false
	}
	delete(h.tickets, value)
	if h.now().After(t.expires) {
		return "", false
	}
	return t.userID, true
}

func (h *Hub) register(userID string) *client {
	c := &client{userID: userID, send: make(chan Event, sendBuffer)}
	h.mu.Lock()
	defer h.mu.Unlock()
	list := append(h.clients[userID], c)
	if len(list) > maxConnectionsPerUser {
		close(list[0].send)
		list = list[1:]
	}
	h.clients[userID] = list
	return c
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.clients[c.userID]
	for i, existing := range list {
		if existing == c {
			h.clients[c.userID] = append(list[:i], list[i+1:]...)
			close(c.send)
			break
		}
	}
	if len(h.clients[c.userID]) == 0 {
		delete(h.clients, c.userID)
	}
}

// Publish delivers an event to every open connection of userID. Slow clients
// whose buffer is full miss the event rather than blocking the caller; they
// re-sync through the REST API when they reconnect.
func (h *Hub) Publish(userID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients[userID] {
		select {
		case c.send <- event:
		default:
		}
	}
}

// DisconnectUser closes every connection of userID (account deleted, blocked, ...).
func (h *Hub) DisconnectUser(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, c := range h.clients[userID] {
		close(c.send)
	}
	delete(h.clients, userID)
}

// IsOnline reports whether the user has at least one open connection.
func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}
