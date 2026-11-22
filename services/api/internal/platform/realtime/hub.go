// Package realtime fans out server events to connected clients.
//
// The Hub is in-process: it is correct for a single API instance. To scale
// horizontally, implement Publisher on top of PostgreSQL LISTEN/NOTIFY or a
// message broker; feature services only depend on the Publisher interface.
package realtime

import (
	"encoding/json"
	"sync"
	"time"
)

// Event is delivered to subscribers as a Server-Sent Event.
type Event struct {
	Type string    `json:"type"`
	Data any       `json:"data"`
	At   time.Time `json:"at"`
}

// Publisher sends events to every live connection of the given user.
type Publisher interface {
	Publish(userID string, eventType string, data any)
}

// Subscription is a single client connection.
type Subscription struct {
	C      <-chan []byte
	ch     chan []byte
	userID string
}

// Hub keeps subscriptions per user.
type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[*Subscription]struct{}
	// maxPerUser bounds concurrent streams per account (multi-device support).
	maxPerUser int
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[*Subscription]struct{}), maxPerUser: 5}
}

// Subscribe registers a new stream for userID. When the per-user limit is
// reached, the oldest arbitrary stream is evicted.
func (h *Hub) Subscribe(userID string) *Subscription {
	ch := make(chan []byte, 32)
	sub := &Subscription{C: ch, ch: ch, userID: userID}

	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.subs[userID]
	if !ok {
		set = make(map[*Subscription]struct{})
		h.subs[userID] = set
	}
	if len(set) >= h.maxPerUser {
		for victim := range set {
			delete(set, victim)
			close(victim.ch)
			break
		}
	}
	set[sub] = struct{}{}
	return sub
}

// Unsubscribe removes a stream. It is safe to call more than once.
func (h *Hub) Unsubscribe(sub *Subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.subs[sub.userID]
	if !ok {
		return
	}
	if _, ok := set[sub]; !ok {
		return
	}
	delete(set, sub)
	close(sub.ch)
	if len(set) == 0 {
		delete(h.subs, sub.userID)
	}
}

// Publish never blocks: slow consumers drop events and resynchronise through
// the REST API on reconnect.
func (h *Hub) Publish(userID, eventType string, data any) {
	payload, err := json.Marshal(Event{Type: eventType, Data: data, At: time.Now().UTC()})
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for sub := range h.subs[userID] {
		select {
		case sub.ch <- payload:
		default:
		}
	}
}

// Connections returns the number of live streams for a user (used in tests).
func (h *Hub) Connections(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs[userID])
}

// Noop discards events; useful for tests and tools.
type Noop struct{}

func (Noop) Publish(string, string, any) {}
