// Package realtime pushes server events to connected clients over WebSocket.
//
// The hub is in-memory: it serves a single API instance. To scale horizontally, fan events out
// between instances (for example with Postgres LISTEN/NOTIFY or Redis pub/sub) behind Publish.
package realtime

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout   = 5 * time.Second
	writeTimeout  = 10 * time.Second
	pongTimeout   = 60 * time.Second
	pingInterval  = 25 * time.Second
	maxFrameBytes = 4096
	sendBuffer    = 64
)

// Event is the envelope delivered to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher is the narrow interface feature services depend on.
type Publisher interface {
	Publish(userID string, event Event)
}

// Authenticator validates an access token and returns the user id and the token expiry.
type Authenticator func(token string) (userID string, expiresAt time.Time, err error)

type client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
	once   sync.Once
}

func (c *client) close() { c.once.Do(func() { close(c.send); _ = c.conn.Close() }) }

type Hub struct {
	mu       sync.RWMutex
	clients  map[string]map[*client]struct{}
	upgrader websocket.Upgrader
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*client]struct{}),
		// Authentication happens with the first frame (bearer token), not with cookies, so the
		// Origin header carries no authority and browsers on other origins may connect.
		upgrader: websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: func(*http.Request) bool { return true }},
	}
}

// Publish delivers an event to every open connection of userID. Slow consumers are dropped.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	targets := make([]*client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		if !h.trySend(c, payload) {
			h.remove(c)
		}
	}
}

func (h *Hub) trySend(c *client, payload []byte) (ok bool) {
	defer func() {
		if recover() != nil { // send on closed channel: client already gone
			ok = false
		}
	}()
	select {
	case c.send <- payload:
		return true
	default:
		return false
	}
}

// DisconnectUser closes every connection of a user (account deletion, logout everywhere).
func (h *Hub) DisconnectUser(userID string) {
	h.mu.RLock()
	targets := make([]*client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		h.remove(c)
	}
}

// Online reports whether the user has an open connection.
func (h *Hub) Online(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	if set := h.clients[c.userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
	h.mu.Unlock()
	c.close()
}

type authFrame struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

var errBadAuth = errors.New("realtime: authentication frame required")

// Handler upgrades GET /ws. The client must send {"type":"auth","token":"<access token>"} within 5 seconds.
// The connection is closed when the access token expires; clients refresh and reconnect.
func (h *Hub) Handler(authenticate Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		conn.SetReadLimit(maxFrameBytes)

		userID, expiresAt, err := h.handshake(conn, authenticate)
		if err != nil {
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(4401, "unauthorized"), time.Now().Add(time.Second))
			_ = conn.Close()
			return
		}

		cl := &client{userID: userID, conn: conn, send: make(chan []byte, sendBuffer)}
		h.add(cl)
		h.trySend(cl, mustJSON(Event{Type: "ready"}))

		expiry := time.AfterFunc(time.Until(expiresAt), func() { h.remove(cl) })
		go h.writeLoop(cl)
		h.readLoop(cl)
		expiry.Stop()
		h.remove(cl)
	}
}

func (h *Hub) handshake(conn *websocket.Conn, authenticate Authenticator) (string, time.Time, error) {
	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	var frame authFrame
	if err := conn.ReadJSON(&frame); err != nil || frame.Type != "auth" || frame.Token == "" {
		return "", time.Time{}, errBadAuth
	}
	return authenticate(frame.Token)
}

func (h *Hub) readLoop(c *client) {
	_ = c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	})
	for {
		// Clients only keep the socket alive; inbound frames are read and discarded.
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	}
}

func (h *Hub) writeLoop(c *client) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case payload, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				h.remove(c)
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.remove(c)
				return
			}
		}
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("realtime: marshal: %v", err)
		return []byte(`{"type":"error"}`)
	}
	return b
}

// Noop discards events; useful for tests and one-off tools.
type Noop struct{}

func (Noop) Publish(string, Event) {}
