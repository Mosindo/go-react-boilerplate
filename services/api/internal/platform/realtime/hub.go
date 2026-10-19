package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

// Event is the envelope pushed to connected clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher delivers events to every live connection of a user.
type Publisher interface {
	Publish(userID string, ev Event)
}

type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}

const (
	sendBuffer      = 32
	authTimeout     = 10 * time.Second
	writeTimeout    = 10 * time.Second
	readLimitBytes  = 4096
	maxConnsPerUser = 5
)

type client struct {
	send chan []byte
}

// Hub is an in-process fan-out. Running several API instances requires
// replacing it with a shared pub/sub (Redis, Postgres LISTEN/NOTIFY); the
// Publisher interface is the seam for that.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{conns: make(map[string]map[*client]struct{})}
}

func (h *Hub) register(userID string, c *client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.conns[userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.conns[userID] = set
	}
	if len(set) >= maxConnsPerUser {
		return false
	}
	set[c] = struct{}{}
	return true
}

func (h *Hub) unregister(userID string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.conns[userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.conns, userID)
		}
	}
}

// Publish never blocks: a client whose buffer is full is considered stalled and
// simply misses the event (it re-syncs through the REST API on reconnect).
func (h *Hub) Publish(userID string, ev Event) {
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.conns[userID] {
		select {
		case c.send <- payload:
		default:
		}
	}
}

// ConnCount reports live connections for a user (used by tests).
func (h *Hub) ConnCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns[userID])
}

type authMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Handler upgrades to a WebSocket. Authentication happens with the first
// message so the access token never appears in a URL or access log.
func (h *Hub) Handler(verify func(token string) (string, error), originPatterns []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: originPatterns})
		if err != nil {
			return
		}
		defer conn.CloseNow()
		conn.SetReadLimit(readLimitBytes)

		ctx := c.Request.Context()
		authCtx, cancel := context.WithTimeout(ctx, authTimeout)
		_, raw, err := conn.Read(authCtx)
		cancel()
		if err != nil {
			conn.Close(websocket.StatusPolicyViolation, "auth required")
			return
		}
		var msg authMessage
		if json.Unmarshal(raw, &msg) != nil || msg.Type != "auth" {
			conn.Close(websocket.StatusPolicyViolation, "auth required")
			return
		}
		userID, err := verify(msg.Token)
		if err != nil {
			conn.Close(websocket.StatusPolicyViolation, "invalid token")
			return
		}

		cl := &client{send: make(chan []byte, sendBuffer)}
		if !h.register(userID, cl) {
			conn.Close(websocket.StatusTryAgainLater, "too many connections")
			return
		}
		defer h.unregister(userID, cl)

		ctx, cancelAll := context.WithCancel(ctx)
		defer cancelAll()

		if err := writeJSON(ctx, conn, Event{Type: "ready"}); err != nil {
			return
		}

		go func() {
			defer cancelAll()
			for {
				select {
				case <-ctx.Done():
					return
				case payload := <-cl.send:
					wctx, wcancel := context.WithTimeout(ctx, writeTimeout)
					err := conn.Write(wctx, websocket.MessageText, payload)
					wcancel()
					if err != nil {
						return
					}
				}
			}
		}()

		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				if !errors.Is(err, context.Canceled) && websocket.CloseStatus(err) == -1 {
					log.Printf(`{"event":"ws_read_end","error":%q}`, err.Error())
				}
				return
			}
			var in struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(data, &in) == nil && in.Type == "ping" {
				select {
				case cl.send <- []byte(`{"type":"pong"}`):
				default:
				}
			}
		}
	}
}

func writeJSON(ctx context.Context, conn *websocket.Conn, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, payload)
}
