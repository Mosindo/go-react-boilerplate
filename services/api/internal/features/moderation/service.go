package moderation

import (
	"context"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo *Repository
	pub  Publisher
}

func NewService(repo *Repository, pub Publisher) *Service { return &Service{repo: repo, pub: pub} }

// Block hides both users from each other everywhere and removes any match (with its conversation).
func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelf
	}
	ok, err := s.repo.UserExists(ctx, blockedID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	removed, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if removed {
		s.pub.Publish(blockedID, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": blockerID}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, blockerID)
}

// Report files a moderation report; repeating the same open report within 24h is idempotent.
func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) error {
	if reporterID == req.UserID {
		return ErrSelf
	}
	if !reasons[req.Reason] {
		return ErrInvalidReason
	}
	details := profiles.CleanText(req.Details)
	if utf8.RuneCountInString(details) > MaxDetailsRunes {
		return ErrDetailsLong
	}
	ok, err := s.repo.UserExists(ctx, req.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	if err := s.repo.InsertReport(ctx, reporterID, req.UserID, req.Reason, details); err != nil {
		return err
	}
	if req.AlsoBlock {
		return s.Block(ctx, reporterID, req.UserID)
	}
	return nil
}
