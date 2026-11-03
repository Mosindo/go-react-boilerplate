// Package realtime fans server events out to connected WebSocket clients.
// The hub is in-process; run a single API instance or add a pub/sub bridge
// (Postgres LISTEN/NOTIFY, Redis) before scaling horizontally.
package realtime

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher is what features depend on to push events to users.
type Publisher interface {
	Publish(userID string, ev Event)
}

// NopPublisher is used when realtime is not wired (tests, seeding).
type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}

// TokenVerifier turns an access token into a user id.
type TokenVerifier func(ctx context.Context, token string) (string, bool)

type client struct {
	userID string
	send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub { return &Hub{clients: map[string]map[*client]struct{}{}} }

func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = map[*client]struct{}{}
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

// Publish never blocks: slow consumers drop events and resync via REST.
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

const (
	authTimeout = 5 * time.Second
	pongWait    = 60 * time.Second
	pingEvery   = 25 * time.Second
	maxFrame    = 4096
)

// Handler upgrades GET /ws. The client must send {"type":"auth","token":"..."}
// within 5 seconds; tokens never travel in the URL.
func (h *Hub) Handler(verify TokenVerifier, allowedOrigins []string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin == "" {
				return true // native clients send no Origin
			}
			_, ok := allowed[origin]
			return ok
		},
	}
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		go h.serve(conn, verify)
	}
}

func (h *Hub) serve(conn *websocket.Conn, verify TokenVerifier) {
	defer conn.Close()
	conn.SetReadLimit(maxFrame)
	_ = conn.SetReadDeadline(time.Now().Add(authTimeout))

	var hello struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := conn.ReadJSON(&hello); err != nil || hello.Type != "auth" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	userID, ok := verify(ctx, hello.Token)
	cancel()
	if !ok {
		_ = conn.WriteJSON(Event{Type: "error", Data: "unauthorized"})
		return
	}

	cl := &client{userID: userID, send: make(chan []byte, 32)}
	h.add(cl)
	defer h.remove(cl)
	_ = conn.WriteJSON(Event{Type: "ready", Data: nil})

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		}
	}()

	ticker := time.NewTicker(pingEvery)
	defer ticker.Stop()
	for {
		select {
		case msg := <-cl.send:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf(`{"event":"ws_write_error","user_id":%q}`, userID)
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}
