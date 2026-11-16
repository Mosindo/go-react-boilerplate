package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var (
	ErrEmptyMessage   = errors.New("message cannot be empty")
	ErrMessageTooLong = errors.New("message is too long (2000 characters max)")
)

type Notifier interface {
	NotifyMessage(ctx context.Context, recipientID, senderID, matchID string)
	ConversationRead(ctx context.Context, userID, matchID string)
}

type Service struct {
	repo      Repository
	notifier  Notifier
	publisher realtime.Publisher
	now       func() time.Time
}

func NewService(repo Repository, notifier Notifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, notifier: notifier, publisher: publisher, now: time.Now}
}

// NormalizeBody trims the message and strips control characters except
// newlines and tabs.
func NormalizeBody(body string) (string, error) {
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(body))
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(cleaned) > MaxMessageLength {
		return "", ErrMessageTooLong
	}
	return cleaned, nil
}

func (s *Service) List(ctx context.Context, userID, matchID, before string, limit int) (ListResponse, error) {
	if _, err := s.repo.Membership(ctx, userID, matchID); err != nil {
		return ListResponse{}, err
	}
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	items, err := s.repo.List(ctx, matchID, before, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{Messages: items}
	if len(items) > limit {
		resp.Messages = items[:limit]
		last := resp.Messages[limit-1].ID
		resp.NextBefore = &last
	}
	if resp.Messages == nil {
		resp.Messages = []Message{}
	}
	return resp, nil
}

func (s *Service) Send(ctx context.Context, userID, matchID, body, clientID string) (Message, error) {
	other, err := s.repo.Membership(ctx, userID, matchID)
	if err != nil {
		return Message{}, err
	}
	normalized, err := NormalizeBody(body)
	if err != nil {
		return Message{}, err
	}
	var cid *string
	if clientID != "" {
		lower := strings.ToLower(clientID)
		cid = &lower
	}
	msg, created, err := s.repo.Insert(ctx, matchID, userID, normalized, cid)
	if err != nil {
		return Message{}, err
	}
	if created {
		event := realtime.Event{Type: "message.created", Data: msg}
		s.publisher.Publish(other, event)
		s.publisher.Publish(userID, event)
		s.notifier.NotifyMessage(ctx, other, userID, matchID)
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Membership(ctx, userID, matchID)
	if err != nil {
		return err
	}
	at := s.now().UTC()
	n, err := s.repo.MarkRead(ctx, matchID, userID, at)
	if err != nil {
		return err
	}
	s.notifier.ConversationRead(ctx, userID, matchID)
	if n > 0 {
		s.publisher.Publish(other, realtime.Event{Type: "conversation.read", Data: ReadEvent{MatchID: matchID, ReaderID: userID, ReadAt: at}})
	}
	return nil
}
