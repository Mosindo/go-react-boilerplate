package safety

import (
	"context"
	"strings"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

type Events interface {
	Publish(userID string, event realtime.Event)
}

type Service struct {
	repo   Repository
	events Events
}

func NewService(repo Repository, events Events) *Service {
	return &Service{repo: repo, events: events}
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return platformerrors.Invalid("userId", "you cannot block yourself")
	}
	if _, err := s.repo.Block(ctx, userID, targetID); err != nil {
		return err
	}
	s.events.Publish(targetID, realtime.Event{Type: realtime.TypeUnmatch})
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) ListBlocked(ctx context.Context, userID string) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, userID)
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	if userID == strings.ToLower(req.UserID) {
		return platformerrors.Invalid("userId", "you cannot report yourself")
	}
	reason := strings.TrimSpace(req.Reason)
	if _, ok := validReasons[reason]; !ok {
		return platformerrors.Invalid("reason", "is not a valid report reason")
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > maxReportDetails {
		return platformerrors.Invalid("details", "is too long (max 1000 characters)")
	}
	if err := s.repo.Report(ctx, userID, strings.ToLower(req.UserID), reason, details); err != nil {
		return err
	}
	if req.Block {
		return s.Block(ctx, userID, strings.ToLower(req.UserID))
	}
	return nil
}
