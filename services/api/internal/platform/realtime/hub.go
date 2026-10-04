// Package realtime pushes server events to connected clients over WebSocket.
// Clients only receive; every write goes through the REST API so that persistence and
// authorization stay in one place. The hub is in-memory (single API instance); swap Publisher
// for a Postgres LISTEN/NOTIFY or Redis fan-out to scale horizontally.
package realtime

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout   = 5 * time.Second
	writeTimeout  = 10 * time.Second
	pongTimeout   = 60 * time.Second
	pingInterval  = 25 * time.Second
	maxFrameBytes = 4 << 10
	sendBuffer    = 32
	maxPerUser    = 5
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher is what features depend on.
type Publisher interface {
	Publish(userID string, eventType string, data any)
}

type Nop struct{}

func (Nop) Publish(string, string, any) {}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}

	secret   []byte
	validate middleware.SessionValidator
	upgrader websocket.Upgrader
}

type client struct {
	conn *websocket.Conn
	send chan []byte
}

func NewHub(secret []byte, validate middleware.SessionValidator) *Hub {
	return &Hub{
		clients:  map[string]map[*client]struct{}{},
		secret:   secret,
		validate: validate,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			// Cross-site WebSocket hijacking relies on ambient credentials (cookies). This endpoint has
			// none: the bearer token travels in the first frame, so any origin may connect.
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

// Publish delivers an event to every live connection of userID. It never blocks.
func (h *Hub) Publish(userID string, eventType string, data any) {
	payload, err := json.Marshal(Event{Type: eventType, Data: data})
	if err != nil {
		log.Printf("realtime: marshal %s: %v", eventType, err)
		return
	}
	h.mu.RLock()
	targets := make([]*client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		select {
		case c.send <- payload:
		default:
			// Slow consumer: drop it, it will reconnect and resync through REST.
			c.conn.Close()
		}
	}
}

// ConnectedUsers is used by tests and health diagnostics.
func (h *Hub) ConnectedUsers() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) register(userID string, c *client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[userID]
	if set == nil {
		set = map[*client]struct{}{}
		h.clients[userID] = set
	}
	if len(set) >= maxPerUser {
		return false
	}
	set[c] = struct{}{}
	return true
}

func (h *Hub) unregister(userID string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.clients[userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
}

// Handler upgrades the request and runs the connection until it closes.
func (h *Hub) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return // Upgrade already replied.
		}
		go h.serve(conn)
	}
}

func (h *Hub) serve(conn *websocket.Conn) {
	defer conn.Close()
	conn.SetReadLimit(maxFrameBytes)

	// 1. Authenticate with the first frame (keeps tokens out of URLs and logs).
	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	var first struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := conn.ReadJSON(&first); err != nil || first.Type != "auth" {
		closeWith(conn, 4401, "auth required")
		return
	}
	userID, sessionID, err := middleware.ParseAccessToken(h.secret, first.Token)
	if err != nil {
		closeWith(conn, 4401, "invalid token")
		return
	}
	if h.validate != nil {
		ok, err := h.validate(context.Background(), userID, sessionID)
		if err != nil || !ok {
			closeWith(conn, 4401, "invalid session")
			return
		}
	}

	cl := &client{conn: conn, send: make(chan []byte, sendBuffer)}
	if !h.register(userID, cl) {
		closeWith(conn, 4429, "too many connections")
		return
	}
	defer h.unregister(userID, cl)

	ready, _ := json.Marshal(Event{Type: "ready"})
	cl.send <- ready

	done := make(chan struct{})
	go h.writeLoop(cl, done)

	// 2. Drain incoming frames (ignored) to process pongs and detect disconnects.
	_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongTimeout))
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
	}
	close(done)
}

func (h *Hub) writeLoop(c *client, done <-chan struct{}) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case msg := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.conn.Close()
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.conn.Close()
				return
			}
		}
	}
}

func closeWith(conn *websocket.Conn, code int, reason string) {
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
}
