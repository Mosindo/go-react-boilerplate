package realtime

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout  = 10 * time.Second
	pongWait     = 60 * time.Second
	pingInterval = 50 * time.Second
	writeWait    = 10 * time.Second
	maxFrameSize = 4096
	sendBuffer   = 32
)

type authFrame struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades GET /ws. The access token is sent as the first frame
// ({"type":"auth","token":"..."}) rather than in the URL, so it never lands in access logs.
func Handler(hub *Hub, jwtSecret []byte, allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimSpace(o)] = struct{}{}
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
			return err == nil && strings.EqualFold(u.Host, r.Host)
		},
	}

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetReadLimit(maxFrameSize)

		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
		var frame authFrame
		if err := conn.ReadJSON(&frame); err != nil || frame.Type != "auth" {
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "auth required"), time.Now().Add(writeWait))
			return
		}
		claims, err := middleware.ParseAccessToken(jwtSecret, frame.Token)
		if err != nil {
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid token"), time.Now().Add(writeWait))
			return
		}

		cl := &client{send: make(chan Event, sendBuffer)}
		hub.register(claims.UserID, cl)
		defer hub.unregister(claims.UserID, cl)

		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteJSON(Event{Type: "ready"}); err != nil {
			return
		}

		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		})

		// Reader: the protocol is server-push only; reading drains control frames and detects close.
		closed := make(chan struct{})
		go func() {
			defer close(closed)
			for {
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
			case ev := <-cl.send:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteJSON(ev); err != nil {
					return
				}
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-closed:
				return
			}
		}
	}
}
