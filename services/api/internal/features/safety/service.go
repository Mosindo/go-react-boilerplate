package safety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var (
	ErrSelf           = errors.New("cannot target yourself")
	ErrDetailsTooLong = errors.New("details too long")
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return ErrSelf
	}
	removed, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		return err
	}
	if removed != "" {
		payload := map[string]string{"matchId": removed}
		s.pub.Publish(userID, realtime.Event{Type: realtime.EventMatchRemoved, Data: payload})
		s.pub.Publish(targetID, realtime.Event{Type: realtime.EventMatchRemoved, Data: payload})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) ListBlocks(ctx context.Context, userID string) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, userID, 200)
}

// Report files a moderation report and blocks the reported member so the
// reporter is protected immediately.
func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) (string, error) {
	target := strings.ToLower(req.UserID)
	if userID == target {
		return "", ErrSelf
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return "", ErrDetailsTooLong
	}
	id, err := s.repo.CreateReport(ctx, userID, target, req.Reason, details)
	if err != nil {
		return "", err
	}
	if err := s.Block(ctx, userID, target); err != nil {
		return "", err
	}
	return id, nil
}
