package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout     = 5 * time.Second
	writeTimeout    = 10 * time.Second
	pongWait        = 70 * time.Second
	pingInterval    = 30 * time.Second
	sessionRecheck  = 2 * time.Minute
	maxInboundBytes = 4096
	closeUnauth     = 4401
)

// SocketHandler upgrades /ws connections. Authentication happens with the first frame so the
// access token never appears in a URL (and therefore never in logs).
type SocketHandler struct {
	hub      *realtime.Hub
	secret   []byte
	checker  middleware.SessionChecker
	upgrader websocket.Upgrader
}

func NewSocketHandler(hub *realtime.Hub, secret []byte, checker middleware.SessionChecker, allowedOrigins []string) *SocketHandler {
	allowed := map[string]struct{}{}
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	return &SocketHandler{
		hub: hub, secret: secret, checker: checker,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true // native clients send no Origin header
				}
				if _, ok := allowed[origin]; ok {
					return true
				}
				u, err := url.Parse(origin)
				return err == nil && u.Host == r.Host
			},
		},
	}
}

type inbound struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

func (h *SocketHandler) Serve(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(maxInboundBytes)

	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	var first inbound
	if err := conn.ReadJSON(&first); err != nil || first.Type != "auth" {
		closeWith(conn, closeUnauth, "unauthorized")
		return
	}
	identity, ok := middleware.Authenticate(c.Request.Context(), h.secret, h.checker, first.Token)
	if !ok {
		closeWith(conn, closeUnauth, "unauthorized")
		return
	}

	client := realtime.NewClient(identity.UserID)
	h.hub.Register(client)
	defer h.hub.Unregister(client)

	_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := conn.WriteJSON(realtime.Event{Type: "ready"}); err != nil {
		return
	}

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })

	readDone := make(chan struct{})
	pongs := make(chan struct{}, 4)
	go func() {
		defer close(readDone)
		for {
			var msg inbound
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
			if msg.Type == "ping" {
				select {
				case pongs <- struct{}{}:
				default:
				}
			}
		}
	}()

	ping := time.NewTicker(pingInterval)
	recheck := time.NewTicker(sessionRecheck)
	defer ping.Stop()
	defer recheck.Stop()

	for {
		var err error
		_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		select {
		case payload := <-client.Outbound():
			err = conn.WriteMessage(websocket.TextMessage, payload)
		case <-pongs:
			pong, _ := json.Marshal(realtime.Event{Type: "pong"})
			err = conn.WriteMessage(websocket.TextMessage, pong)
		case <-ping.C:
			err = conn.WriteMessage(websocket.PingMessage, nil)
		case <-recheck.C:
			active, cerr := h.checker.IsSessionActive(context.Background(), identity.UserID, identity.SessionID)
			if cerr == nil && !active {
				closeWith(conn, closeUnauth, "session ended")
				return
			}
		case <-client.Done():
			closeWith(conn, websocket.CloseGoingAway, "closing")
			return
		case <-readDone:
			return
		}
		if err != nil {
			return
		}
	}
}

func closeWith(conn *websocket.Conn, code int, reason string) {
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
}
