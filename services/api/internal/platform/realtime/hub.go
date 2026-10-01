// Package realtime fans server events out to connected WebSocket clients.
// It is in-memory: with several API replicas, put a pub/sub (e.g. Redis) behind Publish.
package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Event is the envelope pushed to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub { return &Hub{clients: make(map[string]map[*client]struct{})} }

// Publish sends ev to every live connection of userID. It never blocks: slow clients drop events
// and recover by refetching (clients refetch on reconnect).
func (h *Hub) Publish(userID string, ev Event) {
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
		}
	}
}

// Disconnect closes every connection of userID (account deletion, block of session).
func (h *Hub) Disconnect(userID string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		_ = c.conn.Close()
	}
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients[c.userID], c)
	if len(h.clients[c.userID]) == 0 {
		delete(h.clients, c.userID)
	}
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 45 * time.Second
	maxMsgSize = 512
)

// Serve runs the read/write pumps for an upgraded connection until it closes.
func (h *Hub) Serve(userID string, conn *websocket.Conn) {
	c := &client{userID: userID, conn: conn, send: make(chan []byte, 32)}
	h.add(c)
	defer func() {
		h.remove(c)
		_ = conn.Close()
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn.SetReadLimit(maxMsgSize)
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })
		for {
			// Clients only listen; inbound frames are ignored (writes go through the REST API).
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case msg := <-c.send:
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
