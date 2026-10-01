package realtime

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const ticketTTL = 30 * time.Second

// Tickets are short-lived single-use credentials exchanged for a WebSocket connection, so the
// long-lived access token never appears in a URL (and therefore never in proxy or access logs).
type Tickets struct {
	mu    sync.Mutex
	items map[string]ticket
}

type ticket struct {
	userID  string
	expires time.Time
}

func NewTickets() *Tickets { return &Tickets{items: make(map[string]ticket)} }

func (t *Tickets) Issue(userID string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	id := hex.EncodeToString(buf)
	now := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, v := range t.items {
		if now.After(v.expires) {
			delete(t.items, k)
		}
	}
	t.items[id] = ticket{userID: userID, expires: now.Add(ticketTTL)}
	return id, nil
}

// Redeem consumes a ticket, returning its user when valid.
func (t *Tickets) Redeem(id string) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	tk, ok := t.items[id]
	if !ok {
		return "", false
	}
	delete(t.items, id)
	if time.Now().After(tk.expires) {
		return "", false
	}
	return tk.userID, true
}

// RegisterRoutes mounts POST /realtime/ticket (authenticated) and GET /ws (ticket-authenticated).
func RegisterRoutes(r gin.IRouter, hub *Hub, tickets *Tickets, requireUser gin.HandlerFunc, allowedOrigins []string) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     originChecker(allowedOrigins),
	}
	r.POST("/realtime/ticket", requireUser, middleware.NewLimiter(60, time.Minute).ByUser(), func(c *gin.Context) {
		id, err := tickets.Issue(c.GetString("userID"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "une erreur interne est survenue", "code": "internal"})
			return
		}
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, gin.H{"ticket": id, "expiresIn": int(ticketTTL.Seconds())})
	})
	r.GET("/ws", func(c *gin.Context) {
		userID, ok := tickets.Redeem(c.Query("ticket"))
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "ticket invalide ou expiré", "code": "unauthorized"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		hub.Serve(userID, conn)
	})
}

// originChecker accepts native clients (no Origin header), same-host browsers and configured origins.
func originChecker(allowed []string) func(*http.Request) bool {
	set := map[string]bool{}
	for _, o := range allowed {
		set[strings.TrimRight(o, "/")] = true
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		if set[strings.TrimRight(origin, "/")] {
			return true
		}
		u, err := url.Parse(origin)
		return err == nil && strings.EqualFold(u.Host, r.Host)
	}
}
