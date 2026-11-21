package chat

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNormalizeContent(t *testing.T) {
	got, err := NormalizeContent("  hello\r\nworld\x00 ")
	if err != nil || got != "hello\nworld" {
		t.Fatalf("unexpected normalization %q err=%v", got, err)
	}
	if _, err := NormalizeContent(" \n\t "); !errors.Is(err, ErrMessageContentNeeded) {
		t.Fatalf("expected empty message error, got %v", err)
	}
	if _, err := NormalizeContent(strings.Repeat("é", MaxMessageRunes+1)); !errors.Is(err, ErrMessageTooLong) {
		t.Fatalf("expected too long error, got %v", err)
	}
}

func TestToResponseReadState(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Minute)
	msg := ChatMessage{ID: "m", CreatedAt: now}
	if toResponse(msg, Conversation{}).IsRead {
		t.Fatal("unread when the other member never opened the conversation")
	}
	if toResponse(msg, Conversation{OtherLastReadAt: &earlier}).IsRead {
		t.Fatal("unread when read before the message was sent")
	}
	if !toResponse(msg, Conversation{OtherLastReadAt: &now}).IsRead {
		t.Fatal("read when last_read_at >= created_at")
	}
}
