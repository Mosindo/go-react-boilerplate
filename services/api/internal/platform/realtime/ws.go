package realtime

import (
	"net/http"
	"strings"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
	maxReadLen = 512
)

// TicketHandler issues a one-time WebSocket ticket for the authenticated user.
func (h *Hub) TicketHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		value, err := h.IssueTicket(httpx.UserID(c))
		if err != nil {
			httpx.Fail(c, "realtime.ticket", err)
			return
		}
		httpx.NoStore(c)
		c.JSON(http.StatusOK, gin.H{"ticket": value, "expiresInSeconds": int(ticketTTL.Seconds())})
	}
}

// ConnectHandler upgrades the request to a WebSocket after redeeming the
// ticket. Native apps send no Origin header; browsers must come from an allowed origin.
func (h *Hub) ConnectHandler(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[strings.TrimSpace(origin)] = struct{}{}
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			_, ok := allowed[origin]
			return ok
		},
	}

	return func(c *gin.Context) {
		userID, ok := h.redeemTicket(c.Query("ticket"))
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired ticket", "code": "unauthorized"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		cl := h.register(userID)
		go h.writePump(conn, cl)
		h.readPump(conn, cl)
	}
}

// readPump only services control frames: clients never send application data.
func (h *Hub) readPump(conn *websocket.Conn, cl *client) {
	defer func() {
		h.unregister(cl)
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
	}
}

func (h *Hub) writePump(conn *websocket.Conn, cl *client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()
	for {
		select {
		case event, ok := <-cl.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := conn.WriteJSON(event); err != nil {
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
