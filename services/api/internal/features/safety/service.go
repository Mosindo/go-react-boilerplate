package safety

import (
	"context"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"
)

var (
	ErrSelf          = errors.New("you cannot do this on yourself")
	ErrUserNotFound  = errors.New("user not found")
	ErrInvalidReason = errors.New("invalid report reason")
	ErrDetailsLong   = errors.New("details are too long (1000 characters max)")
)

// MatchRemover is implemented by the matches module (realtime fan-out).
type MatchRemover interface {
	PublishRemoved(matchID string, userIDs ...string)
}

type Service struct {
	repo    Repository
	matches MatchRemover
}

func NewService(repo Repository, matches MatchRemover) *Service {
	return &Service{repo: repo, matches: matches}
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	targetID = strings.ToLower(targetID)
	if userID == targetID {
		return ErrSelf
	}
	matchID, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		if errors.Is(err, ErrRepoUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if matchID != "" {
		s.matches.PublishRemoved(matchID, userID, targetID)
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, strings.ToLower(targetID))
}

func (s *Service) ListBlocked(ctx context.Context, userID string) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, userID)
}

func ValidateReport(reason, details string) (string, error) {
	if !slices.Contains(ReportReasons, reason) {
		return "", ErrInvalidReason
	}
	details = strings.TrimSpace(details)
	if utf8.RuneCountInString(details) > MaxDetailsRunes {
		return "", ErrDetailsLong
	}
	return details, nil
}

func (s *Service) Report(ctx context.Context, userID, targetID, reason, details string) error {
	targetID = strings.ToLower(targetID)
	if userID == targetID {
		return ErrSelf
	}
	details, err := ValidateReport(reason, details)
	if err != nil {
		return err
	}
	matchID, err := s.repo.Report(ctx, userID, targetID, reason, details, AutoReviewThreshold)
	if err != nil {
		if errors.Is(err, ErrRepoUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if matchID != "" {
		s.matches.PublishRemoved(matchID, userID, targetID)
	}
	return nil
}
