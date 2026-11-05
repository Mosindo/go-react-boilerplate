package safety

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalid = errors.New("invalid request")
)

type Service struct {
	repo   Repository
	events Events
}

func NewService(repo Repository, events Events) *Service {
	return &Service{repo: repo, events: events}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrInvalid
	}
	ended, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if ended != nil && s.events != nil {
		s.events.MatchEnded(ended.MatchID, ended.ConversationID, blockerID, blockedID)
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) List(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, blockerID, 200)
}

func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) error {
	if reporterID == req.UserID || !validReasons[req.Reason] {
		return ErrInvalid
	}
	details := strings.TrimSpace(req.Details)
	if err := s.repo.CreateReport(ctx, reporterID, req.UserID, req.Reason, details); err != nil {
		return err
	}
	if req.AlsoBlock {
		return s.Block(ctx, reporterID, req.UserID)
	}
	return nil
}
