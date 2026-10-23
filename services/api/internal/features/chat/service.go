package chat

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/pagination"
	"example.com/api/internal/platform/realtime"
)

var errNotFound = platformerrors.NotFound("match_not_found", "conversation not found")

// Notifier is implemented by the notifications feature.
type Notifier interface {
	NotifyMessage(ctx context.Context, recipientID, matchID, senderID, senderName, preview string) error
	MarkConversationRead(ctx context.Context, userID, matchID string) error
}

type Service struct {
	repo     Repository
	notifier Notifier
	pub      realtime.Publisher
}

func NewService(repo Repository, notifier Notifier, pub realtime.Publisher) *Service {
	return &Service{repo: repo, notifier: notifier, pub: pub}
}

// NormalizeContent trims a message and enforces length and character rules.
func NormalizeContent(raw string) (string, error) {
	content := strings.TrimSpace(raw)
	if content == "" {
		return "", platformerrors.Validation(map[string]string{"content": "message is empty"})
	}
	if !utf8.ValidString(content) || utf8.RuneCountInString(content) > MaxContentRune {
		return "", platformerrors.Validation(map[string]string{"content": "message must be at most 2000 characters"})
	}
	for _, r := range content {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", platformerrors.Validation(map[string]string{"content": "message contains invalid characters"})
		}
	}
	return content, nil
}

func (s *Service) Send(ctx context.Context, userID, matchID, raw string) (Message, error) {
	content, err := NormalizeContent(raw)
	if err != nil {
		return Message{}, err
	}
	member, err := s.repo.Member(ctx, matchID, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Message{}, errNotFound
		}
		return Message{}, err
	}
	msg, err := s.repo.Insert(ctx, matchID, userID, content)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Message{}, errNotFound
		}
		return Message{}, err
	}
	s.pub.Publish(member.OtherUserID, "message.new", msg)
	s.pub.Publish(userID, "message.new", msg) // other devices of the sender
	if err := s.notifier.NotifyMessage(ctx, member.OtherUserID, matchID, userID, member.SelfName, content); err != nil {
		log.Printf(`{"event":"message_notification_failed","error":%q}`, err.Error())
	}
	return msg, nil
}

func (s *Service) List(ctx context.Context, userID, matchID string, limit int, cursor string) (MessagePage, error) {
	var before *time.Time
	var beforeID string
	if cursor != "" {
		t, id, ok := pagination.Decode(cursor)
		if !ok {
			return MessagePage{}, platformerrors.BadRequest("invalid_cursor", "invalid cursor")
		}
		before, beforeID = &t, id
	}
	if _, err := s.repo.Member(ctx, matchID, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return MessagePage{}, errNotFound
		}
		return MessagePage{}, err
	}
	items, err := s.repo.List(ctx, matchID, userID, limit+1, before, beforeID)
	if err != nil {
		return MessagePage{}, err
	}
	var next *string
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		c := pagination.Encode(last.CreatedAt, last.ID)
		next = &c
	}
	return MessagePage{Messages: items, NextCursor: next}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, matchID string) error {
	count, readAt, other, err := s.repo.MarkRead(ctx, matchID, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return errNotFound
		}
		return err
	}
	if err := s.notifier.MarkConversationRead(ctx, userID, matchID); err != nil {
		log.Printf(`{"event":"mark_notification_read_failed","error":%q}`, err.Error())
	}
	if count > 0 {
		s.pub.Publish(other, "message.read", ReadReceipt{MatchID: matchID, ReaderID: userID, ReadAt: readAt})
	}
	return nil
}
