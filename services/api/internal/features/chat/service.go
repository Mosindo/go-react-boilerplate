package chat

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

const MaxMessageRunes = 2000

type Service struct {
	repo      *Repository
	publisher realtime.Publisher
}

func NewService(repo *Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func mapErr(err error) error {
	if errors.Is(err, ErrNotParticipant) {
		return platformerrors.ErrNotFound
	}
	return err
}

// NormalizeBody trims and validates message text.
func NormalizeBody(raw string) (string, error) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return "", platformerrors.Invalid("message cannot be empty")
	}
	if utf8.RuneCountInString(body) > MaxMessageRunes {
		return "", platformerrors.Invalid("message is too long")
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", platformerrors.Invalid("message contains invalid characters")
		}
	}
	return body, nil
}

func (s *Service) Conversations(ctx context.Context, userID string, before time.Time, limit int) ([]Conversation, error) {
	return s.repo.ListConversations(ctx, userID, before, limit)
}

func (s *Service) Messages(ctx context.Context, convID, userID string, beforeID int64, limit int) (MessagesPage, error) {
	if beforeID <= 0 {
		beforeID = math.MaxInt64
	}
	page, err := s.repo.Messages(ctx, convID, userID, beforeID, limit)
	return page, mapErr(err)
}

func (s *Service) Send(ctx context.Context, convID, userID, rawBody string) (Message, error) {
	body, err := NormalizeBody(rawBody)
	if err != nil {
		return Message{}, err
	}
	msg, other, err := s.repo.Send(ctx, convID, userID, body)
	if err != nil {
		return Message{}, mapErr(err)
	}
	ev := realtime.Event{Type: "message", Data: msg}
	s.publisher.Publish(other, ev)
	s.publisher.Publish(userID, ev)
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, convID, userID string) error {
	upTo, other, err := s.repo.MarkRead(ctx, convID, userID)
	if err != nil {
		return mapErr(err)
	}
	s.publisher.Publish(other, realtime.Event{Type: "read", Data: map[string]any{
		"conversationId": convID, "readerId": userID, "upToMessageId": upTo,
	}})
	return nil
}

func (s *Service) Hide(ctx context.Context, convID, userID string) error {
	return mapErr(s.repo.Hide(ctx, convID, userID))
}
