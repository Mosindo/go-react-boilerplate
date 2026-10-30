// Package realtime delivers lightweight "something changed" signals to
// connected clients through long polling. Events never carry private content:
// clients refetch the affected resource through the normal authorized API.
package realtime

import (
	"context"
	"sync"
	"time"
)

type Event struct {
	Seq            int64  `json:"seq"`
	Type           string `json:"type"`
	ConversationID string `json:"conversationId,omitempty"`
}

const (
	TypeMessage      = "message"
	TypeRead         = "read"
	TypeMatch        = "match"
	TypeNotification = "notification"
	TypeUnmatch      = "unmatch"

	maxBuffered = 100
	bufferTTL   = 10 * time.Minute
)

type mailbox struct {
	events  []Event
	lastSeq int64
	touched time.Time
	waiters []chan struct{}
}

// Hub is an in-process fan-out. With several API instances, replace it with a
// shared transport (Postgres LISTEN/NOTIFY or Redis) behind the same methods.
type Hub struct {
	mu        sync.Mutex
	mailboxes map[string]*mailbox
	nextSeq   int64
	now       func() time.Time
}

func NewHub() *Hub {
	return &Hub{mailboxes: make(map[string]*mailbox), now: time.Now}
}

func (h *Hub) Publish(userID string, event Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.pruneLocked()
	mb := h.mailboxes[userID]
	if mb == nil {
		mb = &mailbox{}
		h.mailboxes[userID] = mb
	}
	h.nextSeq++
	event.Seq = h.nextSeq
	mb.events = append(mb.events, event)
	if len(mb.events) > maxBuffered {
		mb.events = mb.events[len(mb.events)-maxBuffered:]
	}
	mb.lastSeq = event.Seq
	mb.touched = h.now()
	for _, w := range mb.waiters {
		close(w)
	}
	mb.waiters = nil
}

// Cursor returns the sequence number a new subscriber should start after.
func (h *Hub) Cursor() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.nextSeq
}

// Wait returns the events for userID newer than after. It blocks until at
// least one exists, ctx ends, or timeout passes (then it returns none).
func (h *Hub) Wait(ctx context.Context, userID string, after int64, timeout time.Duration) []Event {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		h.mu.Lock()
		mb := h.mailboxes[userID]
		if mb != nil {
			if pending := eventsAfter(mb.events, after); len(pending) > 0 {
				h.mu.Unlock()
				return pending
			}
		}
		if mb == nil {
			mb = &mailbox{touched: h.now()}
			h.mailboxes[userID] = mb
		}
		ch := make(chan struct{})
		mb.waiters = append(mb.waiters, ch)
		h.mu.Unlock()

		select {
		case <-ch:
		case <-ctx.Done():
			return nil
		case <-deadline.C:
			return nil
		}
	}
}

func eventsAfter(events []Event, after int64) []Event {
	for i, e := range events {
		if e.Seq > after {
			out := make([]Event, len(events)-i)
			copy(out, events[i:])
			return out
		}
	}
	return nil
}

func (h *Hub) pruneLocked() {
	cutoff := h.now().Add(-bufferTTL)
	for id, mb := range h.mailboxes {
		if len(mb.waiters) == 0 && mb.touched.Before(cutoff) {
			delete(h.mailboxes, id)
		}
	}
}
