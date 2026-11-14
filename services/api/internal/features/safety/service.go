package safety

import (
	"context"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var (
	ErrSelf          = errors.New("cannot target yourself")
	ErrInvalidReason = errors.New("invalid reason")
	ErrInvalidDetail = errors.New("invalid details")
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) notifyRemoved(blockerID, blockedID, matchID string) {
	if matchID == "" || s.pub == nil {
		return
	}
	event := realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}}
	s.pub.Publish(blockedID, event)
	s.pub.Publish(blockerID, event)
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelf
	}
	matchID, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	s.notifyRemoved(blockerID, blockedID, matchID)
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) Blocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, blockerID, limit, offset)
}

func (s *Service) Report(ctx context.Context, reporterID, reportedID, reason, details string) error {
	if reporterID == reportedID {
		return ErrSelf
	}
	if !slices.Contains(ReportReasons, reason) {
		return ErrInvalidReason
	}
	details = strings.TrimSpace(details)
	if utf8.RuneCountInString(details) > 1000 {
		return ErrInvalidDetail
	}
	return s.repo.Report(ctx, reporterID, reportedID, reason, details, SuspensionThreshold)
}
