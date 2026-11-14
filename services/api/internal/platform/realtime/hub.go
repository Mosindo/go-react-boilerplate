// Package realtime pushes server events to connected clients over WebSocket.
//
// Protocol: the client opens GET /ws, then sends {"type":"auth","token":"<access token>"}
// within authTimeout. The server answers {"type":"ready"} and then streams
// events {"type":"message.created","data":{...}}. Tokens never travel in the
// URL, so they do not end up in proxy logs.
//
// The hub is in-memory (single API instance). To scale horizontally, fan
// events out through Postgres LISTEN/NOTIFY or Redis behind Publish.
package realtime

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

const (
	authTimeout  = 10 * time.Second
	writeTimeout = 5 * time.Second
	pingInterval = 25 * time.Second
	sendBuffer   = 32
	maxConnsUser = 5
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher is what features depend on; it keeps them decoupled from the
// transport and easy to fake in tests.
type Publisher interface {
	Publish(userID string, event Event)
}

// Authenticator turns an access token into a user id.
type Authenticator func(ctx context.Context, token string) (string, error)

type client struct {
	userID string
	send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf(`{"event":"realtime_marshal_error","error":%q}`, err.Error())
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
			// Slow consumer: drop the event; the client resyncs over REST.
		}
	}
}

// Disconnect closes every socket of a user (logout everywhere, deletion).
func (h *Hub) Disconnect(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients[userID] {
		close(c.send)
	}
	delete(h.clients, userID)
}

func (h *Hub) register(c *client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[c.userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.clients[c.userID] = set
	}
	if len(set) >= maxConnsUser {
		return false
	}
	set[c] = struct{}{}
	return true
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[c.userID]
	if _, ok := set[c]; !ok {
		return
	}
	delete(set, c)
	close(c.send)
	if len(set) == 0 {
		delete(h.clients, c.userID)
	}
}

type authMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades the request. originPatterns restricts browser origins
// (native apps send no Origin header and are accepted).
func (h *Hub) Handler(authenticate Authenticator, originPatterns []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: originPatterns})
		if err != nil {
			return
		}
		conn.SetReadLimit(4096)
		ctx := c.Request.Context()

		authCtx, cancel := context.WithTimeout(ctx, authTimeout)
		_, raw, err := conn.Read(authCtx)
		cancel()
		if err != nil {
			conn.Close(websocket.StatusPolicyViolation, "authentication required")
			return
		}
		var msg authMessage
		if json.Unmarshal(raw, &msg) != nil || msg.Type != "auth" || msg.Token == "" {
			conn.Close(websocket.StatusPolicyViolation, "authentication required")
			return
		}
		userID, err := authenticate(ctx, msg.Token)
		if err != nil {
			conn.Close(websocket.StatusPolicyViolation, "invalid token")
			return
		}

		cl := &client{userID: userID, send: make(chan []byte, sendBuffer)}
		if !h.register(cl) {
			conn.Close(websocket.StatusTryAgainLater, "too many connections")
			return
		}
		defer h.unregister(cl)

		ready, _ := json.Marshal(Event{Type: "ready"})
		if err := write(ctx, conn, ready); err != nil {
			return
		}

		// Reader: we do not expect client messages after auth, but reading
		// is required to process control frames and detect disconnects.
		readerDone := make(chan struct{})
		go func() {
			defer close(readerDone)
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
			case <-readerDone:
				return
			case payload, ok := <-cl.send:
				if !ok {
					conn.Close(websocket.StatusNormalClosure, "closed")
					return
				}
				if err := write(ctx, conn, payload); err != nil {
					return
				}
			case <-ticker.C:
				pingCtx, cancel := context.WithTimeout(ctx, writeTimeout)
				err := conn.Ping(pingCtx)
				cancel()
				if err != nil {
					return
				}
			}
		}
	}
}

func write(ctx context.Context, conn *websocket.Conn, payload []byte) error {
	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return conn.Write(writeCtx, websocket.MessageText, payload)
}
