// Package realtime exposes the WebSocket endpoint and the single-use ticket that authorises it.
package realtime

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"example.com/api/internal/platform/httpx"
	platformrt "example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	ticketTTL    = 30 * time.Second
	writeWait    = 10 * time.Second
	pongWait     = 60 * time.Second
	pingInterval = 25 * time.Second
	maxFrameSize = 1024
)

type ticketStore struct {
	mu      sync.Mutex
	tickets map[string]ticket
}

type ticket struct {
	userID  string
	expires time.Time
}

func (s *ticketStore) issue(userID string) (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	t := base64.RawURLEncoding.EncodeToString(b)
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.tickets {
		if now.After(v.expires) {
			delete(s.tickets, k)
		}
	}
	s.tickets[t] = ticket{userID: userID, expires: now.Add(ticketTTL)}
	return t, nil
}

// redeem consumes the ticket: it works exactly once.
func (s *ticketStore) redeem(t string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.tickets[t]
	delete(s.tickets, t)
	if !ok || time.Now().After(v.expires) {
		return "", false
	}
	return v.userID, true
}

type Handler struct {
	hub      *platformrt.Hub
	tickets  *ticketStore
	upgrader websocket.Upgrader
}

// NewHandler builds the WebSocket handler. allowedOrigins restricts browser origins; native apps send none.
func NewHandler(hub *platformrt.Hub, allowedOrigins []string) *Handler {
	allowed := map[string]bool{}
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return &Handler{
		hub:     hub,
		tickets: &ticketStore{tickets: map[string]ticket{}},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" || allowed[origin] {
					return true
				}
				u, err := url.Parse(origin)
				return err == nil && strings.EqualFold(u.Host, r.Host)
			},
		},
	}
}

func (h *Handler) Ticket(c *gin.Context) {
	t, err := h.tickets.issue(httpx.UserID(c))
	if err != nil {
		httpx.Respond(c, "realtime.ticket", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": t, "expiresInSeconds": int(ticketTTL.Seconds())})
}

func (h *Handler) Connect(c *gin.Context) {
	userID, ok := h.tickets.redeem(c.Query("ticket"))
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "invalid or expired ticket")
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &platformrt.Client{UserID: userID, Send: make(chan []byte, 64)}
	if !h.hub.Register(client) {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "too many connections"), time.Now().Add(writeWait))
		_ = conn.Close()
		return
	}
	go h.writeLoop(conn, client)
	h.readLoop(conn, client)
}

func (h *Handler) readLoop(conn *websocket.Conn, client *platformrt.Client) {
	defer func() {
		h.hub.Unregister(client)
		close(client.Send)
		_ = conn.Close()
	}()
	conn.SetReadLimit(maxFrameSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		var frame struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(data, &frame) == nil && frame.Type == "ping" {
			select {
			case client.Send <- []byte(`{"type":"pong","data":{}}`):
			default:
			}
		}
	}
}

func (h *Handler) writeLoop(conn *websocket.Conn, client *platformrt.Client) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-client.Send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
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
