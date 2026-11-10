// Package realtime fans out server events to connected clients (WebSocket).
//
// The hub is process-local. Running several API replicas requires a shared
// bus (e.g. Postgres LISTEN/NOTIFY or Redis pub/sub) feeding Hub.Publish; the
// Publisher interface keeps features independent from that choice. Clients
// always re-sync over REST on (re)connect, so realtime is an optimization and
// never the only path to data.
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
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher is what features depend on.
type Publisher interface {
	Publish(userID string, event Event)
}

// Client is a single connection's outbound queue.
type Client struct {
	UserID string
	Send   chan []byte
}

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

const (
	ticketTTL       = 60 * time.Second
	clientQueueSize = 64
)

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]struct{}),
		tickets: make(map[string]ticket),
		now:     time.Now,
	}
}

func (h *Hub) Register(userID string) *Client {
	c := &Client{UserID: userID, Send: make(chan []byte, clientQueueSize)}
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
	set := h.clients[c.UserID]
	if _, ok := set[c]; !ok {
		return
	}
	delete(set, c)
	close(c.Send)
	if len(set) == 0 {
		delete(h.clients, c.UserID)
	}
}

// Disconnect closes every connection of a user (account deletion, block...).
func (h *Hub) Disconnect(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients[userID] {
		close(c.Send)
	}
	delete(h.clients, userID)
}

// Publish never blocks: a client whose queue is full misses the event and
// will catch up through its REST re-sync.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
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

func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}

// IssueTicket returns a single-use, short-lived token used to open the
// WebSocket. This keeps the long-lived access token out of URLs and logs.
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

// RedeemTicket consumes a ticket and returns its user.
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
