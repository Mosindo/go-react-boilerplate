package realtime

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout  = 10 * time.Second
	writeTimeout = 10 * time.Second
	pongWait     = 60 * time.Second
	pingInterval = 25 * time.Second
	maxInbound   = 4096
)

// Authenticator validates an access token and returns the user id and the
// moment the token expires.
type Authenticator func(token string) (userID string, expiresAt time.Time, err error)

type inbound struct {
	Type  string `json:"type"`
	Token string `json:"token,omitempty"`
}

// Origin checks are intentionally permissive: the socket is authenticated by
// an access token sent as the first frame (never by cookies), so a foreign
// page cannot ride an existing session (no CSWSH risk). Tokens are not put in
// the URL to keep them out of access logs.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// Handler upgrades the connection and runs the read/write pumps.
// Protocol: client sends {"type":"auth","token":"<access token>"} first,
// server answers {"type":"ready"}; then server pushes events. The server
// closes the socket when the access token expires so the client reconnects
// with a fresh token.
func (h *Hub) Handler(authenticate Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		conn.SetReadLimit(maxInbound)

		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
		var first inbound
		if err := conn.ReadJSON(&first); err != nil || first.Type != "auth" {
			closeWith(conn, websocket.ClosePolicyViolation, "auth required")
			return
		}
		userID, expiresAt, err := authenticate(first.Token)
		if err != nil {
			closeWith(conn, 4401, "invalid token")
			return
		}

		var once sync.Once
		done := make(chan struct{})
		cl := &client{userID: userID, send: make(chan []byte, 64)}
		cl.close = func() {
			once.Do(func() {
				close(done)
				_ = conn.Close()
			})
		}
		h.register(cl)
		defer func() {
			h.unregister(cl)
			cl.close()
		}()

		ready, _ := json.Marshal(Event{Type: "ready"})
		_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := conn.WriteMessage(websocket.TextMessage, ready); err != nil {
			return
		}

		go writePump(conn, cl, done, expiresAt)
		readPump(conn, cl)
	}
}

func readPump(conn *websocket.Conn, cl *client) {
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		var msg inbound
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		if msg.Type == "ping" {
			pong, _ := json.Marshal(Event{Type: "pong"})
			select {
			case cl.send <- pong:
			default:
			}
		}
	}
}

func writePump(conn *websocket.Conn, cl *client, done <-chan struct{}, expiresAt time.Time) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	untilExpiry := time.Until(expiresAt)
	if untilExpiry < time.Second {
		untilExpiry = time.Second
	}
	expiry := time.NewTimer(untilExpiry)
	defer expiry.Stop()

	for {
		select {
		case <-done:
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
		case <-expiry.C:
			closeWith(conn, 4401, "token expired")
			cl.close()
			return
		}
	}
}

func closeWith(conn *websocket.Conn, code int, reason string) {
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
	_ = conn.Close()
}
