package realtime

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/api/internal/platform/authtoken"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout  = 10 * time.Second
	writeTimeout = 10 * time.Second
	pongTimeout  = 60 * time.Second
	pingInterval = 25 * time.Second
	maxInbound   = 4096
)

type authMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades GET /realtime to a WebSocket. The client must send
// {"type":"auth","token":"<access token>"} as its first frame: tokens never appear
// in URLs or access logs. The connection is closed when the access token expires,
// so clients reconnect with a fresh token.
func Handler(hub *Hub, secret []byte, allowedOrigins []string, onConnect func(userID string)) gin.HandlerFunc {
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
				return true // native mobile clients do not send Origin
			}
			if _, ok := allowed[origin]; ok {
				return true
			}
			if len(allowed) == 0 {
				// Without an explicit allow-list only same-host origins are accepted.
				u, err := url.Parse(origin)
				return err == nil && strings.EqualFold(u.Host, r.Host) || isLocalOrigin(origin)
			}
			return false
		},
	}

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetReadLimit(maxInbound)

		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
		var msg authMessage
		if err := conn.ReadJSON(&msg); err != nil || msg.Type != "auth" {
			closeWith(conn, websocket.ClosePolicyViolation, "auth required")
			return
		}
		claims, err := authtoken.Parse(secret, msg.Token)
		if err != nil {
			closeWith(conn, 4001, "invalid token")
			return
		}
		userID := claims.UserID
		c.Set("userID", userID)

		cl := hub.register(userID)
		defer hub.unregister(userID, cl)
		if onConnect != nil {
			onConnect(userID)
		}

		ready, _ := json.Marshal(Event{Type: "ready", Data: gin.H{"userId": userID}, At: time.Now().UTC()})
		_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := conn.WriteMessage(websocket.TextMessage, ready); err != nil {
			return
		}

		// Reader: only pongs/close frames are expected; it detects dead peers.
		done := make(chan struct{})
		_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongTimeout))
		})
		go func() {
			defer close(done)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()

		expiry := time.NewTimer(time.Until(claims.ExpiresAt.Time))
		defer expiry.Stop()
		ping := time.NewTicker(pingInterval)
		defer ping.Stop()

		for {
			select {
			case payload, ok := <-cl.send:
				if !ok {
					closeWith(conn, websocket.CloseNormalClosure, "closed")
					return
				}
				_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
				if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
					return
				}
			case <-ping.C:
				_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-expiry.C:
				closeWith(conn, 4001, "token expired")
				return
			case <-done:
				return
			case <-c.Request.Context().Done():
				return
			}
		}
	}
}

func closeWith(conn *websocket.Conn, code int, reason string) {
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
}

func isLocalOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1"
}
