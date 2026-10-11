package safety

import (
	"context"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo *Repository
	pub  realtime.Publisher
}

func NewService(repo *Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) Block(ctx context.Context, blocker, blocked string) error {
	if blocker == blocked {
		return ErrSelf
	}
	rm, err := s.repo.Block(ctx, blocker, blocked)
	if err != nil {
		return err
	}
	if rm != nil {
		for _, id := range []string{rm.UserA, rm.UserB} {
			s.pub.Publish(id, "conversation.removed", map[string]string{"conversationId": rm.ConversationID})
		}
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blocker, blocked string) error {
	return s.repo.Unblock(ctx, blocker, blocked)
}

func (s *Service) List(ctx context.Context, blocker string) ([]Block, error) {
	return s.repo.List(ctx, blocker)
}

func (s *Service) Report(ctx context.Context, reporter, reported, reason, details string) (string, error) {
	if reporter == reported {
		return "", ErrSelf
	}
	if !validReasons[reason] {
		return "", ErrBadReason
	}
	details = strings.TrimSpace(details)
	if utf8.RuneCountInString(details) > 1000 {
		return "", ErrLongDetail
	}
	return s.repo.Report(ctx, reporter, reported, reason, details)
}
