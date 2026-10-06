package realtime

import (
	"context"
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
	pingInterval = 50 * time.Second
	writeWait    = 10 * time.Second
	maxFrameSize = 4096
)

// Handler upgrades GET /ws to a WebSocket. Browsers and React Native cannot
// set an Authorization header on the handshake, and tokens in URLs leak into
// logs, so the client authenticates with its first frame: {"type":"auth","token":"..."}.
type Handler struct {
	hub       *Hub
	jwtSecret []byte
	validator middleware.SessionValidator
	upgrader  websocket.Upgrader
}

func NewHandler(hub *Hub, jwtSecret []byte, validator middleware.SessionValidator, allowedOrigins []string) *Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	return &Handler{
		hub:       hub,
		jwtSecret: jwtSecret,
		validator: validator,
		upgrader: websocket.Upgrader{
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
		},
	}
}

func (h *Handler) Serve(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(maxFrameSize)

	userID, ok := h.authenticate(c.Request.Context(), conn)
	if !ok {
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "unauthorized"),
			time.Now().Add(writeWait))
		return
	}

	cl := h.hub.register(userID)
	defer h.hub.unregister(cl)

	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := conn.WriteJSON(Event{Type: "ready"}); err != nil {
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		})
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
		case <-done:
			return
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
		}
	}
}

func (h *Handler) authenticate(ctx context.Context, conn *websocket.Conn) (string, bool) {
	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return "", false
	}
	var frame struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &frame); err != nil || frame.Type != "auth" || frame.Token == "" {
		return "", false
	}
	identity, err := middleware.Authenticate(ctx, h.jwtSecret, h.validator, frame.Token)
	if err != nil {
		return "", false
	}
	return identity.UserID, true
}
