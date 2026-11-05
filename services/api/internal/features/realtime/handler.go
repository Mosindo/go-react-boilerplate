package realtime

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	ticketTTL  = 30 * time.Second
	pingEvery  = 30 * time.Second
	pongWait   = 70 * time.Second
	writeWait  = 10 * time.Second
	maxReadLen = 1024
)

// Handler issues single-use tickets and upgrades connections. Tickets keep
// long-lived JWTs out of URLs (and therefore out of proxy/access logs).
type Handler struct {
	hub      *Hub
	upgrader websocket.Upgrader
	mu       sync.Mutex
	tickets  map[string]ticket
}

type ticket struct {
	userID  string
	expires time.Time
}

func NewHandler(hub *Hub, allowedOrigins []string) *Handler {
	allowed := map[string]bool{}
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return &Handler{
		hub:     hub,
		tickets: make(map[string]ticket),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			// Native apps send no Origin header; browsers must be allow-listed.
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || allowed[origin]
			},
		},
	}
}

func (h *Handler) Ticket(c *gin.Context) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		httpx.Internal(c)
		return
	}
	t := base64.RawURLEncoding.EncodeToString(buf)
	now := time.Now()
	h.mu.Lock()
	for k, v := range h.tickets {
		if now.After(v.expires) {
			delete(h.tickets, k)
		}
	}
	h.tickets[t] = ticket{userID: httpx.UserID(c), expires: now.Add(ticketTTL)}
	h.mu.Unlock()
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": t})
}

func (h *Handler) redeem(t string) (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	v, ok := h.tickets[t]
	delete(h.tickets, t) // single use
	if !ok || time.Now().After(v.expires) {
		return "", false
	}
	return v.userID, true
}

func (h *Handler) Connect(c *gin.Context) {
	userID, ok := h.redeem(c.Query("ticket"))
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "invalid ticket")
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // Upgrade already replied
	}
	cl := &client{userID: userID, send: make(chan []byte, 32)}
	h.hub.register(cl)
	go h.writePump(conn, cl)
	h.readPump(conn, cl)
}

// readPump only services control frames; clients never push data this way.
func (h *Handler) readPump(conn *websocket.Conn, cl *client) {
	defer func() {
		h.hub.unregister(cl)
		_ = conn.Close()
	}()
	conn.SetReadLimit(maxReadLen)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	}
}

func (h *Handler) writePump(conn *websocket.Conn, cl *client) {
	ticker := time.NewTicker(pingEvery)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-cl.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, nil)
				_ = conn.Close()
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				_ = conn.Close()
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				_ = conn.Close()
				return
			}
		}
	}
}
