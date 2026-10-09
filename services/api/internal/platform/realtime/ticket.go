package realtime

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

const ticketTTL = 30 * time.Second

// Tickets issues short-lived, single-use tokens that authenticate a WebSocket
// handshake. Browsers cannot set an Authorization header on WebSocket
// requests, and putting the long-lived JWT in a URL would leak it into logs.
type Tickets struct {
	mu    sync.Mutex
	items map[string]ticket
	now   func() time.Time
}

type ticket struct {
	userID  string
	expires time.Time
}

func NewTickets() *Tickets {
	return &Tickets{items: make(map[string]ticket), now: time.Now}
}

func (t *Tickets) Issue(userID string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	value := base64.RawURLEncoding.EncodeToString(buf)

	now := t.now()
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, v := range t.items {
		if !now.Before(v.expires) {
			delete(t.items, k)
		}
	}
	t.items[value] = ticket{userID: userID, expires: now.Add(ticketTTL)}
	return value, nil
}

// Redeem consumes a ticket: it works exactly once.
func (t *Tickets) Redeem(value string) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	item, ok := t.items[value]
	if !ok {
		return "", false
	}
	delete(t.items, value)
	if !t.now().Before(item.expires) {
		return "", false
	}
	return item.userID, true
}
