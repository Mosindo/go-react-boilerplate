package chat

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout    = 5 * time.Second
	pongWait       = 70 * time.Second
	pingEvery      = 30 * time.Second
	writeWait      = 10 * time.Second
	sessionRecheck = 5 * time.Minute
	maxClientMsg   = 4096
)

// WSHandler upgrades /ws. Authentication happens through the first frame
// ({"type":"auth","token":"<access token>"}) so tokens never appear in URLs or logs.
type WSHandler struct {
	hub      *realtime.Hub
	secret   []byte
	sessions middleware.SessionChecker
	upgrader websocket.Upgrader
}

func NewWSHandler(hub *realtime.Hub, secret []byte, sessions middleware.SessionChecker) *WSHandler {
	return &WSHandler{
		hub: hub, secret: secret, sessions: sessions,
		// Auth is carried by the first frame, not by cookies, so cross-site
		// connections cannot act on behalf of a user.
		upgrader: websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 4096, CheckOrigin: func(*http.Request) bool { return true }},
	}
}

type clientFrame struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

func (h *WSHandler) Serve(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(maxClientMsg)

	userID, sessionID, ok := h.authenticate(conn)
	if !ok {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4401, "unauthorized"), time.Now().Add(writeWait))
		return
	}

	client := h.hub.Register(userID)
	defer h.hub.Unregister(userID, client)
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := conn.WriteJSON(realtime.Event{Type: "ready"}); err != nil {
		return
	}

	done := make(chan struct{})
	go func() { // reader: keep-alive frames only
		defer close(done)
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
			var f clientFrame
			if json.Unmarshal(data, &f) == nil && f.Type == "ping" {
				select {
				case client.Out <- []byte(`{"type":"pong"}`):
				default:
				}
			}
		}
	}()

	ping := time.NewTicker(pingEvery)
	recheck := time.NewTicker(sessionRecheck)
	defer ping.Stop()
	defer recheck.Stop()
	for {
		select {
		case msg := <-client.Out:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ping.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-recheck.C:
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			active, err := h.sessions.IsSessionActive(ctx, sessionID, userID)
			cancel()
			if err == nil && !active {
				_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4401, "session ended"), time.Now().Add(writeWait))
				return
			}
		case <-done:
			return
		}
	}
}

func (h *WSHandler) authenticate(conn *websocket.Conn) (string, string, bool) {
	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	_, data, err := conn.ReadMessage()
	if err != nil {
		return "", "", false
	}
	var f clientFrame
	if json.Unmarshal(data, &f) != nil || f.Type != "auth" {
		return "", "", false
	}
	userID, sessionID, err := middleware.ParseAccessToken(h.secret, f.Token)
	if err != nil {
		return "", "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	active, err := h.sessions.IsSessionActive(ctx, sessionID, userID)
	if err != nil {
		log.Printf(`{"event":"ws_auth_error","error":%q}`, err.Error())
		return "", "", false
	}
	return userID, sessionID, active
}
