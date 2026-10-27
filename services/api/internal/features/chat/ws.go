package chat

import (
	"net/http"
	"time"

	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout  = 5 * time.Second
	pongWait     = 60 * time.Second
	pingInterval = 25 * time.Second
	writeWait    = 10 * time.Second
	maxFrameSize = 4096
)

// SocketHandler serves GET /ws. The token is sent as the first frame
// ({"type":"auth","token":"..."}) rather than in the URL, which would leak it
// into proxy and access logs. The socket is server-push only: all writes go
// through the REST API so that every rule is enforced in one place.
type SocketHandler struct {
	hub      *realtime.Hub
	secret   []byte
	upgrader websocket.Upgrader
}

func NewSocketHandler(hub *realtime.Hub, secret []byte, allowedOrigins []string) *SocketHandler {
	allowed := map[string]struct{}{}
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	return &SocketHandler{
		hub:    hub,
		secret: secret,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Native apps send no Origin header. Browsers do, and must be allow-listed
			// (protects against cross-site WebSocket hijacking).
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

func (h *SocketHandler) Serve(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(maxFrameSize)

	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	var first struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := conn.ReadJSON(&first); err != nil || first.Type != "auth" {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "auth required"), time.Now().Add(writeWait))
		return
	}
	claims, err := middleware.ParseAccessToken(h.secret, first.Token)
	if err != nil {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid token"), time.Now().Add(writeWait))
		return
	}

	client := h.hub.Register(claims.UserID)
	defer h.hub.Unregister(client)

	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := conn.WriteJSON(map[string]string{"type": "ready"}); err != nil {
		return
	}

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			// Incoming frames are ignored (keep-alive only); any read error ends the session.
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		}
	}()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case data, ok := <-client.Send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
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
