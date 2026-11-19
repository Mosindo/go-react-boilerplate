// Package realtime fans out server events to connected clients over WebSocket.
//
// The hub lives in process memory: it is the delivery channel for a single API
// instance. Every event is also persisted (messages, notifications) so clients
// recover state over REST after reconnecting. Swapping Publish for a shared bus
// (e.g. Postgres LISTEN/NOTIFY or Redis) is enough to scale horizontally.
package realtime

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"
)

// Event is the envelope pushed to clients.
type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// Publisher is what features depend on to emit events.
type Publisher interface {
	Publish(userID string, event Event)
}

const clientBuffer = 32

type Client struct {
	userID string
	send   chan []byte
}

func (c *Client) Messages() <-chan []byte { return c.send }

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}

	ticketsMu sync.Mutex
	tickets   map[string]ticket
	now       func() time.Time
}

type ticket struct {
	userID    string
	expiresAt time.Time
}

const ticketTTL = 30 * time.Second

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]struct{}),
		tickets: make(map[string]ticket),
		now:     time.Now,
	}
}

// IssueTicket returns a single-use, short-lived token that authenticates a
// WebSocket handshake (browsers cannot set Authorization headers on WebSockets,
// and long-lived access tokens must not end up in URLs or proxy logs).
func (h *Hub) IssueTicket(userID string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	value := base64.RawURLEncoding.EncodeToString(buf)

	h.ticketsMu.Lock()
	defer h.ticketsMu.Unlock()
	now := h.now()
	for k, t := range h.tickets {
		if now.After(t.expiresAt) {
			delete(h.tickets, k)
		}
	}
	h.tickets[value] = ticket{userID: userID, expiresAt: now.Add(ticketTTL)}
	return value, nil
}

// RedeemTicket consumes a ticket and returns its user id.
func (h *Hub) RedeemTicket(value string) (string, bool) {
	h.ticketsMu.Lock()
	defer h.ticketsMu.Unlock()
	t, ok := h.tickets[value]
	if !ok {
		return "", false
	}
	delete(h.tickets, value)
	if h.now().After(t.expiresAt) {
		return "", false
	}
	return t.userID, true
}

func (h *Hub) Register(userID string) *Client {
	c := &Client{userID: userID, send: make(chan []byte, clientBuffer)}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]struct{})
	}
	h.clients[userID][c] = struct{}{}
	return c
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[c.userID]
	if _, ok := set[c]; !ok {
		return
	}
	delete(set, c)
	close(c.send)
	if len(set) == 0 {
		delete(h.clients, c.userID)
	}
}

// Disconnect closes every connection of a user (logout everywhere, account deletion).
func (h *Hub) Disconnect(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients[userID] {
		close(c.send)
	}
	delete(h.clients, userID)
}

// Publish sends an event to every connection of userID. Slow clients whose buffer
// is full miss the event; they resynchronise over REST.
func (h *Hub) Publish(userID string, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- data:
		default:
		}
	}
}

// ConnectedCount is used by tests and diagnostics.
func (h *Hub) ConnectedCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}
