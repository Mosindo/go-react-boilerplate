// Package realtime pushes server events to connected clients over WebSocket.
//
// The channel is server -> client only: every write (messages, swipes, reads)
// goes through the validated REST API, and the hub simply fans the resulting
// events out to the right users. The hub is in-memory, so it serves a single
// API instance; swap Publish for Postgres LISTEN/NOTIFY or Redis pub/sub to
// scale horizontally.
package realtime

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

const (
	ticketAudience  = "ws"
	ticketTTL       = 60 * time.Second
	writeWait       = 10 * time.Second
	pongWait        = 60 * time.Second
	pingPeriod      = 25 * time.Second
	sendBuffer      = 32
	maxConnsPerUser = 5
)

// Event is the JSON envelope sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type client struct {
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

// Publish delivers an event to every live connection of userID. It never blocks:
// a client whose buffer is full is dropped and must reconnect (and refetch).
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	conns := make([]*client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	for _, c := range conns {
		select {
		case c.send <- payload:
		default:
			h.remove(userID, c)
		}
	}
}

// Disconnect closes every connection of a user (e.g. after account deletion).
func (h *Hub) Disconnect(userID string) {
	h.mu.RLock()
	conns := make([]*client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		conns = append(conns, c)
	}
	h.mu.RUnlock()
	for _, c := range conns {
		h.remove(userID, c)
	}
}

func (h *Hub) add(userID string, c *client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.clients[userID] = set
	}
	if len(set) >= maxConnsPerUser {
		return false
	}
	set[c] = struct{}{}
	return true
}

func (h *Hub) remove(userID string, c *client) {
	h.mu.Lock()
	set := h.clients[userID]
	_, ok := set[c]
	if ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
	h.mu.Unlock()
	if ok {
		close(c.send)
		_ = c.conn.Close()
	}
}

// IssueTicket mints a short-lived, single-purpose token so the WebSocket
// handshake never carries the long-lived access token in a URL.
func IssueTicket(secret []byte, userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		Audience:  jwt.ClaimStrings{ticketAudience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ticketTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func parseTicket(secret []byte, raw string) (string, bool) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	}, jwt.WithAudience(ticketAudience), jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.Subject == "" {
		return "", false
	}
	return claims.Subject, true
}

// Handler upgrades GET /ws?ticket=... to a WebSocket.
func (h *Hub) Handler(secret []byte, allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" { // native apps send no Origin
				return true
			}
			_, ok := allowed[origin]
			return ok
		},
	}

	return func(c *gin.Context) {
		userID, ok := parseTicket(secret, c.Query("ticket"))
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid ticket"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return // Upgrade already replied
		}
		cl := &client{conn: conn, send: make(chan []byte, sendBuffer)}
		if !h.add(userID, cl) {
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "too many connections"),
				time.Now().Add(writeWait))
			_ = conn.Close()
			return
		}
		go h.writePump(userID, cl)
		h.readPump(userID, cl)
	}
}

func (h *Hub) readPump(userID string, c *client) {
	defer h.remove(userID, c)
	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		// Inbound frames are ignored; reading keeps pongs and close frames flowing.
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) writePump(userID string, c *client) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case payload, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				log.Printf(`{"event":"ws_write_error","user_id":%q}`, userID)
				h.remove(userID, c)
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.remove(userID, c)
				return
			}
		}
	}
}
