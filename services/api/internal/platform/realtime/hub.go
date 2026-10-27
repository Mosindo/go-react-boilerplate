// Package realtime fans server events out to connected WebSocket clients.
// It is in-memory and per-process; scaling horizontally requires replacing it
// with a pub/sub backed implementation of the same Publisher interface.
package realtime

import (
	"context"
	"encoding/json"
	"sync"
)

// Event is the envelope sent to clients.
type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Publisher is what features depend on to push events.
type Publisher interface {
	Publish(userID string, eventType string, payload any)
}

const sendBuffer = 32

type Client struct {
	UserID string
	Send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*Client]struct{})}
}

func (h *Hub) Register(userID string) *Client {
	client := &Client{UserID: userID, Send: make(chan []byte, sendBuffer)}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]struct{})
	}
	h.clients[userID][client] = struct{}{}
	return client
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[client.UserID]; ok {
		delete(set, client)
		if len(set) == 0 {
			delete(h.clients, client.UserID)
		}
	}
}

// Publish never blocks: a slow client simply misses the event and recovers by
// refetching over REST (the REST API remains the source of truth).
func (h *Hub) Publish(userID string, eventType string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	data, err := json.Marshal(Event{Type: eventType, Payload: raw})
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients[userID] {
		select {
		case client.Send <- data:
		default:
		}
	}
}

// Disconnect closes every connection of a user (account deletion, logout-all).
func (h *Hub) Disconnect(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients[userID] {
		close(client.Send)
	}
	delete(h.clients, userID)
}

// PrepareUserDeletion lets the Hub act as an auth deletion hook: once the account
// is gone, its live connections are closed.
func (h *Hub) PrepareUserDeletion(_ context.Context, userID string) (func(), error) {
	return func() { h.Disconnect(userID) }, nil
}
