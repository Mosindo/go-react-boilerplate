package realtime

import (
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/middleware"
	platform "example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 70 * time.Second
	pingPeriod = 30 * time.Second
)

type Handler struct {
	hub      *platform.Hub
	tickets  *platform.Tickets
	upgrader websocket.Upgrader
}

// NewHandler builds the WebSocket endpoint. Browsers must come from an allowed
// origin; native apps send no Origin header and rely on the one-time ticket.
func NewHandler(hub *platform.Hub, tickets *platform.Tickets, allowedOrigins []string) *Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	return &Handler{
		hub:     hub,
		tickets: tickets,
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

func (h *Handler) Ticket(c *gin.Context) {
	ticket, err := h.tickets.Issue(c.GetString("userID"))
	if err != nil {
		httpx.Fail(c, "realtime.ticket", http.StatusInternalServerError, "internal error", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": ticket, "expiresIn": 60})
}

func (h *Handler) Connect(c *gin.Context) {
	userID, ok := h.tickets.Redeem(c.Query("ticket"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired ticket"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // Upgrade already replied
	}

	client := platform.NewClient(userID)
	h.hub.Register(client)
	defer h.hub.Unregister(client)
	defer conn.Close()

	// Reader: the server only needs pongs and detects disconnects; clients send nothing meaningful.
	conn.SetReadLimit(512)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		}
	}()

	ping := time.NewTicker(pingPeriod)
	defer ping.Stop()
	for {
		select {
		case msg, open := <-client.Messages():
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !open {
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "bye"))
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ping.C:
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
	r.POST("/realtime/ticket", requireUser, middleware.NewLimiter(60, time.Minute).ByUser(), h.Ticket)
	r.GET("/realtime", middleware.NewLimiter(60, time.Minute).ByIP(), h.Connect)
}
