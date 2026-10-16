// Package realtime fans events out to connected WebSocket clients. The hub is
// in-process: with several API instances, back it with Postgres LISTEN/NOTIFY or Redis.
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
	Data any    `json:"data,omitempty"`
}

// Client is one live connection. Send must never block the publisher.
type Client struct {
	UserID string
	send   chan []byte
}

func NewClient(userID string) *Client {
	return &Client{UserID: userID, send: make(chan []byte, 32)}
}

// Messages is the outbound queue; it is closed when the hub drops the client.
func (c *Client) Messages() <-chan []byte { return c.send }

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
	h.drop(c)
}

func (h *Hub) drop(c *Client) {
	set := h.clients[c.UserID]
	if _, ok := set[c]; !ok {
		return
	}
	delete(set, c)
	close(c.send)
	if len(set) == 0 {
		delete(h.clients, c.UserID)
	}
}

// Publish delivers an event to every connection of a user. Slow clients are dropped
// rather than allowed to stall the caller; they reconnect and resync over REST.
func (h *Hub) Publish(userID string, ev Event) {
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
			h.drop(c)
		}
	}
}

// Online reports whether the user has at least one live connection.
func (h *Hub) Online(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

// Tickets are single-use, short-lived credentials for the WebSocket handshake, so the
// access token never travels in a URL.
type Tickets struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	entries map[string]ticket
}

type ticket struct {
	userID  string
	expires time.Time
}

func NewTickets(ttl time.Duration) *Tickets {
	return &Tickets{ttl: ttl, now: time.Now, entries: make(map[string]ticket)}
}

func (t *Tickets) Issue(userID string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	value := base64.RawURLEncoding.EncodeToString(buf)

	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	for k, v := range t.entries {
		if now.After(v.expires) {
			delete(t.entries, k)
		}
	}
	t.entries[value] = ticket{userID: userID, expires: now.Add(t.ttl)}
	return value, nil
}

// Redeem consumes the ticket; a second call with the same value fails.
func (t *Tickets) Redeem(value string) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.entries[value]
	if !ok {
		return "", false
	}
	delete(t.entries, value)
	if t.now().After(entry.expires) {
		return "", false
	}
	return entry.userID, true
}

// Kick drops every connection of a user (account deletion, forced sign-out).
func (h *Hub) Kick(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients[userID] {
		h.drop(c)
	}
}
