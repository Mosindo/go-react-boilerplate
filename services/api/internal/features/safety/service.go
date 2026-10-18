package safety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var (
	ErrSelf           = errors.New("you cannot do that to yourself")
	ErrDetailsTooLong = errors.New("details must be at most 1000 characters")
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelf
	}
	matchIDs, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	for _, id := range matchIDs {
		s.publisher.Publish(blockedID, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": id}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) List(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, blockerID)
}

func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) (string, error) {
	if reporterID == req.UserID {
		return "", ErrSelf
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return "", ErrDetailsTooLong
	}
	id, err := s.repo.CreateReport(ctx, reporterID, req.UserID, req.Reason, details)
	if err != nil {
		return "", err
	}
	if req.Block {
		if err := s.Block(ctx, reporterID, req.UserID); err != nil {
			return "", err
		}
	}
	return id, nil
}
