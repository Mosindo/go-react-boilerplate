package realtime

import (
	"context"
	"testing"
	"time"
)

func TestHubDeliversEventPublishedWhileWaiting(t *testing.T) {
	hub := NewHub()
	cursor := hub.Cursor()

	go func() {
		time.Sleep(20 * time.Millisecond)
		hub.Publish("u1", Event{Type: TypeMessage, ConversationID: "c1"})
	}()

	events := hub.Wait(context.Background(), "u1", cursor, 2*time.Second)
	if len(events) != 1 || events[0].Type != TypeMessage || events[0].ConversationID != "c1" {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestHubReplaysMissedEventsAndIsolatesUsers(t *testing.T) {
	hub := NewHub()
	hub.Publish("u1", Event{Type: TypeMatch})
	hub.Publish("u2", Event{Type: TypeMessage})

	events := hub.Wait(context.Background(), "u1", 0, 50*time.Millisecond)
	if len(events) != 1 || events[0].Type != TypeMatch {
		t.Fatalf("u1 should only see its own event: %#v", events)
	}
	if again := hub.Wait(context.Background(), "u1", events[0].Seq, 30*time.Millisecond); len(again) != 0 {
		t.Fatalf("expected timeout without events, got %#v", again)
	}
}
