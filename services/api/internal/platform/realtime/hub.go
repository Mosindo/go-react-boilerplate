// Package realtime delivers server-pushed events to connected clients over WebSocket.
// The hub is in-memory and therefore scoped to a single API process.
package realtime

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/gin-gonic/gin"
)

const (
	authTimeout   = 10 * time.Second
	writeTimeout  = 10 * time.Second
	pongTimeout   = 60 * time.Second
	pingInterval  = 25 * time.Second
	maxMessage    = 4096
	sendQueueSize = 32
)

// Event is the envelope pushed to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher pushes events to a user's connected devices.
type Publisher interface {
	Publish(userID string, event Event)
}

// TokenValidator resolves an access token to a user id.
type TokenValidator func(token string) (userID string, err error)

type client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[c.userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.clients[c.userID] = set
	}
	set[c] = struct{}{}
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.clients[c.userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Publish sends the event to every connection of the user. Slow consumers are dropped.
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
			_ = c.conn.Close()
		}
	}
}

// Connected reports whether the user has at least one live connection.
func (h *Hub) Connected(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

func originAllowed(allowed []string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // native clients send no Origin
		}
		for _, a := range allowed {
			if a == origin {
				return true
			}
		}
		u, err := url.Parse(origin)
		return err == nil && strings.EqualFold(u.Host, r.Host)
	}
}

// Handler upgrades GET /ws. The client must send {"type":"auth","token":"<access token>"}
// as its first frame within a few seconds; tokens never travel in the URL.
func (h *Hub) Handler(validate TokenValidator, allowedOrigins []string) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     originAllowed(allowedOrigins),
	}
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		conn.SetReadLimit(maxMessage)
		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))

		var hello struct {
			Type  string `json:"type"`
			Token string `json:"token"`
		}
		if err := conn.ReadJSON(&hello); err != nil || hello.Type != "auth" {
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "auth required"), time.Now().Add(time.Second))
			_ = conn.Close()
			return
		}
		userID, err := validate(hello.Token)
		if err != nil {
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid token"), time.Now().Add(time.Second))
			_ = conn.Close()
			return
		}

		cl := &client{userID: userID, conn: conn, send: make(chan []byte, sendQueueSize)}
		h.add(cl)
		defer func() {
			h.remove(cl)
			_ = conn.Close()
		}()

		_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongTimeout))
		})

		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				// Clients only keep the connection alive; discard anything they send.
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
				_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
			}
		}()

		ready, _ := json.Marshal(Event{Type: "ready"})
		_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := conn.WriteMessage(websocket.TextMessage, ready); err != nil {
			return
		}

		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case msg := <-cl.send:
				_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}
}
