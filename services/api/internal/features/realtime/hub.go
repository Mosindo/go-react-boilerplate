package realtime

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Hub is the in-process connection registry. It implements events.Publisher.
// It is single-instance only: scaling out needs a pub/sub backplane (see
// docs/API.md).
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
	closed  bool
}

func NewHub() *Hub { return &Hub{clients: make(map[string]map[*client]struct{})} }

type client struct {
	hub    *Hub
	conn   *websocket.Conn
	userID string
	sid    string
	send   chan []byte
	once   sync.Once
	done   chan struct{}
}

func newClient(h *Hub, conn *websocket.Conn, userID, sid string) *client {
	return &client{hub: h, conn: conn, userID: userID, sid: sid, send: make(chan []byte, 32), done: make(chan struct{})}
}

// Publish delivers the event to every open connection of userID. It never
// blocks: a client whose buffer is full is considered dead and dropped.
func (h *Hub) Publish(userID, eventType string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		log.Printf(`{"event":"ws_marshal_failed","type":%q,"error":%q}`, eventType, err.Error())
		return
	}
	frame, err := json.Marshal(Envelope{Type: eventType, Data: raw})
	if err != nil {
		return
	}
	h.mu.RLock()
	targets := make([]*client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		select {
		case c.send <- frame:
		default:
			c.shutdown(websocket.ClosePolicyViolation, "slow consumer")
		}
	}
}

func (h *Hub) register(c *client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return false
	}
	set := h.clients[c.userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.clients[c.userID] = set
	}
	if len(set) >= MaxConnsPerUser {
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

// Connections reports the number of open connections of a user (tests, metrics).
func (h *Hub) Connections(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}

// Close terminates every connection (graceful shutdown).
func (h *Hub) Close() {
	h.mu.Lock()
	h.closed = true
	var all []*client
	for _, set := range h.clients {
		for c := range set {
			all = append(all, c)
		}
	}
	h.mu.Unlock()
	for _, c := range all {
		c.shutdown(websocket.CloseGoingAway, "server shutting down")
	}
}

// shutdown sends a close frame and tears the connection down exactly once.
func (c *client) shutdown(code int, reason string) {
	c.once.Do(func() {
		close(c.done)
		_ = c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
		_ = c.conn.Close()
	})
}
