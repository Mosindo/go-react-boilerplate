// Package realtime fournit un hub WebSocket en memoire pour pousser des evenements
// aux utilisateurs connectes (nouveau message, match, notification).
package realtime

import (
	"encoding/json"
	"sync"
)

// Event est l'enveloppe JSON envoyee aux clients.
type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// Publisher est l'interface consommee par les features pour pousser un evenement.
type Publisher interface {
	Publish(userID string, event Event)
}

type client struct {
	userID string
	send   chan []byte
}

// Hub maintient les connexions par utilisateur.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*client]struct{})}
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[c.userID]
	if set == nil {
		set = make(map[*client]struct{})
		h.clients[c.userID] = set
	}
	set[c] = struct{}{}
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.userID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// Publish envoie l'evenement a toutes les connexions de l'utilisateur (non bloquant).
func (h *Hub) Publish(userID string, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- data:
		default: // client trop lent: on abandonne l'evenement (il resyncra via REST)
		}
	}
}

// Online indique si l'utilisateur a au moins une connexion active.
func (h *Hub) Online(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

// Noop est un Publisher inactif (tests, usages sans temps reel).
type Noop struct{}

func (Noop) Publish(string, Event) {}
