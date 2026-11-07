package chat

import (
	"context"
	"encoding/json"
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
)

// SessionChecker mirrors middleware.SessionChecker.
type SessionChecker = middleware.SessionChecker

// WSHandler upgrades /ws to a WebSocket. The first client frame must be {"type":"auth","token":"<access token>"}
// (tokens never appear in URLs/logs). Server -> client events: message, read, match, unmatch, notification.
type WSHandler struct {
	hub      *realtime.Hub
	secret   []byte
	check    SessionChecker
	upgrader websocket.Upgrader
}

func NewWSHandler(hub *realtime.Hub, secret []byte, check SessionChecker, allowedOrigins []string) *WSHandler {
	allowed := map[string]bool{}
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return &WSHandler{
		hub: hub, secret: secret, check: check,
		upgrader: websocket.Upgrader{
			ReadBufferSize: 1024, WriteBufferSize: 1024,
			// Native apps send no Origin header; browsers must be on the allow-list.
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || allowed[origin]
			},
		},
	}
}

func (h *WSHandler) Serve(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(4096)

	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	var hello struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if _, raw, err := conn.ReadMessage(); err != nil || json.Unmarshal(raw, &hello) != nil || hello.Type != "auth" {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "auth required"), time.Now().Add(writeWait))
		return
	}
	userID, sessionID, err := middleware.ParseAccessToken(h.secret, hello.Token)
	if err == nil && h.check != nil {
		var active bool
		active, err = h.check(context.Background(), sessionID, userID)
		if err == nil && !active {
			err = context.Canceled
		}
	}
	if err != nil {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "unauthorized"), time.Now().Add(writeWait))
		return
	}

	client := h.hub.Register(userID)
	defer h.hub.Unregister(client)
	_ = conn.WriteJSON(realtime.Event{Type: "ready", Data: nil})

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })
	done := make(chan struct{})
	go func() { // reader: only needed to process control frames and detect disconnects
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
		case msg := <-client.Send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
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
