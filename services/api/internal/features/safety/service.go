package safety

import (
	"context"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo   Repository
	events realtime.Publisher
}

func NewService(repo Repository, events realtime.Publisher) *Service {
	return &Service{repo: repo, events: events}
}

func validateTarget(userID, targetID string) error {
	if !profiles.IsUUID(targetID) {
		return errUserNotFound
	}
	if userID == targetID {
		return platformerrors.BadRequest("you cannot target yourself")
	}
	return nil
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if err := validateTarget(userID, targetID); err != nil {
		return err
	}
	removed, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		return err
	}
	if removed != nil {
		event := realtime.Event{"type": "match.removed", "matchId": removed.MatchID, "conversationId": removed.ConversationID}
		s.events.Publish(userID, event)
		s.events.Publish(targetID, event)
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	if !profiles.IsUUID(targetID) {
		return errUserNotFound
	}
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) ListBlocks(ctx context.Context, userID string) (BlocksResult, error) {
	list, err := s.repo.ListBlocks(ctx, userID)
	return BlocksResult{Blocks: list}, err
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	if err := validateTarget(userID, req.UserID); err != nil {
		return err
	}
	if _, ok := validReasons[req.Reason]; !ok {
		return platformerrors.BadRequest("invalid report reason")
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return platformerrors.BadRequest("details must be at most 1000 characters")
	}
	return s.repo.CreateReport(ctx, userID, req.UserID, req.Reason, details)
}
