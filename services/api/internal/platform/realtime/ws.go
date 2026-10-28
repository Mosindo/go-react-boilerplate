package realtime

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout  = 5 * time.Second
	pongWait     = 60 * time.Second
	writeWait    = 10 * time.Second
	maxFrameSize = 4096
)

// closeUnauthorized is the application close code for a failed or missing auth frame.
const closeUnauthorized = 4401

type clientFrame struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades GET /ws. Authentication happens in the first frame so tokens never appear in URLs or logs.
func (h *Hub) Handler(jwtSecret []byte, allowedOrigins []string) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     originChecker(allowedOrigins),
	}
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return // Upgrade already wrote the HTTP error.
		}
		conn.SetReadLimit(maxFrameSize)

		userID, ok := authenticate(conn, jwtSecret)
		if !ok {
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(closeUnauthorized, "unauthorized"), time.Now().Add(writeWait))
			_ = conn.Close()
			return
		}

		cl := &client{userID: userID, send: make(chan []byte, sendBuffer)}
		h.add(cl)
		defer h.remove(cl)

		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteJSON(Event{Type: "ready"}); err != nil {
			_ = conn.Close()
			return
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			readLoop(conn, cl)
		}()
		writeLoop(conn, cl, done)
	}
}

func authenticate(conn *websocket.Conn, secret []byte) (string, bool) {
	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return "", false
	}
	var frame clientFrame
	if err := json.Unmarshal(raw, &frame); err != nil || frame.Type != "auth" {
		return "", false
	}
	claims, err := middleware.ParseAccessToken(secret, frame.Token)
	if err != nil {
		return "", false
	}
	return claims.UserID, true
}

func readLoop(conn *websocket.Conn, cl *client) {
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		var frame clientFrame
		if json.Unmarshal(raw, &frame) == nil && frame.Type == "ping" {
			payload, _ := json.Marshal(Event{Type: "pong"})
			select {
			case cl.send <- payload:
			default:
			}
		}
	}
}

func writeLoop(conn *websocket.Conn, cl *client, done <-chan struct{}) {
	defer conn.Close()
	for {
		select {
		case payload, ok := <-cl.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "slow consumer"))
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

// originChecker allows non-browser clients (no Origin header, e.g. React Native) and, for browsers,
// only the configured origins or the same host.
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
