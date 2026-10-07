package realtime

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	ticketTTL    = 60 * time.Second
	writeWait    = 10 * time.Second
	pongWait     = 70 * time.Second
	pingInterval = 30 * time.Second
	maxReadBytes = 512
)

// Handler issues one-shot tickets (so long-lived credentials never appear in the
// WebSocket URL) and upgrades connections.
type Handler struct {
	hub      *Hub
	upgrader websocket.Upgrader

	mu      sync.Mutex
	tickets map[string]ticket
}

type ticket struct {
	userID  string
	expires time.Time
}

func NewHandler(hub *Hub, allowedOrigins []string) *Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	return &Handler{
		hub:     hub,
		tickets: make(map[string]ticket),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true // native apps do not send an Origin
				}
				if _, ok := allowed[origin]; ok {
					return true
				}
				u, err := url.Parse(origin)
				return err == nil && u.Host == r.Host
			},
		},
	}
}

func (h *Handler) Ticket(c *gin.Context) {
	userID := c.GetString("userID")
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong", "code": "internal"})
		return
	}
	value := base64.RawURLEncoding.EncodeToString(raw)
	now := time.Now()

	h.mu.Lock()
	for k, t := range h.tickets {
		if now.After(t.expires) {
			delete(h.tickets, k)
		}
	}
	h.tickets[value] = ticket{userID: userID, expires: now.Add(ticketTTL)}
	h.mu.Unlock()

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": value, "expiresIn": int(ticketTTL.Seconds())})
}

func (h *Handler) consume(value string) (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	t, ok := h.tickets[value]
	if !ok {
		return "", false
	}
	delete(h.tickets, value)
	if time.Now().After(t.expires) {
		return "", false
	}
	return t.userID, true
}

func (h *Handler) Serve(c *gin.Context) {
	userID, ok := h.consume(c.Query("ticket"))
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid ticket", "code": "unauthorized"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	cl := h.hub.register(userID)
	go h.writePump(conn, cl)
	h.readPump(conn, cl)
}

func (h *Handler) readPump(conn *websocket.Conn, cl *client) {
	defer func() {
		h.hub.unregister(cl)
		_ = conn.Close()
	}()
	conn.SetReadLimit(maxReadBytes)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		if containsPing(msg) {
			select {
			case cl.send <- []byte(`{"type":"pong"}`):
			default:
			}
		}
	}
}

func containsPing(msg []byte) bool {
	var in struct {
		Type string `json:"type"`
	}
	return json.Unmarshal(msg, &in) == nil && in.Type == "ping"
}

func (h *Handler) writePump(conn *websocket.Conn, cl *client) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()
	for {
		select {
		case payload, ok := <-cl.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
