package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout  = 5 * time.Second
	pingInterval = 30 * time.Second
	pongWait     = 70 * time.Second
	writeWait    = 10 * time.Second
	readLimit    = 4096
)

// Authenticator resolves an access token to a user id.
type Authenticator func(ctx context.Context, token string) (userID string, err error)

type incoming struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades GET /ws. The first client frame must be {"type":"auth","token":...};
// tokens are never accepted in the URL so they cannot leak into logs.
func (h *Hub) Handler(authenticate Authenticator, allowedOrigins []string) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     originChecker(allowedOrigins),
	}
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return // Upgrade already replied.
		}
		defer conn.Close()
		conn.SetReadLimit(readLimit)

		userID, ok := h.handshake(c.Request.Context(), conn, authenticate)
		if !ok {
			return
		}
		cl := &client{userID: userID, send: make(chan []byte, sendBuffer), done: make(chan struct{})}
		h.register(cl)
		defer h.unregister(cl)

		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteJSON(Event{Type: "ready"}); err != nil {
			return
		}

		go h.readLoop(conn, cl)
		h.writeLoop(conn, cl)
	}
}

func (h *Hub) handshake(ctx context.Context, conn *websocket.Conn, authenticate Authenticator) (string, bool) {
	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	var msg incoming
	if err := conn.ReadJSON(&msg); err != nil || msg.Type != "auth" || msg.Token == "" {
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "auth required"), time.Now().Add(writeWait))
		return "", false
	}
	userID, err := authenticate(ctx, msg.Token)
	if err != nil {
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "unauthorized"), time.Now().Add(writeWait))
		return "", false
	}
	return userID, true
}

func (h *Hub) readLoop(conn *websocket.Conn, cl *client) {
	defer cl.close()
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		var msg incoming
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		if msg.Type == "ping" {
			payload, _ := json.Marshal(Event{Type: "pong"})
			select {
			case cl.send <- payload:
			default:
			}
		}
	}
}

func (h *Hub) writeLoop(conn *websocket.Conn, cl *client) {
	ticker := time.NewTicker(pingInterval)
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
		case <-cl.done:
			return
		}
	}
}

// originChecker accepts requests without an Origin header (native apps) and
// browsers whose origin is explicitly allowed or same-host.
func originChecker(allowed []string) func(*http.Request) bool {
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		set[strings.TrimRight(o, "/")] = struct{}{}
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		if _, ok := set[strings.TrimRight(origin, "/")]; ok {
			return true
		}
		u, err := url.Parse(origin)
		return err == nil && strings.EqualFold(u.Host, r.Host)
	}
}
