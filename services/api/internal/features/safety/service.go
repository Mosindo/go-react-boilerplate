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
	ErrSelfAction     = errors.New("you cannot do this on your own profile")
	ErrInvalidReason  = errors.New("unsupported report reason")
	ErrDetailsTooLong = errors.New("details must be at most 1000 characters")
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.Nop{}
	}
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

func (s *Service) notifyRemoval(actorID, targetID string) {
	// Both sides drop the conversation from their UI immediately.
	s.publisher.Publish(targetID, "match.removed", map[string]string{"userId": actorID})
	s.publisher.Publish(actorID, "match.removed", map[string]string{"userId": targetID})
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if err := s.ensureTarget(ctx, blockerID, blockedID); err != nil {
		return err
	}
	hadMatch, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if hadMatch {
		s.notifyRemoval(blockerID, blockedID)
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, blockerID, limit, offset)
}

// ValidateReport normalizes report input.
func ValidateReport(reason, details string) (string, string, error) {
	r := strings.ToLower(strings.TrimSpace(reason))
	if !slices.Contains(ReportReasons, r) {
		return "", "", ErrInvalidReason
	}
	d := strings.TrimSpace(details)
	if utf8.RuneCountInString(d) > MaxReportDetailsRunes {
		return "", "", ErrDetailsTooLong
	}
	return r, d, nil
}

// Report always blocks the reported member for the reporter (safest default)
// and hides the profile pending review once enough distinct members report it.
func (s *Service) Report(ctx context.Context, reporterID, reportedID, reason, details string) error {
	r, d, err := ValidateReport(reason, details)
	if err != nil {
		return err
	}
	if err := s.ensureTarget(ctx, reporterID, reportedID); err != nil {
		return err
	}
	if err := s.repo.Report(ctx, reporterID, reportedID, r, d, AutoHideReportThreshold, UnderageHideThreshold); err != nil {
		return err
	}
	s.notifyRemoval(reporterID, reportedID)
	return nil
}
