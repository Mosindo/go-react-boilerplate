package realtime

import (
	"testing"
	"time"
)

func TestTicketsAreSingleUseAndExpire(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	s := NewTicketStore()
	s.now = func() time.Time { return now }

	token, err := s.Issue("u1", "s1")
	if err != nil {
		t.Fatal(err)
	}
	ticket, ok := s.Redeem(token)
	if !ok || ticket.UserID != "u1" || ticket.SessionID != "s1" {
		t.Fatalf("unexpected redeem result %+v %v", ticket, ok)
	}
	if _, ok := s.Redeem(token); ok {
		t.Fatal("ticket must be single use")
	}

	expired, _ := s.Issue("u1", "s1")
	now = now.Add(TicketTTL + time.Second)
	if _, ok := s.Redeem(expired); ok {
		t.Fatal("expired ticket must be rejected")
	}
}
