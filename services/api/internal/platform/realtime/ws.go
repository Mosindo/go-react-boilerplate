package realtime

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

const (
	ticketAudience = "realtime"
	ticketTTL      = 30 * time.Second
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingEvery      = 50 * time.Second
	maxFrameBytes  = 1024
)

// upgrader accepts any Origin: the socket is authenticated by a short-lived,
// single-purpose ticket that a third-party page cannot obtain.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

type ticketClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

// Gateway issues tickets and serves the WebSocket endpoint.
type Gateway struct {
	hub    *Hub
	secret []byte
}

func NewGateway(hub *Hub, secret []byte) *Gateway {
	return &Gateway{hub: hub, secret: secret}
}

func (g *Gateway) RegisterRoutes(r gin.IRouter, requireUser gin.HandlerFunc) {
	r.POST("/realtime/ticket", requireUser, g.issueTicket)
	r.GET("/realtime/ws", g.serve)
}

func (g *Gateway) issueTicket(c *gin.Context) {
	now := time.Now()
	claims := ticketClaims{
		UserID: c.GetString("userID"),
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{ticketAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ticketTTL)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(g.secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not issue ticket"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": signed, "expiresInSeconds": int(ticketTTL.Seconds())})
}

func (g *Gateway) userFromTicket(raw string) (string, error) {
	claims := &ticketClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return g.secret, nil
	}, jwt.WithAudience(ticketAudience), jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.UserID == "" {
		return "", errors.New("invalid ticket")
	}
	return claims.UserID, nil
}

func (g *Gateway) serve(c *gin.Context) {
	userID, err := g.userFromTicket(c.Query("ticket"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid ticket"})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	cl := g.hub.register(userID)
	defer g.hub.unregister(userID, cl)

	conn.SetReadLimit(maxFrameBytes)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			// Clients never send application data; reading drives pong/close handling.
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(pingEvery)
	defer ticker.Stop()
	defer conn.Close()

	_ = conn.WriteJSON(Event{Type: "ready"})
	for {
		select {
		case ev := <-cl.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteJSON(ev); err != nil {
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
