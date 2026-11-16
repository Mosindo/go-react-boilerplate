// Package realtime pushes server events to connected clients over WebSocket.
//
// Authentication uses single-use, short-lived tickets obtained through an
// authenticated REST call, so access tokens never travel in URLs (and thus
// never end up in proxy logs). The hub is in-process: with several API
// replicas, swap Publish for a fan-out over Postgres LISTEN/NOTIFY or Redis.
package realtime

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"
)

// Event is the envelope sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher is the dependency features use to emit events.
type Publisher interface {
	Publish(userID string, event Event)
}

type client struct {
	userID    string
	sessionID string
	send      chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}

	ticketsMu sync.Mutex
	tickets   map[string]ticket
	now       func() time.Time
}

type ticket struct {
	userID    string
	sessionID string
	expiresAt time.Time
}

const ticketTTL = 30 * time.Second

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*client]struct{}),
		tickets: make(map[string]ticket),
		now:     time.Now,
	}
}

// maxTickets bounds pending tickets so the map cannot grow without limit.
const maxTickets = 50_000

// IssueTicket returns a one-time ticket valid for 30 seconds, bound to the
// session that requested it.
func (h *Hub) IssueTicket(userID, sessionID string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	value := base64.RawURLEncoding.EncodeToString(buf)

	h.ticketsMu.Lock()
	defer h.ticketsMu.Unlock()
	now := h.now()
	if len(h.tickets) >= maxTickets/2 {
		for k, t := range h.tickets {
			if now.After(t.expiresAt) {
				delete(h.tickets, k)
			}
		}
	}
	if len(h.tickets) >= maxTickets {
		return "", errTooManyTickets
	}
	h.tickets[value] = ticket{userID: userID, sessionID: sessionID, expiresAt: now.Add(ticketTTL)}
	return value, nil
}

var errTooManyTickets = errors.New("too many pending realtime tickets")

// RedeemTicket consumes a ticket and returns its user and session.
func (h *Hub) RedeemTicket(value string) (string, string, bool) {
	h.ticketsMu.Lock()
	defer h.ticketsMu.Unlock()
	t, ok := h.tickets[value]
	if !ok {
		return "", "", false
	}
	delete(h.tickets, value)
	if h.now().After(t.expiresAt) {
		return "", "", false
	}
	return t.userID, t.sessionID, true
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

// Publish sends an event to every connection of a user. Slow consumers whose
// buffer is full are disconnected instead of blocking the publisher.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf(`{"event":"realtime_marshal_error","error":%q}`, err.Error())
		return
	}
	var slow []*client
	h.mu.RLock()
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
	defer h.mu.Unlock()
	for c := range h.clients[userID] {
		close(c.send)
	}
	delete(h.clients, userID)
}

// ConnectionCount is used by tests and health diagnostics.
func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}
