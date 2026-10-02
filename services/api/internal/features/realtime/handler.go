// Package realtime exposes the WebSocket endpoint. Clients obtain a 60 second,
// single-purpose ticket over authenticated REST, then open the socket with it,
// so long-lived access tokens never appear in URLs or proxy logs.
package realtime

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/middleware"
	rt "example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/token"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 70 * time.Second
	pingPeriod = 30 * time.Second
)

type Handler struct {
	hub      *rt.Hub
	secret   []byte
	sessions middleware.SessionChecker
	upgrader websocket.Upgrader
}

func NewHandler(hub *rt.Hub, secret []byte, sessions middleware.SessionChecker, allowedOrigins []string) *Handler {
	return &Handler{
		hub: hub, secret: secret, sessions: sessions,
		upgrader: websocket.Upgrader{
			ReadBufferSize: 1024, WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				// Native clients send no Origin; browsers must be on the allow-list.
				return origin == "" || slices.Contains(allowedOrigins, origin)
			},
		},
	}
}

func (h *Handler) Ticket(c *gin.Context) {
	t, err := token.Sign(h.secret, httpx.UserID(c), c.GetString("sessionID"), token.TypeWS, time.Minute)
	if err != nil {
		httpx.Internal(c, "realtime.ticket", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": t, "expiresInSeconds": 60})
}

func (h *Handler) Connect(c *gin.Context) {
	claims, err := token.Parse(h.secret, strings.TrimSpace(c.Query("ticket")), token.TypeWS)
	if err != nil {
		httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "invalid ticket")
		return
	}
	if ok, err := h.sessions.SessionActive(c.Request.Context(), claims.SessionID, claims.UserID); err != nil || !ok {
		httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "session ended")
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // Upgrade already wrote the HTTP error
	}
	client := h.hub.Register(claims.UserID)
	go h.writePump(conn, client)
	h.readPump(conn, claims.UserID, client)
}

// readPump discards client frames (commands go through REST) and detects disconnects.
func (h *Handler) readPump(conn *websocket.Conn, userID string, client *rt.Client) {
	defer func() {
		h.hub.Unregister(userID, client)
		_ = conn.Close()
	}()
	conn.SetReadLimit(512)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Handler) writePump(conn *websocket.Conn, client *rt.Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()
	for {
		select {
		case msg, ok := <-client.Send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
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
		}
	}
}
