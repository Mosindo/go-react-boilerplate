package realtime

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
	maxReadLen = 1024
)

type Handler struct {
	hub      *Hub
	upgrader websocket.Upgrader
}

// NewHandler builds the WebSocket endpoints. Authentication is ticket based:
// the origin check is unnecessary because a ticket can only be obtained with a
// valid bearer token, which browsers do not attach automatically.
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub, upgrader: websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin:     func(*http.Request) bool { return true },
	}}
}

func (h *Handler) Ticket(c *gin.Context) {
	t, err := h.hub.IssueTicket(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "internal"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": t})
}

func (h *Handler) Connect(c *gin.Context) {
	userID, ok := h.hub.RedeemTicket(c.Query("ticket"))
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid ticket", "code": "unauthorized"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := h.hub.Register(userID)
	defer h.hub.Unregister(client)
	defer conn.Close()

	conn.SetReadLimit(maxReadLen)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })

	done := make(chan struct{})
	go func() { // reader: only drains pings/close frames; clients send via REST
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		}
	}()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case msg := <-client.Send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.POST("/ws/ticket", requireUser, h.Ticket)
	r.GET("/ws", h.Connect)
}
