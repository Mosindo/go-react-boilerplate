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
	ErrSelfAction    = errors.New("you cannot block or report yourself")
	ErrUserNotFound  = errors.New("user not found")
	ErrInvalidReason = errors.New("invalid report reason")
	ErrDetailsLength = errors.New("details cannot exceed 1000 characters")
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) announceRemoval(userID, otherID string, removed *RemovedMatch) {
	if removed == nil {
		return
	}
	event := realtime.Event{Type: "match.removed", Data: map[string]string{
		"matchId": removed.MatchID, "conversationId": removed.ConversationID,
	}}
	s.publisher.Publish(userID, event)
	s.publisher.Publish(otherID, event)
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelfAction
	}
	removed, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	s.announceRemoval(blockerID, blockedID, removed)
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, blockerID, limit, offset)
}

// ValidateReport checks a report payload.
func ValidateReport(reason, details string) (string, string, error) {
	reason = strings.TrimSpace(reason)
	if !slices.Contains(ReportReasons, reason) {
		return "", "", ErrInvalidReason
	}
	details = strings.TrimSpace(details)
	if utf8.RuneCountInString(details) > maxReportDetails {
		return "", "", ErrDetailsLength
	}
	return reason, details, nil
}

// Report files a report and blocks the reported member for the reporter.
func (s *Service) Report(ctx context.Context, reporterID, reportedID, reason, details string) error {
	if reporterID == reportedID {
		return ErrSelfAction
	}
	reason, details, err := ValidateReport(reason, details)
	if err != nil {
		return err
	}
	removed, err := s.repo.CreateReport(ctx, reporterID, reportedID, reason, details, reviewThreshold, reviewWindow)
	if err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	s.announceRemoval(reporterID, reportedID, removed)
	return nil
}
