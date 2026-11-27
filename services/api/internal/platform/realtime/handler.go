package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout        = 10 * time.Second
	writeTimeout       = 10 * time.Second
	pongTimeout        = 60 * time.Second
	pingInterval       = 25 * time.Second
	sessionRecheck     = time.Minute
	maxIncomingMessage = 4096
	sendBuffer         = 64
)

type authFrame struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades GET /ws. The first client frame must be
// {"type":"auth","token":"<access token>"}; the token is never put in the URL
// so it cannot leak through proxies or access logs.
func Handler(hub *Hub, secret []byte, sessions middleware.SessionChecker, allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[strings.TrimSpace(origin)] = struct{}{}
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Native apps send no Origin. Authentication is token-based (no
			// cookies), so cross-site hijacking is not possible; the allow
			// list is still honoured when configured.
			if origin == "" || len(allowed) == 0 {
				return true
			}
			_, ok := allowed[origin]
			return ok
		},
	}

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		serve(hub, conn, secret, sessions)
	}
}

func serve(hub *Hub, conn *websocket.Conn, secret []byte, sessions middleware.SessionChecker) {
	defer conn.Close()
	conn.SetReadLimit(maxIncomingMessage)

	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	var frame authFrame
	if err := conn.ReadJSON(&frame); err != nil || frame.Type != "auth" || frame.Token == "" {
		writeClose(conn, websocket.ClosePolicyViolation, "auth required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	identity, err := middleware.Authenticate(ctx, secret, sessions, frame.Token)
	cancel()
	if err != nil {
		writeClose(conn, 4401, "invalid token")
		return
	}

	var once sync.Once
	done := make(chan struct{})
	cl := &client{
		userID: identity.UserID,
		send:   make(chan []byte, sendBuffer),
		close:  func() { once.Do(func() { close(done) }) },
	}
	hub.register(cl)
	defer hub.unregister(cl)

	ready, _ := json.Marshal(Event{Type: "ready", At: time.Now().UTC()})
	_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := conn.WriteMessage(websocket.TextMessage, ready); err != nil {
		return
	}

	_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongTimeout))
	})

	// Reader: clients only send keep-alives; anything else is ignored.
	go func() {
		defer cl.close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
		}
	}()

	ping := time.NewTicker(pingInterval)
	defer ping.Stop()
	recheck := time.NewTicker(sessionRecheck)
	defer recheck.Stop()

	for {
		select {
		case <-done:
			writeClose(conn, websocket.CloseNormalClosure, "bye")
			return
		case msg := <-cl.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ping.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-recheck.C:
			if sessions == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			active, err := sessions.IsSessionActive(ctx, identity.SessionID, identity.UserID)
			cancel()
			if err == nil && !active {
				writeClose(conn, 4401, "session ended")
				return
			}
		}
	}
}

func writeClose(conn *websocket.Conn, code int, reason string) {
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
}
