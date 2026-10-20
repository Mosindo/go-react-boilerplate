// Package realtime implements an in-process WebSocket fan-out hub keyed by user.
// It is the single place where server events reach connected clients.
package realtime

import (
	"encoding/json"
	"sync"
)

// Event is the envelope sent to clients: {"type": "...", ...payload}.
type Event map[string]any

// Publisher delivers events to every live connection of a user.
type Publisher interface {
	Publish(userID string, event Event)
}

type client struct {
	send chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

func (h *Hub) register(userID string) *client {
	cl := &client{send: make(chan []byte, 32)}
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*client]struct{})
	}
	h.clients[userID][cl] = struct{}{}
	h.mu.Unlock()
	return cl
}

func (h *Hub) unregister(userID string, cl *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[userID]; ok {
		if _, present := set[cl]; present {
			delete(set, cl)
			close(cl.send)
		}
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
}

// Publish never blocks: a client whose buffer is full is dropped and must reconnect.
func (h *Hub) Publish(userID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	var slow []*client
	for cl := range h.clients[userID] {
		select {
		case cl.send <- payload:
		default:
			slow = append(slow, cl)
		}
	}
	h.mu.RUnlock()
	for _, cl := range slow {
		h.unregister(userID, cl)
	}
}

// DisconnectUser closes every connection of a user (account deletion, block).
func (h *Hub) DisconnectUser(userID string) {
	h.mu.RLock()
	var all []*client
	for cl := range h.clients[userID] {
		all = append(all, cl)
	}
	h.mu.RUnlock()
	for _, cl := range all {
		h.unregister(userID, cl)
	}
}

// NopPublisher is used in tests that do not care about events.
type NopPublisher struct{}

func (NopPublisher) Publish(string, Event) {}
