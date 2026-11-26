package realtime

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
)

type Handler struct {
	hub            *Hub
	tickets        *Tickets
	allowedOrigins map[string]struct{}
	upgrader       websocket.Upgrader
	onConnect      func(userID string)
}

// NewHandler wires the WebSocket endpoints. Native apps send no Origin header; browsers do,
// and browser origins must be listed in ALLOWED_ORIGINS (or be the API's own host).
func NewHandler(hub *Hub, tickets *Tickets, allowedOrigins []string, onConnect func(userID string)) *Handler {
	h := &Handler{hub: hub, tickets: tickets, allowedOrigins: make(map[string]struct{}), onConnect: onConnect}
	for _, o := range allowedOrigins {
		h.allowedOrigins[o] = struct{}{}
	}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin:     h.checkOrigin,
	}
	return h
}

func (h *Handler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if _, ok := h.allowedOrigins[origin]; ok {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}

func (h *Handler) RegisterRoutes(r gin.IRouter, requireUser gin.HandlerFunc) {
	r.POST("/realtime/ticket", requireUser, h.IssueTicket)
	r.GET("/realtime", h.Connect)
}

func (h *Handler) IssueTicket(c *gin.Context) {
	value, err := h.tickets.Issue(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Une erreur interne est survenue.", "code": "internal_error"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, gin.H{"ticket": value, "expiresInSeconds": int(h.tickets.ttl.Seconds())})
}

func (h *Handler) Connect(c *gin.Context) {
	userID, ok := h.tickets.Redeem(c.Query("ticket"))
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Ticket temps réel invalide.", "code": "invalid_ticket"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	cl := &client{userID: userID, send: make(chan []byte, 32)}
	h.hub.register(cl)
	if h.onConnect != nil {
		h.onConnect(userID)
	}

	go writePump(conn, cl)
	readPump(conn)
	h.hub.unregister(cl)
}

// readPump only services control frames; clients send nothing meaningful over the socket.
func readPump(conn *websocket.Conn) {
	defer conn.Close()
	conn.SetReadLimit(512)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func writePump(conn *websocket.Conn, cl *client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ready","data":null}`))
	for {
		select {
		case msg, ok := <-cl.send:
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
