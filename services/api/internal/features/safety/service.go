package safety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

var ErrSelf = ValidationError{Message: "you cannot target yourself"}

type Service struct {
	repo   *Repository
	events realtime.Publisher
}

func NewService(repo *Repository, events realtime.Publisher) *Service {
	return &Service{repo: repo, events: events}
}

func (s *Service) Block(ctx context.Context, blockerID, targetID string) error {
	if blockerID == targetID {
		return ErrSelf
	}
	conversationID, err := s.repo.block(ctx, blockerID, targetID)
	if err != nil {
		return err
	}
	if conversationID != "" {
		evt := realtime.Event{Type: "conversation_removed", Data: map[string]string{"conversationId": conversationID}}
		s.events.Publish(blockerID, evt)
		s.events.Publish(targetID, evt)
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, targetID string) error {
	return s.repo.unblock(ctx, blockerID, targetID)
}

func (s *Service) ListBlocked(ctx context.Context, blockerID string, limit int, cur *httpx.Cursor) (httpx.Page[BlockedUser], error) {
	rows, err := s.repo.listBlocked(ctx, blockerID, limit, cur)
	if err != nil {
		return httpx.Page[BlockedUser]{}, err
	}
	return httpx.Paginate(rows, limit, func(b BlockedUser) httpx.Cursor { return httpx.Cursor{At: b.CreatedAt, ID: b.UserID} }), nil
}

func (s *Service) Report(ctx context.Context, reporterID, targetID, reason, details string) (string, error) {
	if reporterID == targetID {
		return "", ErrSelf
	}
	if _, ok := validReasons[reason]; !ok {
		return "", ValidationError{Message: "invalid reason"}
	}
	details = strings.TrimSpace(details)
	if utf8.RuneCountInString(details) > 1000 || !utf8.ValidString(details) {
		return "", ValidationError{Message: "details must be at most 1000 characters"}
	}
	return s.repo.report(ctx, reporterID, targetID, reason, details)
}

func IsNotFound(err error) bool { return errors.Is(err, ErrTargetNotFound) }
