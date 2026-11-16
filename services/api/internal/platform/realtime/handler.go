package realtime

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 25 * time.Second
	sendBuffer = 64
)

// SessionChecker reports whether a session is still active.
type SessionChecker func(ctx context.Context, sessionID, userID string) (bool, error)

type Handler struct {
	hub          *Hub
	upgrader     websocket.Upgrader
	checkSession SessionChecker
}

// NewHandler builds the WebSocket handlers. Native apps send no Origin header;
// browsers must come from ALLOWED_ORIGINS when that list is configured.
//
// Every open socket re-validates its session on each ping (25 s), so logout,
// password change or account deletion also closes realtime delivery.
func NewHandler(hub *Hub, allowedOrigins []string, checkSession SessionChecker) *Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}
	return &Handler{
		hub:          hub,
		checkSession: checkSession,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" || len(allowed) == 0 {
					return true
				}
				_, ok := allowed[origin]
				return ok
			},
		},
	}
}

// Ticket issues a one-time WebSocket ticket for the authenticated user.
func (h *Handler) Ticket(c *gin.Context) {
	value, err := h.hub.IssueTicket(c.GetString("userID"), c.GetString("sessionID"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not issue ticket", "code": "internal"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": value, "expiresInSeconds": int(ticketTTL.Seconds())})
}

// Connect upgrades the connection after redeeming the ticket.
func (h *Handler) Connect(c *gin.Context) {
	userID, sessionID, ok := h.hub.RedeemTicket(c.Query("ticket"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid ticket", "code": "unauthorized"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	cl := &client{userID: userID, sessionID: sessionID, send: make(chan []byte, sendBuffer)}
	h.hub.register(cl)

	go h.writePump(conn, cl)
	readPump(conn, h.hub, cl)
}

func readPump(conn *websocket.Conn, hub *Hub, cl *client) {
	defer func() {
		hub.unregister(cl)
		_ = conn.Close()
	}()
	conn.SetReadLimit(512)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		// Clients do not send commands; reading only drives pong/close handling.
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	}
}

func (h *Handler) sessionActive(cl *client) bool {
	if h.checkSession == nil {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	active, err := h.checkSession(ctx, cl.sessionID, cl.userID)
	// On a transient DB error keep the socket; the next tick re-checks.
	return active || err != nil
}

func (h *Handler) writePump(conn *websocket.Conn, cl *client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()
	for {
		select {
		case msg, ok := <-cl.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			if !h.sessionActive(cl) {
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "session ended"))
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
