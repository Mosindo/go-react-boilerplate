package realtime

import (
	"net/http"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout = 5 * time.Second
	pongWait    = 60 * time.Second
	pingEvery   = 25 * time.Second
	writeWait   = 10 * time.Second
)

type authMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades GET /ws. The client must send {"type":"auth","token":"<access token>"}
// as its first frame within 5 seconds, so tokens never travel in URLs.
func (h *Hub) Handler(secret []byte) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// Auth is a bearer token sent in-band (no cookies), so cross-site hijacking is not applicable.
		CheckOrigin: func(*http.Request) bool { return true },
	}
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetReadLimit(4096)

		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
		var msg authMessage
		if err := conn.ReadJSON(&msg); err != nil || msg.Type != "auth" {
			_ = conn.WriteJSON(Event{Type: "error", Data: "auth required"})
			return
		}
		userID, _, err := middleware.ParseAccessToken(secret, msg.Token)
		if err != nil {
			_ = conn.WriteJSON(Event{Type: "error", Data: "invalid token"})
			return
		}

		cl := &client{send: make(chan []byte, 32)}
		h.register(userID, cl)
		defer h.unregister(userID, cl)

		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		})
		_ = conn.WriteJSON(Event{Type: "ready"})

		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()

		ticker := time.NewTicker(pingEvery)
		defer ticker.Stop()
		for {
			select {
			case payload := <-cl.send:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
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
