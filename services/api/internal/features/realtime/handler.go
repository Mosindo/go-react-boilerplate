// Package realtime exposes the WebSocket endpoint backed by platform/realtime.
//
// Protocol: POST /realtime/ticket (bearer auth) returns a single-use ticket
// valid 60 s; the client then opens GET /realtime?ticket=... . The server only
// pushes JSON events ({"type": "...", "data": {...}}); clients send nothing
// but control frames.
package realtime

import (
	"net/http"
	"slices"
	"time"

	"example.com/api/internal/platform/httpx"
	hub "example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 25 * time.Second
)

type Handler struct {
	hub      *hub.Hub
	upgrader websocket.Upgrader
}

func NewHandler(h *hub.Hub, allowedOrigins []string) *Handler {
	return &Handler{
		hub: h,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			// Authentication relies on the single-use ticket (never a cookie),
			// so cross-site WebSocket hijacking is not possible. Browser origins
			// are still restricted when an allow-list is configured.
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || len(allowedOrigins) == 0 || slices.Contains(allowedOrigins, origin)
			},
		},
	}
}

func (h *Handler) Ticket(c *gin.Context) {
	ticket, err := h.hub.IssueTicket(httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "realtime.ticket", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"ticket": ticket, "expiresIn": 60})
}

func (h *Handler) Connect(c *gin.Context) {
	userID, ok := h.hub.RedeemTicket(c.Query("ticket"))
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, "invalid_ticket", "invalid or expired realtime ticket")
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // the upgrader already wrote the HTTP error
	}
	client := h.hub.Register(userID)
	go writePump(conn, client)
	readPump(conn)
	h.hub.Unregister(client)
}

func readPump(conn *websocket.Conn) {
	defer conn.Close()
	conn.SetReadLimit(512)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func writePump(conn *websocket.Conn, client *hub.Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()
	for {
		select {
		case msg, ok := <-client.Send:
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
