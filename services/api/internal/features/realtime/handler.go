// Package realtime exposes the WebSocket endpoint backed by platform/realtime.Hub.
package realtime

import (
	"context"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	hub "example.com/api/internal/platform/realtime"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

const (
	pingInterval = 25 * time.Second
	writeTimeout = 10 * time.Second
)

type Handler struct {
	hub            *hub.Hub
	originPatterns []string
}

// NewHandler builds the handler. originPatterns are host patterns (e.g. "app.example.com")
// allowed for browser clients; native clients send no Origin header.
func NewHandler(h *hub.Hub, originPatterns []string) *Handler {
	return &Handler{hub: h, originPatterns: originPatterns}
}

// Ticket issues a single-use 30s ticket for the WebSocket handshake.
func (h *Handler) Ticket(c *gin.Context) {
	t, err := h.hub.IssueTicket(httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "realtime.ticket", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": t, "expiresInSeconds": 30})
}

// Connect upgrades to a WebSocket. Clients only receive events; they send nothing but pings.
func (h *Handler) Connect(c *gin.Context) {
	userID, ok := h.hub.RedeemTicket(c.Query("ticket"))
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "invalid or expired ticket")
		return
	}
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: h.originPatterns})
	if err != nil {
		return
	}
	conn.SetReadLimit(1024)

	client := h.hub.Register(userID)
	defer h.hub.Unregister(client)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Reader: drains control frames and detects disconnects.
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	_ = write(ctx, conn, []byte(`{"type":"ready","payload":null}`))

	for {
		select {
		case <-ctx.Done():
			conn.Close(websocket.StatusNormalClosure, "")
			return
		case msg, open := <-client.Messages():
			if !open {
				conn.Close(websocket.StatusPolicyViolation, "session ended")
				return
			}
			if err := write(ctx, conn, msg); err != nil {
				return
			}
		case <-ticker.C:
			pingCtx, cancelPing := context.WithTimeout(ctx, writeTimeout)
			err := conn.Ping(pingCtx)
			cancelPing()
			if err != nil {
				return
			}
		}
	}
}

func write(ctx context.Context, conn *websocket.Conn, msg []byte) error {
	wctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return conn.Write(wctx, websocket.MessageText, msg)
}
