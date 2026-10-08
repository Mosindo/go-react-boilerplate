// Package realtime delivers server-pushed events to connected clients over WebSocket.
// The hub is in-memory (single API instance). To scale out, back Publish with a
// pub/sub transport (e.g. Postgres LISTEN/NOTIFY) without touching callers.
package realtime

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	ticketTTL    = 30 * time.Second
	writeWait    = 10 * time.Second
	pongWait     = 60 * time.Second
	pingPeriod   = 45 * time.Second
	sendBuffer   = 32
	maxConnsUser = 5
)

// Event is the JSON envelope pushed to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher is the narrow interface features depend on.
type Publisher interface {
	Publish(userID string, ev Event)
}

type client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
}

type ticket struct {
	userID  string
	expires time.Time
}

type Hub struct {
	mu       sync.RWMutex
	clients  map[string]map[*client]struct{}
	tickets  map[string]ticket
	upgrader websocket.Upgrader
}

// NewHub creates a hub. allowedOrigins restricts browser origins; native apps send none.
func NewHub(allowedOrigins []string) *Hub {
	allowed := map[string]struct{}{}
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	return &Hub{
		clients: map[string]map[*client]struct{}{},
		tickets: map[string]ticket{},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				_, ok := allowed[origin]
				return ok
			},
		},
	}
}

// Publish sends an event to every live connection of userID. Slow consumers drop events.
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

// IssueTicket returns a single-use, short-lived ticket so the JWT never travels in a URL.
func (h *Hub) IssueTicket(userID string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	t := base64.RawURLEncoding.EncodeToString(buf)
	now := time.Now()
	h.mu.Lock()
	for k, v := range h.tickets {
		if now.After(v.expires) {
			delete(h.tickets, k)
		}
	}
	h.tickets[t] = ticket{userID: userID, expires: now.Add(ticketTTL)}
	h.mu.Unlock()
	return t, nil
}

func (h *Hub) redeem(t string) (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	tk, ok := h.tickets[t]
	if !ok {
		return "", false
	}
	delete(h.tickets, t)
	if time.Now().After(tk.expires) {
		return "", false
	}
	return tk.userID, true
}

// TicketHandler: POST /ws/ticket (authenticated).
func (h *Hub) TicketHandler(c *gin.Context) {
	t, err := h.IssueTicket(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "internal"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": t, "expiresInSeconds": int(ticketTTL.Seconds())})
}

// ServeWS: GET /ws?ticket=... — upgrades and serves server->client events.
func (h *Hub) ServeWS(c *gin.Context) {
	userID, ok := h.redeem(c.Query("ticket"))
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid ticket", "code": "unauthorized"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	cl := &client{userID: userID, conn: conn, send: make(chan []byte, sendBuffer)}
	if !h.register(cl) {
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "too many connections"), time.Now().Add(writeWait))
		_ = conn.Close()
		return
	}
	go h.writeLoop(cl)
	h.readLoop(cl)
}

func (h *Hub) register(cl *client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[cl.userID]
	if set == nil {
		set = map[*client]struct{}{}
		h.clients[cl.userID] = set
	}
	if len(set) >= maxConnsUser {
		return false
	}
	set[cl] = struct{}{}
	return true
}

func (h *Hub) unregister(cl *client) {
	h.mu.Lock()
	if set := h.clients[cl.userID]; set != nil {
		delete(set, cl)
		if len(set) == 0 {
			delete(h.clients, cl.userID)
		}
	}
	h.mu.Unlock()
	close(cl.send)
}

// Online reports whether the user has at least one live connection.
func (h *Hub) Online(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

// Shutdown closes all connections.
func (h *Hub) Shutdown(_ context.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, set := range h.clients {
		for c := range set {
			_ = c.conn.Close()
		}
	}
}

func (h *Hub) readLoop(cl *client) {
	defer func() {
		h.unregister(cl)
		_ = cl.conn.Close()
	}()
	cl.conn.SetReadLimit(512)
	_ = cl.conn.SetReadDeadline(time.Now().Add(pongWait))
	cl.conn.SetPongHandler(func(string) error {
		return cl.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		// Clients never send application data; reads only service control frames.
		if _, _, err := cl.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) writeLoop(cl *client) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-cl.send:
			_ = cl.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = cl.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := cl.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf(`{"event":"ws_write_error","error":%q}`, err.Error())
				_ = cl.conn.Close()
				return
			}
		case <-ticker.C:
			_ = cl.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := cl.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				_ = cl.conn.Close()
				return
			}
		}
	}
}
