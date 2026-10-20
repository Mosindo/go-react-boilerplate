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
	idleTimeout  = 70 * time.Second
	writeTimeout = 10 * time.Second
	maxFrameSize = 4096
)

// TokenVerifier turns an access token into a user id.
type TokenVerifier func(ctx context.Context, token string) (userID string, err error)

type frame struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades to WebSocket. The first frame must authenticate; the
// token never travels in the URL.
func (h *Hub) Handler(verify TokenVerifier, allowedOrigins []string) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     originChecker(allowedOrigins),
	}

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetReadLimit(maxFrameSize)

		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
		var first frame
		if err := conn.ReadJSON(&first); err != nil || first.Type != "auth" {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "auth required"), time.Now().Add(time.Second))
			return
		}
		userID, err := verify(c.Request.Context(), first.Token)
		if err != nil {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid token"), time.Now().Add(time.Second))
			return
		}

		cl := h.register(userID)
		defer h.unregister(userID, cl)

		_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := conn.WriteJSON(Event{"type": "ready"}); err != nil {
			return
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				_ = conn.SetReadDeadline(time.Now().Add(idleTimeout))
				var msg frame
				if err := conn.ReadJSON(&msg); err != nil {
					return
				}
				if msg.Type == "ping" {
					pong, _ := json.Marshal(Event{"type": "pong"})
					select {
					case cl.send <- pong:
					default:
					}
				}
			}
		}()

		for {
			select {
			case payload, ok := <-cl.send:
				if !ok {
					return
				}
				_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
				if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}
}

// originChecker accepts native clients (no Origin header) and listed web origins.
func originChecker(allowed []string) func(*http.Request) bool {
	set := make(map[string]struct{}, len(allowed))
	for _, origin := range allowed {
		set[strings.TrimSpace(origin)] = struct{}{}
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		if _, ok := set[origin]; ok {
			return true
		}
		u, err := url.Parse(origin)
		return err == nil && u.Host == r.Host
	}
}
