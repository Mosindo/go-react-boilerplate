package safety

import (
	"context"
	"errors"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func parseTarget(userID, raw string) (string, error) {
	target := strings.ToLower(strings.TrimSpace(raw))
	if !platformerrors.IsUUID(target) {
		return "", platformerrors.BadRequest("invalid_user", "invalid user id")
	}
	if target == userID {
		return "", platformerrors.BadRequest("invalid_user", "you cannot target yourself")
	}
	return target, nil
}

var errUnknownUser = platformerrors.NotFound("not_found", "user not found")

func (s *Service) Block(ctx context.Context, userID, rawTarget string) error {
	target, err := parseTarget(userID, rawTarget)
	if err != nil {
		return err
	}
	matchIDs, err := s.repo.Block(ctx, userID, target)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return errUnknownUser
		}
		return err
	}
	for _, id := range matchIDs {
		s.publisher.Publish(target, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": id}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, rawTarget string) error {
	target, err := parseTarget(userID, rawTarget)
	if err != nil {
		return err
	}
	return s.repo.Unblock(ctx, userID, target)
}

func (s *Service) ListBlocked(ctx context.Context, userID string) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, userID)
}

// Report files a report for human review. A duplicate open report for the same pair is accepted silently.
func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	target, err := parseTarget(userID, req.UserID)
	if err != nil {
		return err
	}
	if !validReasons[req.Reason] {
		return platformerrors.BadRequest("invalid_reason", "invalid report reason")
	}
	details := strings.TrimSpace(req.Details)
	if len([]rune(details)) > 1000 {
		return platformerrors.BadRequest("invalid_details", "details must be at most 1000 characters")
	}
	if _, err := s.repo.Report(ctx, userID, target, req.Reason, details); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return errUnknownUser
		}
		return err
	}
	if req.AlsoBlock {
		return s.Block(ctx, userID, target)
	}
	return nil
}
