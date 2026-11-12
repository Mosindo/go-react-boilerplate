package realtime

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

const (
	authTimeout  = 10 * time.Second
	pingInterval = 25 * time.Second
	writeTimeout = 10 * time.Second
)

type authMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades to WebSocket. The client must send
// {"type":"auth","token":"<access token>"} as its first frame; tokens are
// deliberately not accepted in the URL so they never end up in access logs.
func Handler(hub *Hub, verifier *middleware.TokenVerifier, allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		opts := &websocket.AcceptOptions{OriginPatterns: allowedOrigins}
		if len(allowedOrigins) == 0 {
			// Native mobile clients do not send an Origin; browsers in local
			// development do. Without an explicit allow-list, accept any origin:
			// authentication still happens in-band with a bearer token.
			opts.InsecureSkipVerify = true
		}
		conn, err := websocket.Accept(c.Writer, c.Request, opts)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		conn.SetReadLimit(4096)

		ctx := c.Request.Context()
		identity, ok := authenticate(ctx, conn, verifier)
		if !ok {
			_ = conn.Close(websocket.StatusPolicyViolation, "authentication required")
			return
		}
		c.Set("userID", identity.UserID)

		events, unsubscribe := hub.Subscribe(identity.UserID)
		defer unsubscribe()

		if err := writeJSON(ctx, conn, Event{Type: "ready"}); err != nil {
			return
		}

		// Reader goroutine: drains client frames (only pings/close are expected)
		// and cancels the connection context when the socket goes away.
		readCtx := conn.CloseRead(ctx)

		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-readCtx.Done():
				return
			case event, open := <-events:
				if !open {
					return
				}
				if err := writeJSON(readCtx, conn, event); err != nil {
					return
				}
			case <-ticker.C:
				pingCtx, cancel := context.WithTimeout(readCtx, writeTimeout)
				err := conn.Ping(pingCtx)
				cancel()
				if err != nil {
					return
				}
			}
		}
	}
}

func authenticate(ctx context.Context, conn *websocket.Conn, verifier *middleware.TokenVerifier) (middleware.Identity, bool) {
	authCtx, cancel := context.WithTimeout(ctx, authTimeout)
	defer cancel()
	kind, data, err := conn.Read(authCtx)
	if err != nil || kind != websocket.MessageText {
		return middleware.Identity{}, false
	}
	var msg authMessage
	if err := json.Unmarshal(data, &msg); err != nil || msg.Type != "auth" {
		return middleware.Identity{}, false
	}
	identity, err := verifier.Verify(authCtx, msg.Token)
	if err != nil {
		return middleware.Identity{}, false
	}
	return identity, true
}

func writeJSON(ctx context.Context, conn *websocket.Conn, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf(`{"event":"realtime_marshal_error","type":%q}`, event.Type)
		return nil
	}
	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return conn.Write(writeCtx, websocket.MessageText, payload)
}
