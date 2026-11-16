package realtime

import (
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

type Handler struct {
	hub      *Hub
	upgrader websocket.Upgrader
}

// NewHandler builds the WebSocket handlers. Native apps send no Origin header;
// browsers must come from ALLOWED_ORIGINS when that list is configured.
func NewHandler(hub *Hub, allowedOrigins []string) *Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}
	return &Handler{
		hub: hub,
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
	value, err := h.hub.IssueTicket(c.GetString("userID"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not issue ticket", "code": "internal"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": value, "expiresInSeconds": int(ticketTTL.Seconds())})
}

// Connect upgrades the connection after redeeming the ticket.
func (h *Handler) Connect(c *gin.Context) {
	userID, ok := h.hub.RedeemTicket(c.Query("ticket"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid ticket", "code": "unauthorized"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	cl := &client{userID: userID, send: make(chan []byte, sendBuffer)}
	h.hub.register(cl)

	go writePump(conn, cl)
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

func writePump(conn *websocket.Conn, cl *client) {
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
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
