// Package realtime fans server events out to connected WebSocket clients.
// The hub is in-memory (single API instance). To scale horizontally, back Publish with
// Postgres LISTEN/NOTIFY or Redis pub/sub; feature code only depends on Publisher.
package realtime

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher is what features use to push events to a user's connected devices.
type Publisher interface {
	Publish(userID string, ev Event)
}

// NoopPublisher is used where realtime is not wired (tests, seed).
type NoopPublisher struct{}

func (NoopPublisher) Publish(string, Event) {}

type client struct {
	send chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub { return &Hub{clients: make(map[string]map[*client]struct{})} }

func (h *Hub) add(userID string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*client]struct{})
	}
	h.clients[userID][c] = struct{}{}
}

func (h *Hub) remove(userID string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.clients[userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
}

// Publish never blocks: a client whose buffer is full is considered stalled and misses the event
// (it re-syncs through REST on reconnect).
func (h *Hub) Publish(userID string, ev Event) {
	payload, err := json.Marshal(ev)
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

// Disconnect closes every connection of a user (used on account deletion).
func (h *Hub) Disconnect(userID string) {
	h.mu.Lock()
	set := h.clients[userID]
	delete(h.clients, userID)
	h.mu.Unlock()
	for c := range set {
		close(c.send)
	}
}

const (
	authTimeout  = 10 * time.Second
	writeTimeout = 10 * time.Second
	pongWait     = 60 * time.Second
	pingEvery    = 25 * time.Second
	maxFrame     = 4096
)

type authMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades GET /ws. Native clients cannot reliably set headers on WebSockets, so
// the access token is sent as the first frame ({"type":"auth","token":"..."}) rather than in the URL
// (URLs end up in logs).
func (h *Hub) Handler(secret []byte, sessionActive middleware.SessionChecker, allowedOrigins []string) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" { // native apps send no Origin
				return true
			}
			if u, err := url.Parse(origin); err == nil && strings.EqualFold(u.Host, r.Host) {
				return true
			}
			for _, o := range allowedOrigins {
				if o == origin {
					return true
				}
			}
			return false
		},
	}

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetReadLimit(maxFrame)

		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
		var msg authMessage
		if err := conn.ReadJSON(&msg); err != nil || msg.Type != "auth" {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4401, "auth required"), time.Now().Add(time.Second))
			return
		}
		userID, sessionID, err := middleware.ParseAccessToken(secret, msg.Token)
		if err != nil {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4401, "invalid token"), time.Now().Add(time.Second))
			return
		}
		if sessionActive != nil {
			if ok, err := sessionActive(c.Request.Context(), sessionID); err != nil || !ok {
				_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4401, "session expired"), time.Now().Add(time.Second))
				return
			}
		}

		cl := &client{send: make(chan []byte, 32)}
		h.add(userID, cl)
		defer h.remove(userID, cl)

		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
			conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()

		ticker := time.NewTicker(pingEvery)
		defer ticker.Stop()
		write := func(kind int, data []byte) error {
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			return conn.WriteMessage(kind, data)
		}
		if err := write(websocket.TextMessage, []byte(`{"type":"ready"}`)); err != nil {
			return
		}
		for {
			select {
			case data, ok := <-cl.send:
				if !ok {
					return
				}
				if err := write(websocket.TextMessage, data); err != nil {
					log.Printf(`{"event":"ws_write_error","user_id":%q}`, userID)
					return
				}
			case <-ticker.C:
				if err := write(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}
}
