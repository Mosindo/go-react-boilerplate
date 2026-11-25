// Package realtime pushes server events (new messages, read receipts, matches,
// notifications) to connected clients over WebSocket.
//
// The hub is in-memory: it fits the single-instance deployment of this
// project. To scale horizontally, back Publish with Postgres LISTEN/NOTIFY or
// a message broker; callers only depend on the Publisher interface.
package realtime

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

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

	ticketsMu sync.Mutex
	tickets   map[string]ticket
}

type ticket struct {
	userID    string
	expiresAt time.Time
}

const (
	ticketTTL      = 60 * time.Second
	sendBuffer     = 32
	writeTimeout   = 10 * time.Second
	pingInterval   = 25 * time.Second
	maxConnsByUser = 5
)

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{}), tickets: make(map[string]ticket)}
}

// Publish delivers event to every live connection of userID. Slow clients are
// skipped rather than blocking the publisher.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf(`{"event":"realtime_marshal_error","error":%q}`, err.Error())
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

// IssueTicket returns a single-use, short-lived ticket that authenticates a
// WebSocket upgrade. Tickets avoid putting long-lived tokens in URLs.
func (h *Hub) IssueTicket(userID string) (string, time.Time, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, err
	}
	value := base64.RawURLEncoding.EncodeToString(buf)
	expiresAt := time.Now().Add(ticketTTL)

	h.ticketsMu.Lock()
	defer h.ticketsMu.Unlock()
	now := time.Now()
	for k, t := range h.tickets {
		if now.After(t.expiresAt) {
			delete(h.tickets, k)
		}
	}
	h.tickets[value] = ticket{userID: userID, expiresAt: expiresAt}
	return value, expiresAt, nil
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
	if time.Now().After(t.expiresAt) {
		return "", false
	}
	return t.userID, true
}

// ConnectionCount returns the number of live connections for userID.
func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}

func (h *Hub) register(c *client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[c.userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.clients[c.userID] = set
	}
	if len(set) >= maxConnsByUser {
		return false
	}
	set[c] = struct{}{}
	return true
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.clients[c.userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Serve runs the read/write loops of an accepted connection until it closes.
func (h *Hub) Serve(ctx context.Context, conn *websocket.Conn, userID string) {
	c := &client{userID: userID, send: make(chan []byte, sendBuffer)}
	if !h.register(c) {
		conn.Close(websocket.StatusPolicyViolation, "too many connections")
		return
	}
	defer h.unregister(c)

	// Clients never need to send data; reading only processes control frames
	// and detects disconnects. Large frames are rejected.
	conn.SetReadLimit(512)
	ctx = conn.CloseRead(ctx)

	h.Publish(userID, Event{Type: "ready", Data: map[string]string{}})

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			conn.Close(websocket.StatusNormalClosure, "")
			return
		case payload := <-c.send:
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := conn.Write(writeCtx, websocket.MessageText, payload)
			cancel()
			if err != nil {
				return
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
