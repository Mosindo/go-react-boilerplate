package realtime

import (
	"context"
	"net/http"
	"time"

	"example.com/api/internal/platform/authtoken"
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = 25 * time.Second
	maxInbound = 512
)

// Handler upgrades authenticated requests to WebSocket connections.
type Handler struct {
	hub      *Hub
	secret   []byte
	sessions middleware.SessionChecker
	upgrader websocket.Upgrader
}

// NewHandler authenticates connections with a short-lived realtime ticket
// (obtained through POST /realtime/ticket) passed as ?ticket=. Tickets are
// not cookies, so cross-site WebSocket hijacking is not possible; allowed
// origins are still enforced for browsers when configured.
func NewHandler(hub *Hub, secret []byte, sessions middleware.SessionChecker, allowedOrigins []string) *Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	return &Handler{
		hub:      hub,
		secret:   secret,
		sessions: sessions,
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

func (h *Handler) Serve(c *gin.Context) {
	claims, err := authtoken.Parse(h.secret, c.Query("ticket"), authtoken.TypeRealtime)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid ticket", "code": "unauthorized"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	active, err := h.sessions.IsSessionActive(ctx, claims.SessionID, claims.UserID)
	cancel()
	if err != nil || !active {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired", "code": "unauthorized"})
		return
	}
	c.Set("userID", claims.UserID)

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // the upgrader already replied
	}
	cl := h.hub.register(claims.UserID)
	go h.writeLoop(conn, cl)
	h.readLoop(conn, cl)
}

// readLoop only handles control frames; clients never send business data
// over the socket (everything goes through the REST API and its validation).
func (h *Handler) readLoop(conn *websocket.Conn, cl *client) {
	defer func() {
		h.hub.unregister(cl)
		_ = conn.Close()
	}()
	conn.SetReadLimit(maxInbound)
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

func (h *Handler) writeLoop(conn *websocket.Conn, cl *client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()
	_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ready"}`))
	for {
		select {
		case <-cl.closed:
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		case payload := <-cl.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				cl.close()
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				cl.close()
				return
			}
		}
	}
}
