// Package realtime fans server events out to connected members over
// WebSocket. Clients never send data through the socket: every write goes
// through the REST API (validated and authorised there) and the socket only
// pushes the resulting events.
//
// The hub is in-process. Running several API replicas requires a shared bus
// (for example Postgres LISTEN/NOTIFY or Redis pub/sub) behind Publisher.
package realtime

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 25 * time.Second
	maxInboundSize = 512
	sendBuffer     = 32
)

// Event is the envelope pushed to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher is what features depend on.
type Publisher interface {
	Publish(userID string, event Event)
	Disconnect(userID string)
}

type client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
	once   sync.Once
}

func (c *client) close() {
	c.once.Do(func() {
		close(c.send)
	})
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

// Publish delivers an event to every connection of userID. Slow consumers are
// dropped instead of blocking the publisher.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf(`{"event":"realtime_marshal_error","type":%q}`, event.Type)
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
		case c.send <- payload:
		default:
			h.unregister(c)
		}
	}
}

// Disconnect closes every socket of a member (logout, account deletion).
func (h *Hub) Disconnect(userID string) {
	h.mu.RLock()
	targets := make([]*client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		h.unregister(c)
	}
}

// ConnectionCount is used by tests and diagnostics.
func (h *Hub) ConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	if set, ok := h.clients[c.userID]; ok {
		if _, present := set[c]; present {
			delete(set, c)
			if len(set) == 0 {
				delete(h.clients, c.userID)
			}
		}
	}
	h.mu.Unlock()
	c.close()
}

// Serve takes ownership of an upgraded connection until it closes.
func (h *Hub) Serve(userID string, conn *websocket.Conn) {
	c := &client{userID: userID, conn: conn, send: make(chan []byte, sendBuffer)}
	h.register(c)
	go h.writePump(c)
	h.readPump(c)
}

func (h *Hub) readPump(c *client) {
	defer func() {
		h.unregister(c)
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(maxInboundSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		// Inbound frames are ignored; reading keeps pong/close handling alive.
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) writePump(c *client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	hello, _ := json.Marshal(Event{Type: "connection.ready", Data: map[string]string{"userId": c.userID}})
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteMessage(websocket.TextMessage, hello); err != nil {
		return
	}
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
