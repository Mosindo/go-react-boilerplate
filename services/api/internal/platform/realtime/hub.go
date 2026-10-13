// Package realtime fans server events out to connected WebSocket clients.
// The hub is in-process: with several API instances, back it with Postgres
// LISTEN/NOTIFY or Redis pub/sub behind the same Publisher interface.
package realtime

import "sync"

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// Publisher delivers an event to every live connection of a user.
type Publisher interface {
	Publish(userID string, event Event)
	Disconnect(userID string)
}

const clientBuffer = 32

type Client struct {
	UserID string
	Send   chan Event
	done   chan struct{}
	once   sync.Once
}

func (c *Client) Done() <-chan struct{} { return c.done }

func (c *Client) close() { c.once.Do(func() { close(c.done) }) }

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*Client]struct{})}
}

func (h *Hub) Register(userID string) *Client {
	c := &Client{UserID: userID, Send: make(chan Event, clientBuffer), done: make(chan struct{})}
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]struct{})
	}
	h.clients[userID][c] = struct{}{}
	h.mu.Unlock()
	return c
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if set := h.clients[c.UserID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.UserID)
		}
	}
	h.mu.Unlock()
	c.close()
}

// Publish never blocks: a client whose buffer is full is dropped and must
// reconnect (it re-syncs through the REST API).
func (h *Hub) Publish(userID string, event Event) {
	h.mu.RLock()
	var slow []*Client
	for c := range h.clients[userID] {
		select {
		case c.Send <- event:
		default:
			slow = append(slow, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range slow {
		h.Unregister(c)
	}
}

func (h *Hub) Disconnect(userID string) {
	h.mu.Lock()
	set := h.clients[userID]
	delete(h.clients, userID)
	h.mu.Unlock()
	for c := range set {
		c.close()
	}
}

func (h *Hub) Online(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

// Discard is a Publisher that drops everything (tests, jobs).
type Discard struct{}

func (Discard) Publish(string, Event) {}
func (Discard) Disconnect(string)     {}
