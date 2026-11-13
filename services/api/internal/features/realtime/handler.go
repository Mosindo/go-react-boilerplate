// Package realtime exposes the WebSocket endpoint that streams server events
// (new messages, read receipts, matches, notifications) to a member.
package realtime

import (
	"context"
	"net/http"
	"time"

	"example.com/api/internal/platform/middleware"
	platformrealtime "example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout  = 10 * time.Second
	pongWait     = 60 * time.Second
	pingInterval = 25 * time.Second
	writeWait    = 10 * time.Second
	// CloseTokenExpired tells the client to refresh its token and reconnect.
	CloseTokenExpired = 4001
	CloseUnauthorized = 4003
)

type authMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type Handler struct {
	hub      *platformrealtime.Hub
	secret   []byte
	sessions middleware.SessionValidator
	upgrader websocket.Upgrader
}

// NewHandler builds the endpoint. Native apps send no Origin header; browser
// origins must be listed in ALLOWED_ORIGINS when that list is configured.
func NewHandler(hub *platformrealtime.Hub, secret []byte, sessions middleware.SessionValidator, allowedOrigins []string) *Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
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

func closeWith(conn *websocket.Conn, code int, reason string) {
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(writeWait))
	_ = conn.Close()
}

// Connect upgrades the connection, then expects {"type":"auth","token":"…"}
// as the first frame. Tokens are never put in the URL, so they do not end up
// in access logs.
func (h *Handler) Connect(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(4096)
	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))

	var msg authMessage
	if err := conn.ReadJSON(&msg); err != nil || msg.Type != "auth" {
		closeWith(conn, CloseUnauthorized, "authentication required")
		return
	}
	claims, err := middleware.ParseAccessToken(h.secret, msg.Token)
	if err != nil {
		closeWith(conn, CloseUnauthorized, "invalid token")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	active, err := h.sessions.IsSessionActive(ctx, claims.SessionID, claims.UserID)
	cancel()
	if err != nil || !active {
		closeWith(conn, CloseUnauthorized, "session expired")
		return
	}

	client := platformrealtime.NewClient(claims.UserID)
	h.hub.Register(client)
	defer h.hub.Unregister(client)

	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := conn.WriteJSON(platformrealtime.Event{Type: "ready", At: time.Now().UTC()}); err != nil {
		_ = conn.Close()
		return
	}

	done := make(chan struct{})
	go h.writeLoop(conn, client, claims.ExpiresAt, done)

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		// Client frames are only keep-alives; actions go through REST so
		// they share validation, authorization and rate limits.
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	}
	close(done)
}

func (h *Handler) writeLoop(conn *websocket.Conn, client *platformrealtime.Client, expiresAt time.Time, done <-chan struct{}) {
	ping := time.NewTicker(pingInterval)
	defer ping.Stop()
	expiry := time.NewTimer(time.Until(expiresAt))
	defer expiry.Stop()
	defer conn.Close()

	for {
		select {
		case payload := <-client.Send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ping.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-expiry.C:
			closeWith(conn, CloseTokenExpired, "token expired")
			return
		case <-client.Closed():
			closeWith(conn, CloseUnauthorized, "session ended")
			return
		case <-done:
			return
		}
	}
}
