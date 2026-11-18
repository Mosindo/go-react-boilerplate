// Package realtime fans server events out to connected members over
// WebSocket. Clients never send data through the socket: every write goes
// through the REST API (validated and authorised there) and the socket only
// pushes the resulting events.
//
// The hub is in-process. Running several API replicas requires a shared bus
// (for example Postgres LISTEN/NOTIFY or Redis pub/sub) behind Publisher.
package realtime

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 25 * time.Second
	maxInboundSize = 512
	sendBuffer     = 32
	// MaxConnectionsPerUser bounds sockets per member; the oldest is evicted.
	MaxConnectionsPerUser = 5
)

// SessionChecker lets the hub drop sockets whose session was revoked.
type SessionChecker interface {
	SessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

// Event is the envelope pushed to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher is what features depend on.
type Publisher interface {
	Publish(userID string, event Event)
	Disconnect(userID string)
}

type client struct {
	userID    string
	sessionID string
	conn      *websocket.Conn
	send      chan []byte
	createdAt time.Time

	mu     sync.Mutex
	closed bool
}

// trySend never blocks and never sends on a closed channel.
func (c *client) trySend(payload []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return true
	}
	select {
	case c.send <- payload:
		return true
	default:
		return false
	}
}

func (c *client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

type Hub struct {
	mu       sync.RWMutex
	clients  map[string]map[*client]struct{}
	sessions SessionChecker
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

// WithSessionChecker makes every ping also verify the socket's session.
func (h *Hub) WithSessionChecker(checker SessionChecker) *Hub {
	h.sessions = checker
	return h
}

// Publish delivers an event to every connection of userID. Slow consumers are
// dropped instead of blocking the publisher.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf(`{"event":"realtime_marshal_error","type":%q}`, event.Type)
		return
	}
	h.mu.RLock()
	targets := make([]*client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		if !c.trySend(payload) {
			h.unregister(c)
		}
	}
}

// DisconnectSession closes the sockets opened with one session (logout).
func (h *Hub) DisconnectSession(sessionID string) {
	h.mu.RLock()
	targets := make([]*client, 0)
	for _, set := range h.clients {
		for c := range set {
			if c.sessionID == sessionID {
				targets = append(targets, c)
			}
		}
	}
	h.mu.RUnlock()
	for _, c := range targets {
		h.unregister(c)
	}
}

// Disconnect closes every socket of a member (logout, account deletion).
func (h *Hub) Disconnect(userID string) {
	h.mu.RLock()
	targets := make([]*client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		h.unregister(c)
	}
}

// ConnectionCount is used by tests and diagnostics.
func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*client]struct{})
	}
	set := h.clients[c.userID]
	var oldest *client
	if len(set) >= MaxConnectionsPerUser {
		for existing := range set {
			if oldest == nil || existing.createdAt.Before(oldest.createdAt) {
				oldest = existing
			}
		}
	}
	set[c] = struct{}{}
	h.mu.Unlock()
	if oldest != nil {
		h.unregister(oldest)
	}
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	if set, ok := h.clients[c.userID]; ok {
		if _, present := set[c]; present {
			delete(set, c)
			if len(set) == 0 {
				delete(h.clients, c.userID)
			}
		}
	}
	h.mu.Unlock()
	c.close()
}

// Serve takes ownership of an upgraded connection until it closes.
func (h *Hub) Serve(userID, sessionID string, conn *websocket.Conn) {
	c := &client{userID: userID, sessionID: sessionID, conn: conn, send: make(chan []byte, sendBuffer), createdAt: time.Now()}
	h.register(c)
	go h.writePump(c)
	h.readPump(c)
}

func (h *Hub) readPump(c *client) {
	defer func() {
		h.unregister(c)
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(maxInboundSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		// Inbound frames are ignored; reading keeps pong/close handling alive.
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) writePump(c *client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	hello, _ := json.Marshal(Event{Type: "connection.ready", Data: map[string]string{"userId": c.userID}})
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteMessage(websocket.TextMessage, hello); err != nil {
		return
	}
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			if !h.sessionStillActive(c) {
				h.unregister(c)
				continue
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) sessionStillActive(c *client) bool {
	if h.sessions == nil || c.sessionID == "" {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	active, err := h.sessions.SessionActive(ctx, c.sessionID, c.userID)
	return err != nil || active // keep the socket on transient DB errors
}
