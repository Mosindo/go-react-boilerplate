package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout = 5 * time.Second
	pongWait    = 70 * time.Second
	pingEvery   = 30 * time.Second
	writeWait   = 10 * time.Second
	maxFrame    = 4 << 10
)

// Authenticator turns an access token into a user id, or fails.
type Authenticator func(ctx context.Context, token string) (userID string, ok bool)

// Handler upgrades GET /ws. Browsers cannot set headers on WebSockets and tokens
// must stay out of URLs (logs), so the client authenticates with its first frame:
// {"type":"auth","token":"<access token>"}.
func (h *Hub) Handler(authenticate Authenticator, allowedOrigins []string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // native mobile clients send no Origin
			}
			if _, ok := allowed[origin]; ok {
				return true
			}
			u, err := url.Parse(origin)
			return err == nil && u.Host == r.Host
		},
	}

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetReadLimit(maxFrame)

		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
		var hello struct {
			Type  string `json:"type"`
			Token string `json:"token"`
		}
		if err := conn.ReadJSON(&hello); err != nil || hello.Type != "auth" {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "auth required"), time.Now().Add(writeWait))
			return
		}
		userID, ok := authenticate(c.Request.Context(), hello.Token)
		if !ok {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "unauthorized"), time.Now().Add(writeWait))
			return
		}

		cl := &client{send: make(chan []byte, 32)}
		h.add(userID, cl)
		defer h.remove(userID, cl)

		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })

		done := make(chan struct{})
		go func() { // reader: only keeps the connection alive and detects close
			defer close(done)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
				_ = conn.SetReadDeadline(time.Now().Add(pongWait))
			}
		}()

		ready, _ := json.Marshal(Event{Type: "ready"})
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, ready); err != nil {
			return
		}

		ticker := time.NewTicker(pingEvery)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-cl.send:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
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
			case <-done:
				return
			}
		}
	}
}
