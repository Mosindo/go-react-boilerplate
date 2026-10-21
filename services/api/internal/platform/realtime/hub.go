// Package realtime pushes server events to connected clients over WebSocket.
package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Event is the envelope sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher is what features depend on to emit events.
type Publisher interface {
	Publish(userID string, event Event)
}

// NopPublisher discards events (tests, seeds).
type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}

const (
	authTimeout  = 5 * time.Second
	pongWait     = 60 * time.Second
	pingInterval = 25 * time.Second
	writeWait    = 10 * time.Second
	sendBuffer   = 32
)

type client struct {
	userID string
	send   chan []byte
}

// Hub keeps live connections per user. It is process-local: with several API replicas, back it
// with Postgres LISTEN/NOTIFY or Redis pub/sub.
type Hub struct {
	mu       sync.RWMutex
	clients  map[string]map[*client]struct{}
	upgrader websocket.Upgrader
	validate func(token string) (userID string, err error)
}

// NewHub creates a hub; validate turns an access token into a user id.
func NewHub(validate func(token string) (string, error)) *Hub {
	return &Hub{
		clients:  make(map[string]map[*client]struct{}),
		validate: validate,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Authentication happens with a first message, never with cookies, so cross-site
			// WebSocket hijacking is not applicable.
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default: // slow consumer: drop, the client resyncs through REST
		}
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
	if set := h.clients[c.userID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Handle upgrades the request. The client must send {"type":"auth","token":"<access token>"} first.
func (h *Hub) Handle(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(4096)

	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
	var hello struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := conn.ReadJSON(&hello); err != nil || hello.Type != "auth" {
		return
	}
	userID, err := h.validate(hello.Token)
	if err != nil {
		_ = conn.WriteJSON(Event{Type: "error", Data: "unauthorized"})
		return
	}

	cl := &client{userID: userID, send: make(chan []byte, sendBuffer)}
	h.add(cl)
	defer h.remove(cl)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	go h.writeLoop(ctx, conn, cl)

	cl.send <- mustJSON(Event{Type: "ready", Data: nil})
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })
	for {
		// Clients send nothing after auth except keep-alive frames; any read error ends the session.
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	}
}

func (h *Hub) writeLoop(ctx context.Context, conn *websocket.Conn, cl *client) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cl.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				_ = conn.Close()
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				_ = conn.Close()
				return
			}
		}
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
