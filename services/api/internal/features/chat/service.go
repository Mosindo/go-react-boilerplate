package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
)

type Service struct {
	repo      Repository
	signer    *storage.Signer
	publisher realtime.Publisher
}

func NewService(repo Repository, signer *storage.Signer, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, signer: signer, publisher: publisher}
}

func notFound() error { return perr.NotFound("conversation_not_found", "conversation not found") }

func mapErr(err error) error {
	if errors.Is(err, ErrNotFound) {
		return notFound()
	}
	return err
}

func (s *Service) ListConversations(ctx context.Context, me, before string, limit int) (ConversationPage, error) {
	var cursor *time.Time
	if before != "" {
		t, err := time.Parse(time.RFC3339Nano, before)
		if err != nil {
			return ConversationPage{}, perr.Invalid("invalid_cursor", "before must be an RFC 3339 timestamp")
		}
		cursor = &t
	}
	rows, err := s.repo.ListConversations(ctx, me, cursor, limit+1)
	if err != nil {
		return ConversationPage{}, err
	}
	page := ConversationPage{Items: []Conversation{}}
	if len(rows) > limit {
		rows = rows[:limit]
		next := rows[len(rows)-1].UpdatedAt.Format(time.RFC3339Nano)
		page.NextBefore = &next
	}
	for _, r := range rows {
		c := Conversation{
			ID: r.ID, MatchID: r.MatchID, UnreadCount: r.Unread, UpdatedAt: r.UpdatedAt,
			User: storage.UserSummary{ID: r.OtherID, FirstName: r.OtherName, ThumbURL: s.signer.ThumbURL(r.OtherPhotoID)},
		}
		if r.LastBody != nil && r.LastSender != nil && r.LastAt != nil {
			c.LastMessage = &LastMessage{Body: *r.LastBody, SenderID: *r.LastSender, CreatedAt: *r.LastAt}
		}
		page.Items = append(page.Items, c)
	}
	return page, nil
}

func (s *Service) Messages(ctx context.Context, me, convID string, beforeID int64, limit int) (MessagePage, error) {
	if !isUUID(convID) {
		return MessagePage{}, notFound()
	}
	msgs, otherRead, err := s.repo.Messages(ctx, me, convID, beforeID, limit+1)
	if err != nil {
		return MessagePage{}, mapErr(err)
	}
	page := MessagePage{Items: []Message{}, OtherLastReadAt: &otherRead}
	if len(msgs) > limit {
		msgs = msgs[:limit]
		next := msgs[len(msgs)-1].ID
		page.NextBefore = &next
	}
	page.Items = append(page.Items, msgs...)
	return page, nil
}

func (s *Service) Send(ctx context.Context, me, convID, body string) (Message, error) {
	if !isUUID(convID) {
		return Message{}, notFound()
	}
	body = cleanBody(body)
	if body == "" {
		return Message{}, perr.Invalid("empty_message", "message cannot be empty")
	}
	if utf8.RuneCountInString(body) > maxMessageRunes {
		return Message{}, perr.Invalid("message_too_long", "message must be at most 2000 characters")
	}
	msg, recipient, err := s.repo.Send(ctx, me, convID, body)
	if err != nil {
		return Message{}, mapErr(err)
	}
	s.publisher.Publish(recipient, realtime.Event{Type: "message.new", Data: msg})
	s.publisher.Publish(recipient, realtime.Event{Type: "notification.new", Data: map[string]string{"type": "message"}})
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, me, convID string) error {
	if !isUUID(convID) {
		return notFound()
	}
	other, err := s.repo.MarkRead(ctx, me, convID)
	if err != nil {
		return mapErr(err)
	}
	s.publisher.Publish(other, realtime.Event{Type: "conversation.read", Data: map[string]any{
		"conversationId": convID, "userId": me, "readAt": time.Now().UTC(),
	}})
	return nil
}

// Hide removes the conversation from the caller's list only (local deletion). It reappears when a
// new message arrives, showing only messages newer than the deletion.
func (s *Service) Hide(ctx context.Context, me, convID string) error {
	if !isUUID(convID) {
		return notFound()
	}
	return mapErr(s.repo.Hide(ctx, me, convID))
}

// cleanBody trims and drops control characters other than newlines/tabs.
func cleanBody(v string) string {
	v = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, v)
	return strings.TrimSpace(v)
}

func isUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, r := range v {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if r != '-' {
				return false
			}
		case !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'):
			return false
		}
	}
	return true
}
