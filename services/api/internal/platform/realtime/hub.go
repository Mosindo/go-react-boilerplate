// Package realtime fans out per-user events (new message, read receipt, new
// match, notification) to connected WebSocket clients.
//
// The hub is in-process. Running several API replicas requires a shared
// broker (e.g. Postgres LISTEN/NOTIFY or Redis pub/sub) feeding Publish.
package realtime

import (
	"sync"
)

const subscriberBuffer = 64

// Event is the JSON envelope pushed to clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Publisher is the narrow interface features depend on.
type Publisher interface {
	Publish(userID string, event Event)
}

type subscriber struct {
	ch chan Event
}

type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[*subscriber]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[*subscriber]struct{})}
}

// Subscribe registers a listener for userID. Call the returned func to detach.
func (h *Hub) Subscribe(userID string) (<-chan Event, func()) {
	s := &subscriber{ch: make(chan Event, subscriberBuffer)}
	h.mu.Lock()
	if h.subs[userID] == nil {
		h.subs[userID] = make(map[*subscriber]struct{})
	}
	h.subs[userID][s] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	return s.ch, func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subs[userID], s)
			if len(h.subs[userID]) == 0 {
				delete(h.subs, userID)
			}
			h.mu.Unlock()
			close(s.ch)
		})
	}
}

// Publish delivers event to every connection of userID without blocking:
// slow consumers drop events and recover through their REST refetch.
func (h *Hub) Publish(userID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for s := range h.subs[userID] {
		select {
		case s.ch <- event:
		default:
		}
	}
}

// Connections returns the number of live connections for userID.
func (h *Hub) Connections(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs[userID])
}

// Nop discards events; useful in tests and tools.
type Nop struct{}

func (Nop) Publish(string, Event) {}
