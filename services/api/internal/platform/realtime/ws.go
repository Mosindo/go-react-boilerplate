package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout  = 10 * time.Second
	writeTimeout = 10 * time.Second
	pongTimeout  = 60 * time.Second
	pingInterval = 25 * time.Second
	maxInbound   = 4096
	sendBuffer   = 64
)

// Authenticator validates an access token and returns the user id.
type Authenticator func(ctx context.Context, token string) (string, error)

type authMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	// Authentication is carried in-band (first frame), never by cookies, so a
	// cross-origin page cannot ride an existing session. Any origin is fine.
	CheckOrigin: func(*http.Request) bool { return true },
}

// Handler upgrades GET /ws. The client must send {"type":"auth","token":"..."}
// as its first frame; tokens are kept out of URLs (and thus out of logs).
func (h *Hub) Handler(authenticate Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		conn.SetReadLimit(maxInbound)

		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
		var msg authMessage
		if err := conn.ReadJSON(&msg); err != nil || msg.Type != "auth" || msg.Token == "" {
			closeWith(conn, websocket.ClosePolicyViolation, "auth required")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		userID, err := authenticate(ctx, msg.Token)
		cancel()
		if err != nil || userID == "" {
			closeWith(conn, 4001, "unauthorized")
			return
		}

		cl := &client{userID: userID, send: make(chan []byte, sendBuffer)}
		if !h.register(cl) {
			closeWith(conn, websocket.CloseTryAgainLater, "too many connections")
			return
		}

		ready, _ := json.Marshal(Event{Type: "ready", Data: gin.H{"userId": userID}})
		cl.send <- ready

		go writePump(conn, cl)
		readPump(conn)
		h.unregister(cl)
	}
}

func readPump(conn *websocket.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongTimeout))
	})
	for {
		// Clients only send keep-alive frames; content is ignored.
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
	}
}

func writePump(conn *websocket.Conn, cl *client) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()
	for {
		select {
		case payload, ok := <-cl.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "session ended"))
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func closeWith(conn *websocket.Conn, code int, reason string) {
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
	_ = conn.Close()
}
