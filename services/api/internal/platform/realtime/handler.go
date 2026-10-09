package realtime

import (
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pingPeriod = 25 * time.Second
	pongWait   = 60 * time.Second
	maxReadLen = 512
)

// The handshake is authenticated by a single-use ticket and no cookie is ever
// involved, so cross-site WebSocket hijacking is not applicable: any origin may connect.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// TicketHandler issues a ticket for the authenticated user (POST /ws/ticket).
func TicketHandler(tickets *Tickets) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, err := tickets.Issue(c.GetString("userID"))
		if err != nil {
			httpx.Internal(c)
			return
		}
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, gin.H{"ticket": value})
	}
}

// ConnectHandler upgrades GET /ws?ticket=... to a WebSocket.
func ConnectHandler(hub *Hub, tickets *Tickets) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := tickets.Redeem(c.Query("ticket"))
		if !ok {
			httpx.Fail(c, http.StatusUnauthorized, httpx.CodeUnauthorized, "invalid or expired ticket")
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		cl := hub.register(userID)
		go writePump(hub, cl, conn)
		readPump(hub, cl, conn)
	}
}

// readPump discards client frames; it exists to process control frames and
// detect disconnection.
func readPump(hub *Hub, cl *client, conn *websocket.Conn) {
	defer func() {
		hub.unregister(cl)
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

func writePump(hub *Hub, cl *client, conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
		hub.unregister(cl)
	}()
	for {
		select {
		case msg, ok := <-cl.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
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
