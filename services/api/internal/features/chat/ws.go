package chat

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout  = 10 * time.Second
	pongWait     = 70 * time.Second
	pingInterval = 30 * time.Second
	writeWait    = 10 * time.Second
	maxFrameSize = 4096
)

// SocketHandler upgrades /ws connections. The access token is sent in the first
// frame ({"type":"auth","token":"..."}) rather than in the URL so it never ends
// up in access logs. The socket is receive-only for application data: messages
// are sent through the REST API, which owns validation and authorization.
type SocketHandler struct {
	hub      *realtime.Hub
	secret   []byte
	upgrader websocket.Upgrader
}

func NewSocketHandler(hub *realtime.Hub, secret []byte, allowedOrigins []string) *SocketHandler {
	allowed := map[string]struct{}{}
	for _, o := range allowedOrigins {
		allowed[strings.TrimRight(o, "/")] = struct{}{}
	}
	return &SocketHandler{
		hub:    hub,
		secret: secret,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true // native mobile clients send no Origin
				}
				if _, ok := allowed[strings.TrimRight(origin, "/")]; ok {
					return true
				}
				u, err := url.Parse(origin)
				return err == nil && u.Host == r.Host
			},
		},
	}
}

type clientFrame struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

func (h *SocketHandler) Serve(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(maxFrameSize)

	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	var first clientFrame
	if err := conn.ReadJSON(&first); err != nil || first.Type != "auth" {
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "auth required"), time.Now().Add(writeWait))
		return
	}
	claims, ok := middleware.ParseAccessToken(h.secret, first.Token)
	if !ok {
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid token"), time.Now().Add(writeWait))
		return
	}

	client := h.hub.Register(claims.UserID)
	defer h.hub.Unregister(client)

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// Single writer goroutine: gorilla connections allow one concurrent writer.
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		defer conn.Close()
		write := func(v any) error {
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			return conn.WriteJSON(v)
		}
		if err := write(realtime.Event{Type: "ready"}); err != nil {
			return
		}
		for {
			select {
			case ev := <-client.Send:
				if err := write(ev); err != nil {
					return
				}
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-client.Done():
				return
			}
		}
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		var f clientFrame
		if json.Unmarshal(data, &f) == nil && f.Type == "ping" {
			client.Send <- realtime.Event{Type: "pong"}
		}
	}
}
