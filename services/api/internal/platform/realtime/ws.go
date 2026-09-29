package realtime

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

const (
	ticketTTL     = 60 * time.Second
	ticketPurpose = "ws"
	writeWait     = 10 * time.Second
	pongWait      = 60 * time.Second
	pingPeriod    = 50 * time.Second
	maxReadBytes  = 4096
)

type Handler struct {
	hub      *Hub
	secret   []byte
	upgrader websocket.Upgrader
}

// NewHandler builds the WebSocket handlers. allowedOrigins restricts browser origins;
// native mobile clients send no Origin header and are always accepted (they authenticate by ticket).
func NewHandler(hub *Hub, secret []byte, allowedOrigins []string) *Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}
	return &Handler{
		hub:    hub,
		secret: secret,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				_, ok := allowed[origin]
				return ok
			},
		},
	}
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.POST("/ws/ticket", requireUser, h.Ticket)
	r.GET("/ws", h.Serve)
}

type ticketClaims struct {
	UserID  string `json:"uid"`
	Purpose string `json:"pur"`
	jwt.RegisteredClaims
}

// Ticket issues a 60-second, single-purpose token used only to open the WebSocket.
func (h *Handler) Ticket(c *gin.Context) {
	now := time.Now()
	claims := ticketClaims{
		UserID:  c.GetString("userID"),
		Purpose: ticketPurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ticketTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(h.secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not issue ticket"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": signed, "expiresInSeconds": int(ticketTTL.Seconds())})
}

func (h *Handler) userFromTicket(raw string) (string, bool) {
	claims := &ticketClaims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return h.secret, nil
	}, jwt.WithExpirationRequired())
	if err != nil || !tok.Valid || claims.Purpose != ticketPurpose || claims.UserID == "" {
		return "", false
	}
	return claims.UserID, true
}

// Serve upgrades GET /ws?ticket=... and pumps events to the client. The socket is
// server-to-client only: inbound frames are ignored (sending happens over REST so that
// every rule is enforced by the same code path).
func (h *Handler) Serve(c *gin.Context) {
	userID, ok := h.userFromTicket(c.Query("ticket"))
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid ticket"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	cl := &client{userID: userID, send: make(chan []byte, sendBuffer)}
	h.hub.add(cl)

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn.SetReadLimit(maxReadBytes)
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		})
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		h.hub.remove(cl)
		_ = conn.Close()
	}()
	for {
		select {
		case msg := <-cl.send:
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
