// Package realtime fans out server events (new message, new match, read
// receipts...) to the WebSocket connections of a user.
//
// The hub is in-process. Running several API replicas requires a shared bus
// (e.g. Postgres LISTEN/NOTIFY or Redis pub/sub) behind Publish; clients also
// refetch over REST when they reconnect, so no event is lost permanently.
package realtime

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Event is the envelope pushed to clients: {"type": "...", "data": {...}}.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

const (
	EventMessageCreated   = "message.created"
	EventConversationRead = "conversation.read"
	EventMatchCreated     = "match.created"
	EventMatchRemoved     = "match.removed"
	EventNotification     = "notification.created"
)

// Publisher is what features depend on; tests can pass a no-op.
type Publisher interface {
	Publish(userID string, event Event)
}

type Nop struct{}

func (Nop) Publish(string, Event) {}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 25 * time.Second
	maxMessageSize = 512
	sendBuffer     = 32
)

type client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
	once   sync.Once
}

func (c *client) close() {
	c.once.Do(func() { close(c.send) })
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
			// Slow consumer: drop the connection, the client will reconnect and resync.
			go h.remove(c)
		}
	}
}

// DisconnectUser closes every connection of a user (logout everywhere, deletion).
func (h *Hub) DisconnectUser(userID string) {
	h.mu.RLock()
	list := make([]*client, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		list = append(list, c)
	}
	h.mu.RUnlock()
	for _, c := range list {
		h.remove(c)
	}
}

// IsOnline reports whether the user has at least one open connection.
func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[c.userID]
	if !ok {
		set = make(map[*client]struct{})
		h.clients[c.userID] = set
	}
	set[c] = struct{}{}
}

func (h *Hub) remove(c *client) {
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

// Serve takes ownership of an upgraded connection and blocks until it closes.
func (h *Hub) Serve(conn *websocket.Conn, userID string) {
	c := &client{userID: userID, conn: conn, send: make(chan []byte, sendBuffer)}
	h.add(c)

	go h.writePump(c)
	h.readPump(c)
}

func (h *Hub) readPump(c *client) {
	defer func() {
		h.remove(c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		// Clients only send pings/keep-alives; the payload is ignored.
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) writePump(c *client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
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
