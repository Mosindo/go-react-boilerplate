package realtime

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/jwtauth"
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout   = 5 * time.Second
	writeWait     = 10 * time.Second
	pongWait      = 60 * time.Second
	pingInterval  = 30 * time.Second
	readLimit     = 4096
	recheckEveryN = 10 // re-verify the session every N pings (~5 min)
)

type Handler struct {
	hub      *Hub
	secret   []byte
	sessions middleware.SessionChecker
	origins  map[string]struct{}
	upgrader websocket.Upgrader
}

func NewHandler(hub *Hub, secret []byte, sessions middleware.SessionChecker, allowedOrigins []string) *Handler {
	h := &Handler{hub: hub, secret: secret, sessions: sessions, origins: map[string]struct{}{}}
	for _, o := range allowedOrigins {
		h.origins[strings.ToLower(strings.TrimSpace(o))] = struct{}{}
	}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin:     h.checkOrigin,
	}
	return h
}

// checkOrigin: native clients send no Origin header and are allowed (the auth
// message is the real gate). Browsers must come from ALLOWED_ORIGINS; when that
// list is empty only same-host origins are accepted.
func (h *Handler) checkOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if _, ok := h.origins[strings.ToLower(origin)]; ok {
		return true
	}
	if len(h.origins) == 0 {
		u, err := url.Parse(origin)
		return err == nil && strings.EqualFold(u.Host, r.Host)
	}
	return false
}

func (h *Handler) Serve(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade already wrote the HTTP error (403 on bad origin, 400 otherwise).
		return
	}
	conn.SetReadLimit(readLimit)
	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))

	var first clientFrame
	if err := conn.ReadJSON(&first); err != nil || first.Type != "auth" || first.Token == "" {
		closeWith(conn, CloseUnauthorized, "unauthorized")
		return
	}
	claims, err := jwtauth.Parse(h.secret, first.Token)
	if err != nil || !httpx.IsUUID(claims.UserID) || !httpx.IsUUID(claims.SessionID) {
		closeWith(conn, CloseUnauthorized, "unauthorized")
		return
	}
	userID, sid := strings.ToLower(claims.UserID), strings.ToLower(claims.SessionID)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	active, err := h.sessions.SessionActive(ctx, sid, userID)
	cancel()
	if err != nil || !active {
		closeWith(conn, CloseUnauthorized, "unauthorized")
		return
	}

	cl := newClient(h.hub, conn, userID, sid)
	if !h.hub.register(cl) {
		closeWith(conn, CloseTooMany, "too many connections")
		return
	}
	defer h.hub.unregister(cl)

	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := conn.WriteJSON(map[string]string{"type": "ready"}); err != nil {
		cl.shutdown(websocket.CloseInternalServerErr, "")
		return
	}

	go h.writePump(cl)

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		var frame clientFrame
		if err := conn.ReadJSON(&frame); err != nil {
			cl.shutdown(websocket.CloseNormalClosure, "")
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		if frame.Type == "ping" {
			select {
			case cl.send <- []byte(`{"type":"pong"}`):
			default:
			}
		}
	}
}

func (h *Handler) writePump(cl *client) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	ticks := 0
	for {
		select {
		case <-cl.done:
			return
		case msg := <-cl.send:
			_ = cl.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := cl.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				cl.shutdown(websocket.CloseInternalServerErr, "")
				return
			}
		case <-ticker.C:
			ticks++
			if ticks%recheckEveryN == 0 {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				active, err := h.sessions.SessionActive(ctx, cl.sid, cl.userID)
				cancel()
				if err == nil && !active {
					cl.shutdown(CloseUnauthorized, "session ended")
					return
				}
			}
			_ = cl.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := cl.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				cl.shutdown(websocket.CloseInternalServerErr, "")
				return
			}
		}
	}
}

func closeWith(conn *websocket.Conn, code int, reason string) {
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
	_ = conn.Close()
}
