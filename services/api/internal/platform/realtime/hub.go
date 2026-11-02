// Package realtime fans server events out to connected WebSocket clients.
//
// The hub is in-process: it is correct for a single API instance. To scale
// horizontally, publish through Postgres LISTEN/NOTIFY or Redis and have every
// instance feed its local hub.
package realtime

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Client struct {
	UserID string
	Send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}

	tmu     sync.Mutex
	tickets map[string]ticket
}

type ticket struct {
	userID  string
	expires time.Time
}

func NewHub() *Hub {
	return &Hub{clients: map[string]map[*Client]struct{}{}, tickets: map[string]ticket{}}
}

func (h *Hub) Register(userID string) *Client {
	c := &Client{UserID: userID, Send: make(chan []byte, 64)}
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = map[*Client]struct{}{}
	}
	h.clients[userID][c] = struct{}{}
	h.mu.Unlock()
	return c
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if set := h.clients[c.UserID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.UserID)
		}
	}
	h.mu.Unlock()
}

// Publish delivers the event to every connection of userID. Slow consumers
// drop events instead of blocking the caller; clients resync via REST.
func (h *Hub) Publish(userID string, ev Event) {
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.Send <- payload:
		default:
		}
	}
}

func (h *Hub) Online(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

// IssueTicket creates a single-use, 30 second credential for the WebSocket
// handshake, so access tokens never travel in URLs.
func (h *Hub) IssueTicket(userID string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	t := base64.RawURLEncoding.EncodeToString(buf)
	now := time.Now()
	h.tmu.Lock()
	defer h.tmu.Unlock()
	for k, v := range h.tickets {
		if now.After(v.expires) {
			delete(h.tickets, k)
		}
	}
	h.tickets[t] = ticket{userID: userID, expires: now.Add(30 * time.Second)}
	return t, nil
}

func (h *Hub) RedeemTicket(t string) (string, bool) {
	h.tmu.Lock()
	defer h.tmu.Unlock()
	v, ok := h.tickets[t]
	if !ok {
		return "", false
	}
	delete(h.tickets, t)
	if time.Now().After(v.expires) {
		return "", false
	}
	return v.userID, true
}
