package realtime

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// TicketTTL bounds how long a ticket can wait before being exchanged.
const TicketTTL = 30 * time.Second

// Ticket identifies the member a WebSocket connection belongs to. Browsers
// cannot set an Authorization header on WebSocket upgrades, so clients trade
// their bearer token for a single-use, short-lived ticket passed in the URL.
type Ticket struct {
	UserID    string
	SessionID string
	expiresAt time.Time
}

type TicketStore struct {
	mu      sync.Mutex
	tickets map[string]Ticket
	now     func() time.Time
}

func NewTicketStore() *TicketStore {
	return &TicketStore{tickets: make(map[string]Ticket), now: time.Now}
}

func (s *TicketStore) Issue(userID, sessionID string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for k, t := range s.tickets {
		if now.After(t.expiresAt) {
			delete(s.tickets, k)
		}
	}
	s.tickets[token] = Ticket{UserID: userID, SessionID: sessionID, expiresAt: now.Add(TicketTTL)}
	return token, nil
}

// Redeem consumes a ticket; it can only ever succeed once.
func (s *TicketStore) Redeem(token string) (Ticket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tickets[token]
	if !ok {
		return Ticket{}, false
	}
	delete(s.tickets, token)
	if s.now().After(t.expiresAt) {
		return Ticket{}, false
	}
	return t, true
}
