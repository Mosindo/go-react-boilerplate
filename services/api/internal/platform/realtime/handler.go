package realtime

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

const (
	authTimeout  = 10 * time.Second
	pingInterval = 25 * time.Second
	writeTimeout = 10 * time.Second
	sendBuffer   = 64
)

type clientMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades GET /realtime to a WebSocket. The client must send
// {"type":"auth","token":"<access token>"} as its first frame: tokens never
// travel in URLs (which end up in logs), and because no ambient credential
// (cookie) is used, cross-site WebSocket hijacking is not possible, so origin
// checks are not required for browser clients.
func Handler(hub *Hub, auth *middleware.Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		conn.SetReadLimit(4096)
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		identity, ok := authenticate(ctx, conn, auth)
		if !ok {
			_ = conn.Close(websocket.StatusPolicyViolation, "unauthorized")
			return
		}

		var once sync.Once
		cl := &client{
			userID: identity.UserID,
			send:   make(chan []byte, sendBuffer),
		}
		cl.close = func() {
			once.Do(func() {
				cancel()
			})
		}
		if !hub.register(cl) {
			_ = conn.Close(websocket.StatusTryAgainLater, "too many connections")
			return
		}
		defer hub.unregister(cl)

		ready, _ := json.Marshal(Event{Type: "ready", Data: gin.H{"userId": identity.UserID}})
		if err := write(ctx, conn, ready); err != nil {
			return
		}

		// Reader: only keeps the connection healthy; clients send nothing else.
		go func() {
			defer cl.close()
			for {
				if _, _, err := conn.Read(ctx); err != nil {
					return
				}
			}
		}()

		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				_ = conn.Close(websocket.StatusNormalClosure, "")
				return
			case payload := <-cl.send:
				if err := write(ctx, conn, payload); err != nil {
					return
				}
			case <-ticker.C:
				pingCtx, pingCancel := context.WithTimeout(ctx, writeTimeout)
				err := conn.Ping(pingCtx)
				pingCancel()
				if err != nil {
					return
				}
			}
		}
	}
}

func authenticate(ctx context.Context, conn *websocket.Conn, auth *middleware.Authenticator) (middleware.Identity, bool) {
	authCtx, cancel := context.WithTimeout(ctx, authTimeout)
	defer cancel()
	_, data, err := conn.Read(authCtx)
	if err != nil {
		return middleware.Identity{}, false
	}
	var msg clientMessage
	if err := json.Unmarshal(data, &msg); err != nil || msg.Type != "auth" {
		return middleware.Identity{}, false
	}
	identity, err := auth.Authenticate(authCtx, msg.Token)
	if err != nil {
		if err != middleware.ErrInvalidToken {
			log.Printf(`{"event":"realtime_auth_error","error":%q}`, err.Error())
		}
		return middleware.Identity{}, false
	}
	return identity, true
}

func write(ctx context.Context, conn *websocket.Conn, payload []byte) error {
	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return conn.Write(writeCtx, websocket.MessageText, payload)
}
