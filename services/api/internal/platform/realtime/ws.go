package realtime

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout     = 10 * time.Second
	pingInterval    = 30 * time.Second
	pongWait        = 75 * time.Second
	writeWait       = 10 * time.Second
	revalidateEvery = time.Minute
	maxClientFrame  = 4096
)

type authMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades GET /ws to a WebSocket. The access token is sent in the first
// frame ({"type":"auth","token":"..."}) rather than the URL, so it never lands in logs.
// The server only pushes events; clients send messages through the REST API.
func Handler(hub *Hub, secret []byte, sessions middleware.SessionValidator, allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // native mobile clients send no Origin
			}
			if _, ok := allowed[origin]; ok {
				return true
			}
			u, err := url.Parse(origin)
			return err == nil && strings.EqualFold(u.Host, r.Host)
		},
	}

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetReadLimit(maxClientFrame)

		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
		var first authMessage
		if err := conn.ReadJSON(&first); err != nil || first.Type != "auth" {
			closeWith(conn, websocket.ClosePolicyViolation, "auth required")
			return
		}
		userID, sessionID, err := middleware.ParseAccessToken(secret, first.Token)
		if err != nil {
			closeWith(conn, websocket.ClosePolicyViolation, "invalid token")
			return
		}
		active, err := sessions.SessionActive(c.Request.Context(), sessionID, userID)
		if err != nil || !active {
			closeWith(conn, websocket.ClosePolicyViolation, "invalid session")
			return
		}

		client := NewClient(userID)
		hub.Register(client)
		defer hub.Unregister(client)

		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		})

		done := make(chan struct{})
		go func() {
			defer close(done)
			ping := time.NewTicker(pingInterval)
			revalidate := time.NewTicker(revalidateEvery)
			defer ping.Stop()
			defer revalidate.Stop()

			write := func(v any) bool {
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				return conn.WriteJSON(v) == nil
			}
			if !write(Event{Type: "ready", Data: nil}) {
				return
			}
			for {
				select {
				case ev, ok := <-client.Send:
					if !ok {
						closeWith(conn, websocket.CloseNormalClosure, "session ended")
						return
					}
					if !write(ev) {
						return
					}
				case <-ping.C:
					_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
					if conn.WriteMessage(websocket.PingMessage, nil) != nil {
						return
					}
				case <-revalidate.C:
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					ok, err := sessions.SessionActive(ctx, sessionID, userID)
					cancel()
					if err == nil && !ok {
						closeWith(conn, websocket.CloseNormalClosure, "session ended")
						return
					}
				}
			}
		}()

		// Read loop: only keeps the connection alive and detects disconnects.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		}
		hub.Unregister(client) // closes client.Send so the writer exits
		_ = conn.Close()
		<-done
	}
}

func closeWith(conn *websocket.Conn, code int, reason string) {
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
}
