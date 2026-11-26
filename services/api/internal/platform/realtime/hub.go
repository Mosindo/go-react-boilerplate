// Package realtime delivers server events to connected clients over WebSocket.
//
// The hub is in-process: it fans events out to every connection of a user on this instance.
// Running several API instances requires a shared bus (e.g. Postgres LISTEN/NOTIFY or Redis)
// feeding Publish; the Publisher interface keeps features independent of that choice.
package realtime

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher is what features depend on to push events to users.
type Publisher interface {
	Publish(userID string, event Event)
}

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

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[c.userID]
	if !ok {
		set = make(map[*client]struct{})
		h.clients[c.userID] = set
	}
	set[c] = struct{}{}
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.userID]; ok {
		if _, present := set[c]; present {
			delete(set, c)
			close(c.send)
		}
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Publish never blocks: a connection whose buffer is full misses the event and will
// resynchronise through the REST API (clients refetch on reconnect).
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
		}
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

// Tickets are single-use, short-lived credentials exchanged for a WebSocket connection,
// so access tokens never appear in URLs or proxy logs.
type Tickets struct {
	mu      sync.Mutex
	ttl     time.Duration
	tickets map[string]ticket
}

type ticket struct {
	userID    string
	expiresAt time.Time
}

func NewTickets(ttl time.Duration) *Tickets {
	return &Tickets{ttl: ttl, tickets: make(map[string]ticket)}
}

func (t *Tickets) Issue(userID string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	value := hex.EncodeToString(buf)
	now := time.Now()

	t.mu.Lock()
	defer t.mu.Unlock()
	for k, v := range t.tickets {
		if now.After(v.expiresAt) {
			delete(t.tickets, k)
		}
	}
	t.tickets[value] = ticket{userID: userID, expiresAt: now.Add(t.ttl)}
	return value, nil
}

func (t *Tickets) Redeem(value string) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	tk, ok := t.tickets[value]
	if !ok {
		return "", false
	}
	delete(t.tickets, value)
	if time.Now().After(tk.expiresAt) {
		return "", false
	}
	return tk.userID, true
}
