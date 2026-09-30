// Package realtime pushes live events (new messages, read receipts, matches,
// notifications) to connected clients over WebSocket.
//
// Fan-out goes through PostgreSQL LISTEN/NOTIFY: every API instance listens on
// the same channel and delivers events to its own local connections, so the
// feature works with several replicas without extra infrastructure.
package realtime

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"example.com/api/internal/platform/authtoken"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	channelName        = "realtime_events"
	maxNotifyPayload   = 7500 // PostgreSQL limit is 8000 bytes
	writeWait          = 10 * time.Second
	pongWait           = 60 * time.Second
	pingPeriod         = 25 * time.Second
	sendBuffer         = 32
	maxConnsPerUser    = 5
	listenRetryBackoff = 2 * time.Second
)

// Event is the JSON frame sent to clients.
type Event struct {
	Type           string `json:"type"`
	ConversationID string `json:"conversationId,omitempty"`
	Data           any    `json:"data,omitempty"`
	// Truncated is set when Data was dropped because it exceeded the
	// transport limit; clients then refetch the resource.
	Truncated bool `json:"truncated,omitempty"`
}

// Publisher is what feature services depend on.
type Publisher interface {
	Publish(ctx context.Context, userIDs []string, event Event)
}

// NopPublisher discards events (used in tests).
type NopPublisher struct{}

func (NopPublisher) Publish(context.Context, []string, Event) {}

type envelope struct {
	UserIDs []string        `json:"u"`
	Event   json.RawMessage `json:"e"`
}

type client struct {
	userID string
	send   chan []byte
}

type Hub struct {
	pool    *pgxpool.Pool
	tokens  *authtoken.Manager
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
	upgrade websocket.Upgrader
}

func NewHub(pool *pgxpool.Pool, tokens *authtoken.Manager) *Hub {
	return &Hub{
		pool:    pool,
		tokens:  tokens,
		clients: make(map[string]map[*client]struct{}),
		upgrade: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			// Authentication relies on a short-lived ticket obtained with a
			// bearer token (never a cookie), so cross-site WebSocket hijacking
			// is not possible and any origin (including native apps) is fine.
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

// Publish broadcasts an event to the given users on every instance.
func (h *Hub) Publish(ctx context.Context, userIDs []string, event Event) {
	if len(userIDs) == 0 {
		return
	}
	payload, err := h.encode(userIDs, event)
	if err != nil {
		log.Printf(`{"event":"realtime_encode_failed","type":%q,"error":%q}`, event.Type, err.Error())
		return
	}
	if _, err := h.pool.Exec(ctx, `SELECT pg_notify($1, $2)`, channelName, payload); err != nil {
		log.Printf(`{"event":"realtime_publish_failed","type":%q,"error":%q}`, event.Type, err.Error())
	}
}

func (h *Hub) encode(userIDs []string, event Event) (string, error) {
	raw, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(envelope{UserIDs: userIDs, Event: raw})
	if err != nil {
		return "", err
	}
	if len(payload) <= maxNotifyPayload {
		return string(payload), nil
	}
	event.Data = nil
	event.Truncated = true
	raw, err = json.Marshal(event)
	if err != nil {
		return "", err
	}
	payload, err = json.Marshal(envelope{UserIDs: userIDs, Event: raw})
	return string(payload), err
}

// Run listens for notifications until ctx is cancelled, reconnecting on error.
func (h *Hub) Run(ctx context.Context) {
	for ctx.Err() == nil {
		if err := h.listen(ctx); err != nil && ctx.Err() == nil {
			log.Printf(`{"event":"realtime_listen_error","error":%q}`, err.Error())
			select {
			case <-ctx.Done():
			case <-time.After(listenRetryBackoff):
			}
		}
	}
}

func (h *Hub) listen(ctx context.Context) error {
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "LISTEN "+channelName); err != nil {
		return err
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), "UNLISTEN "+channelName)
	}()

	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		var env envelope
		if err := json.Unmarshal([]byte(notification.Payload), &env); err != nil {
			continue
		}
		h.deliver(env.UserIDs, env.Event)
	}
}

func (h *Hub) deliver(userIDs []string, frame []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, userID := range userIDs {
		for c := range h.clients[userID] {
			select {
			case c.send <- frame:
			default:
				// Slow consumer: drop the frame; the client resyncs on reconnect.
			}
		}
	}
}

func (h *Hub) register(c *client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[c.userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.clients[c.userID] = set
	}
	if len(set) >= maxConnsPerUser {
		return false
	}
	set[c] = struct{}{}
	return true
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.clients[c.userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// TicketHandler issues a one-minute ticket for opening a WebSocket. Tickets
// avoid putting long-lived access tokens in URLs.
func (h *Hub) TicketHandler(c *gin.Context) {
	ticket, err := h.tokens.SignTicket(c.GetString("userID"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "internal_error"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": ticket, "expiresInSeconds": int(authtoken.TicketTTL.Seconds())})
}

// ConnectHandler upgrades the request after validating the ticket.
func (h *Hub) ConnectHandler(c *gin.Context) {
	claims, err := h.tokens.Parse(c.Query("ticket"), authtoken.TypeTicket)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid ticket", "code": "unauthorized"})
		return
	}

	cl := &client{userID: claims.UserID, send: make(chan []byte, sendBuffer)}
	if !h.register(cl) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many connections", "code": "rate_limited"})
		return
	}

	ws, err := h.upgrade.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.unregister(cl)
		return
	}
	go h.writePump(ws, cl)
	h.readPump(ws, cl)
}

func (h *Hub) readPump(ws *websocket.Conn, cl *client) {
	defer func() {
		h.unregister(cl)
		close(cl.send)
		_ = ws.Close()
	}()
	ws.SetReadLimit(512)
	_ = ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error {
		return ws.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		// Clients do not send commands; reads only process control frames.
		if _, _, err := ws.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) writePump(ws *websocket.Conn, cl *client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = ws.Close()
	}()

	hello, _ := json.Marshal(Event{Type: "connected"})
	_ = ws.SetWriteDeadline(time.Now().Add(writeWait))
	if err := ws.WriteMessage(websocket.TextMessage, hello); err != nil {
		return
	}

	for {
		select {
		case frame, ok := <-cl.send:
			_ = ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := ws.WriteMessage(websocket.TextMessage, frame); err != nil {
				return
			}
		case <-ticker.C:
			_ = ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func RegisterRoutes(r gin.IRouter, hub *Hub, requireUser gin.HandlerFunc) {
	r.POST("/realtime/ticket", requireUser, hub.TicketHandler)
	r.GET("/realtime", hub.ConnectHandler)
}
