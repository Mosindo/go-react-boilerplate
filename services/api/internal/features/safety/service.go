package safety

import (
	"context"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrSelfAction    = apperr.BadRequest("self_action", "you cannot do this on your own account")
	ErrUserNotFound  = apperr.NotFound("user_not_found", "user not found")
	ErrNotBlocked    = apperr.NotFound("block_not_found", "this member is not blocked")
	ErrInvalidReason = apperr.BadRequest("invalid_reason", "unknown report reason")
	ErrDetailsLength = apperr.BadRequest("details_too_long", "details must be 1000 characters or less")
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) ensureTarget(ctx context.Context, actorID, targetID string) error {
	if actorID == targetID {
		return ErrSelfAction
	}
	ok, err := s.repo.UserExists(ctx, targetID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUserNotFound
	}
	return nil
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if err := s.ensureTarget(ctx, blockerID, blockedID); err != nil {
		return err
	}
	matchID, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if matchID != "" {
		event := realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}}
		s.publisher.Publish(blockedID, event)
		s.publisher.Publish(blockerID, event)
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	if err := s.repo.Unblock(ctx, blockerID, blockedID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrNotBlocked
		}
		return err
	}
	return nil
}

func (s *Service) ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, blockerID)
}

func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) (ReportResponse, error) {
	reportedID := strings.ToLower(req.UserID)
	reason := strings.TrimSpace(req.Reason)
	if !slices.Contains(ReportReasons, reason) {
		return ReportResponse{}, ErrInvalidReason
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return ReportResponse{}, ErrDetailsLength
	}
	if err := s.ensureTarget(ctx, reporterID, reportedID); err != nil {
		return ReportResponse{}, err
	}
	id, err := s.repo.CreateReport(ctx, reporterID, reportedID, reason, details)
	if err != nil {
		return ReportResponse{}, err
	}
	resp := ReportResponse{ID: id, Status: "open"}
	if req.Block == nil || *req.Block {
		if err := s.Block(ctx, reporterID, reportedID); err != nil {
			return ReportResponse{}, err
		}
		resp.Blocked = true
	}
	return resp, nil
}
