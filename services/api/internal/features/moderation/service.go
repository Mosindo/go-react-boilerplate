package moderation

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

var errUserNotFound = platformerrors.NotFound("user_not_found", "user not found")

func normalizeTarget(self, target string) (string, error) {
	target = strings.ToLower(target)
	if !httpx.IsUUID(target) {
		return "", errUserNotFound
	}
	if target == self {
		return "", platformerrors.BadRequest("invalid_target", "you cannot target yourself")
	}
	return target, nil
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	target, err := normalizeTarget(userID, targetID)
	if err != nil {
		return err
	}
	closed, err := s.repo.Block(ctx, userID, target)
	if err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return errUserNotFound
		}
		return err
	}
	s.announceRemoved(userID, target, closed)
	return nil
}

func (s *Service) announceRemoved(a, b string, matchIDs []string) {
	for _, id := range matchIDs {
		payload := map[string]string{"matchId": id}
		s.pub.Publish(a, "match.removed", payload)
		s.pub.Publish(b, "match.removed", payload)
	}
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	target, err := normalizeTarget(userID, targetID)
	if err != nil {
		return err
	}
	return s.repo.Unblock(ctx, userID, target)
}

func (s *Service) ListBlocked(ctx context.Context, userID string) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, userID)
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	target, err := normalizeTarget(userID, req.UserID)
	if err != nil {
		return err
	}
	fields := map[string]string{}
	if _, ok := validReasons[req.Reason]; !ok {
		fields["reason"] = "invalid reason"
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > maxDetailsRunes || !utf8.ValidString(details) {
		fields["details"] = "must be at most 1000 characters"
	}
	matchID := strings.ToLower(req.MatchID)
	if matchID != "" && !httpx.IsUUID(matchID) {
		fields["matchId"] = "invalid match id"
	}
	if len(fields) > 0 {
		return platformerrors.Validation(fields)
	}
	if _, err := s.repo.Report(ctx, userID, target, matchID, req.Reason, details, SuspensionThreshold); err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return errUserNotFound
		}
		return err
	}
	if req.Block {
		return s.Block(ctx, userID, target)
	}
	return nil
}
