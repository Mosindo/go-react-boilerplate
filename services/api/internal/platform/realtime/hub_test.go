package realtime

import (
	"testing"
	"time"
)

func TestTicketsAreSingleUseAndExpire(t *testing.T) {
	h := NewHub(nil)
	now := time.Unix(1000, 0)
	h.now = func() time.Time { return now }

	id, err := h.IssueTicket("user-1")
	if err != nil || len(id) < 32 {
		t.Fatalf("ticket: %q %v", id, err)
	}
	if uid, ok := h.redeem(id); !ok || uid != "user-1" {
		t.Fatalf("first redeem: %q %v", uid, ok)
	}
	if _, ok := h.redeem(id); ok {
		t.Fatal("ticket must be single use")
	}

	expired, _ := h.IssueTicket("user-2")
	now = now.Add(ticketTTL + time.Second)
	if _, ok := h.redeem(expired); ok {
		t.Fatal("expired ticket must be refused")
	}
	if _, ok := h.redeem("unknown"); ok {
		t.Fatal("unknown ticket must be refused")
	}
}

func TestPublishDropsSlowConsumers(t *testing.T) {
	h := NewHub(nil)
	slow := &client{send: make(chan []byte, 1)}
	if !h.add("u", slow) {
		t.Fatal("add failed")
	}
	h.Publish("u", Event{Type: "a"})
	h.Publish("u", Event{Type: "b"}) // buffer full: consumer is dropped
	if len(h.clients["u"]) != 0 {
		t.Fatal("slow consumer should have been removed")
	}
	h.Publish("nobody", Event{Type: "c"}) // no panic without listeners
}

func TestConnectionCapPerUser(t *testing.T) {
	h := NewHub(nil)
	for i := 0; i < maxConnsPerUsr; i++ {
		if !h.add("u", &client{send: make(chan []byte, 1)}) {
			t.Fatalf("connection %d should be accepted", i)
		}
	}
	if h.add("u", &client{send: make(chan []byte, 1)}) {
		t.Fatal("connection over the cap must be refused")
	}
}
