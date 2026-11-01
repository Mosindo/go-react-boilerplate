package moderation

import (
	"context"
	"strings"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	rt   realtime.Publisher
}

func NewService(repo Repository, rt realtime.Publisher) *Service { return &Service{repo: repo, rt: rt} }

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return ErrSelf
	}
	if err := s.repo.Block(ctx, userID, targetID); err != nil {
		return err
	}
	// Le bloque n'est pas informe du blocage; son client voit simplement le match disparaitre.
	s.rt.Publish(targetID, realtime.Event{Type: "unmatch", Payload: map[string]string{}})
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) Blocks(ctx context.Context, userID string) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, userID)
}

func (s *Service) Report(ctx context.Context, userID, targetID, reason, details string) error {
	if userID == targetID {
		return ErrSelf
	}
	reason = strings.ToLower(strings.TrimSpace(reason))
	if _, ok := validReasons[reason]; !ok {
		return ErrReason
	}
	return s.repo.Report(ctx, userID, targetID, reason, strings.TrimSpace(details))
}
