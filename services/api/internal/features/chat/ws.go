package chat

import (
	"net/http"
	"time"

	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout = 5 * time.Second
	pongWait    = 60 * time.Second
	pingEvery   = 25 * time.Second
	writeWait   = 10 * time.Second
	maxFrame    = 4096
)

// WSHandler upgrades /ws connections. Authentication happens with the first
// frame ({"type":"auth","token":"<access token>"}) so tokens never appear in
// URLs or access logs.
type WSHandler struct {
	hub      *realtime.Hub
	secret   []byte
	upgrader websocket.Upgrader
}

func NewWSHandler(hub *realtime.Hub, secret []byte, allowedOrigins []string) *WSHandler {
	allowed := map[string]bool{}
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return &WSHandler{
		hub:    hub,
		secret: secret,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Native mobile clients send no Origin; browsers must be allow-listed.
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || allowed[origin]
			},
		},
	}
}

type authFrame struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

func (h *WSHandler) Serve(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(maxFrame)

	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	var frame authFrame
	if err := conn.ReadJSON(&frame); err != nil || frame.Type != "auth" {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4401, "unauthorized"), time.Now().Add(writeWait))
		return
	}
	claims, err := middleware.ParseAccessToken(h.secret, frame.Token)
	if err != nil {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4401, "unauthorized"), time.Now().Add(writeWait))
		return
	}

	client := h.hub.Register(claims.UserID)
	defer h.hub.Unregister(client)
	_ = conn.WriteJSON(map[string]string{"type": "ready"})

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// Reader: clients send nothing meaningful after auth; this loop only
	// processes control frames (pong/close) and detects disconnects.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		}
	}()

	ticker := time.NewTicker(pingEvery)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-client.Send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "resync"))
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
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
