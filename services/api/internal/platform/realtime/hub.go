// Package realtime pushes server events to connected clients over WebSocket.
//
// Clients connect to GET /ws and authenticate with a first {"type":"auth"}
// frame (so tokens never appear in URLs or logs). The hub is in-process: with
// several API replicas, back it with Postgres LISTEN/NOTIFY or a broker.
package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout  = 5 * time.Second
	writeTimeout = 10 * time.Second
	pongWait     = 60 * time.Second
	pingEvery    = 25 * time.Second
	maxFrameSize = 4096
	sendBuffer   = 64
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher is the narrow interface feature services depend on.
type Publisher interface {
	Publish(userID string, ev Event)
}

// NopPublisher discards events (used in unit tests).
type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}

type client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
	once   sync.Once
}

func (c *client) close() {
	c.once.Do(func() {
		close(c.send)
		_ = c.conn.Close()
	})
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

func (h *Hub) Publish(userID string, ev Event) {
	payload, err := json.Marshal(ev)
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
		select {
		case c.send <- payload:
		default:
			// Slow consumer: drop the connection, the app reconnects and refetches.
			h.remove(c)
		}
	}
}

// Disconnect closes every connection of a user (account deletion, logout all).
func (h *Hub) Disconnect(userID string) {
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

func (h *Hub) add(c *client) {
	h.mu.Lock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
	h.mu.Unlock()
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

// Authenticator validates an access token and returns the user id.
type Authenticator func(ctx context.Context, token string) (string, error)

func (h *Hub) Handler(authenticate Authenticator, allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // native clients do not send Origin
			}
			if _, ok := allowed[origin]; ok {
				return true
			}
			u, err := url.Parse(origin)
			return err == nil && u.Host == r.Host
		},
	}

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		conn.SetReadLimit(maxFrameSize)

		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
		var first struct {
			Type  string `json:"type"`
			Token string `json:"token"`
		}
		if err := conn.ReadJSON(&first); err != nil || first.Type != "auth" {
			_ = conn.Close()
			return
		}
		userID, err := authenticate(c.Request.Context(), first.Token)
		if err != nil {
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "unauthorized"), time.Now().Add(time.Second))
			_ = conn.Close()
			return
		}

		cl := &client{userID: userID, conn: conn, send: make(chan []byte, sendBuffer)}
		h.add(cl)
		go h.writeLoop(cl)
		cl.send <- []byte(`{"type":"ready"}`)
		h.readLoop(cl)
	}
}

func (h *Hub) readLoop(c *client) {
	defer h.remove(c)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		// Clients never send commands; any frame (e.g. app-level ping) only keeps the link alive.
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	}
}

func (h *Hub) writeLoop(c *client) {
	ticker := time.NewTicker(pingEvery)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
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
