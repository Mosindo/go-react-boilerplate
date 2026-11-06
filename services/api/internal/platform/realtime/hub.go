// Package realtime fans out server events to connected WebSocket clients.
//
// The hub is in-process: it serves a single API instance. To scale out, put a
// pub/sub (Redis, NATS) behind the Publisher interface.
package realtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

// Event is the envelope pushed to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher is what features depend on to push events to a user.
type Publisher interface {
	Publish(userID string, event Event)
}

// NopPublisher discards events (tests, or features without realtime).
type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}

const (
	ticketTTL      = 30 * time.Second
	sendBuffer     = 32
	maxConnsPerUsr = 5
)

type client struct {
	send chan []byte
}

type Hub struct {
	mu      sync.Mutex
	clients map[string]map[*client]struct{}
	tickets map[string]ticket
	origins []string
	now     func() time.Time
}

type ticket struct {
	userID  string
	expires time.Time
}

// NewHub creates a hub. allowedOrigins is passed to the WebSocket handshake
// check (empty means same-origin and non-browser clients only).
func NewHub(allowedOrigins []string) *Hub {
	return &Hub{
		clients: make(map[string]map[*client]struct{}),
		tickets: make(map[string]ticket),
		origins: allowedOrigins,
		now:     time.Now,
	}
}

// IssueTicket returns a single-use, short-lived ticket for the WebSocket
// handshake. Tickets avoid putting the long-lived JWT in a URL.
func (h *Hub) IssueTicket(userID string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	id := hex.EncodeToString(buf)
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	for k, t := range h.tickets {
		if now.After(t.expires) {
			delete(h.tickets, k)
		}
	}
	h.tickets[id] = ticket{userID: userID, expires: now.Add(ticketTTL)}
	return id, nil
}

func (h *Hub) redeem(id string) (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	t, ok := h.tickets[id]
	if !ok {
		return "", false
	}
	delete(h.tickets, id)
	if h.now().After(t.expires) {
		return "", false
	}
	return t.userID, true
}

// Publish delivers an event to every connection of userID. Slow consumers are dropped.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for cl := range h.clients[userID] {
		select {
		case cl.send <- payload:
		default:
			// Buffer full: the client is too slow; it will resync on reconnect.
			close(cl.send)
			delete(h.clients[userID], cl)
		}
	}
}

func (h *Hub) add(userID string, cl *client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.clients[userID] = set
	}
	if len(set) >= maxConnsPerUsr {
		return false
	}
	set[cl] = struct{}{}
	return true
}

func (h *Hub) remove(userID string, cl *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[userID]; ok {
		if _, present := set[cl]; present {
			delete(set, cl)
			close(cl.send)
		}
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
}

// Handle upgrades GET /ws?ticket=... to a WebSocket and streams events.
func (h *Hub) Handle(c *gin.Context) {
	userID, ok := h.redeem(c.Query("ticket"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid ticket", "code": "unauthorized"})
		return
	}
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: h.originPatterns()})
	if err != nil {
		return
	}
	cl := &client{send: make(chan []byte, sendBuffer)}
	if !h.add(userID, cl) {
		_ = conn.Close(websocket.StatusTryAgainLater, "too many connections")
		return
	}
	defer h.remove(userID, cl)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Reader: clients send nothing meaningful; reading handles pings/close frames.
	go func() {
		defer cancel()
		conn.SetReadLimit(1024)
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}()

	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = conn.Close(websocket.StatusNormalClosure, "")
			return
		case msg, open := <-cl.send:
			if !open {
				_ = conn.Close(websocket.StatusTryAgainLater, "slow consumer")
				return
			}
			wctx, wcancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(wctx, websocket.MessageText, msg)
			wcancel()
			if err != nil {
				return
			}
		case <-ping.C:
			pctx, pcancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(pctx)
			pcancel()
			if err != nil {
				return
			}
		}
	}
}

func (h *Hub) originPatterns() []string {
	patterns := make([]string, 0, len(h.origins))
	for _, o := range h.origins {
		// OriginPatterns match host[:port], so strip the scheme.
		for i := 0; i+2 < len(o); i++ {
			if o[i:i+3] == "://" {
				o = o[i+3:]
				break
			}
		}
		patterns = append(patterns, o)
	}
	return patterns
}
