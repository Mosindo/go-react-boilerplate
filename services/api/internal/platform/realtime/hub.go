// Package realtime fans out server events (new message, match, read receipt,
// notification) to the WebSocket connections of a user. The hub is in-process;
// running several API replicas requires a shared bus (e.g. Postgres
// LISTEN/NOTIFY or Redis) behind the same Publisher interface.
package realtime

import (
	"encoding/json"
	"log"
	"sync"
)

// Event is the envelope sent to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Publisher delivers events to every connection of a user.
type Publisher interface {
	Publish(userID string, event Event)
}

// Client is one live connection.
type Client struct {
	UserID    string
	SessionID string
	Send      chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*Client]struct{})}
}

const maxConnectionsPerUser = 5

// Register adds a connection. The oldest connection is dropped when a user
// exceeds the per-user limit.
func (h *Hub) Register(userID, sessionID string) *Client {
	client := &Client{UserID: userID, SessionID: sessionID, Send: make(chan []byte, 32)}
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[userID]
	if !ok {
		set = make(map[*Client]struct{})
		h.clients[userID] = set
	}
	if len(set) >= maxConnectionsPerUser {
		for existing := range set {
			delete(set, existing)
			close(existing.Send)
			break
		}
	}
	set[client] = struct{}{}
	return client
}

// Unregister removes a connection and closes its send channel.
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.clients[client.UserID]
	if !ok {
		return
	}
	if _, ok := set[client]; ok {
		delete(set, client)
		close(client.Send)
	}
	if len(set) == 0 {
		delete(h.clients, client.UserID)
	}
}

// Publish sends event to the user without blocking; slow clients are dropped
// and will resynchronise over REST when they reconnect.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf(`{"event":"realtime_marshal_error","error":%q}`, err.Error())
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients[userID] {
		select {
		case client.Send <- payload:
		default:
			delete(h.clients[userID], client)
			close(client.Send)
		}
	}
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
}

// DisconnectSession closes the connections opened with one session (logout).
func (h *Hub) DisconnectSession(userID, sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients[userID] {
		if client.SessionID == sessionID {
			delete(h.clients[userID], client)
			close(client.Send)
		}
	}
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
}

// DisconnectUser closes every connection of a user (logout, account deletion).
func (h *Hub) DisconnectUser(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients[userID] {
		close(client.Send)
	}
	delete(h.clients, userID)
}
