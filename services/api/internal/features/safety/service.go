package safety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
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

func (s *Service) checkTarget(userID, targetID string) error {
	if !profiles.IsUUID(targetID) {
		return platformerrors.Invalid("invalid user id")
	}
	if targetID == userID {
		return platformerrors.Invalid("you cannot do this to yourself")
	}
	return nil
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if err := s.checkTarget(userID, targetID); err != nil {
		return err
	}
	removed, err := s.repo.Block(ctx, userID, targetID)
	if errors.Is(err, ErrUserNotFound) {
		return platformerrors.NotFound("user not found")
	}
	if err != nil {
		return err
	}
	if removed != nil {
		// The blocked person only learns that the match is gone, never why.
		s.publisher.Publish(targetID, realtime.Event{Type: "unmatch", Data: map[string]string{
			"matchId": removed.MatchID, "conversationId": removed.ConversationID,
		}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	if !profiles.IsUUID(targetID) {
		return platformerrors.Invalid("invalid user id")
	}
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) ListBlocked(ctx context.Context, userID string) ([]BlockedUser, error) {
	items, err := s.repo.ListBlocked(ctx, userID)
	if items == nil {
		items = []BlockedUser{}
	}
	return items, err
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	if err := s.checkTarget(userID, req.UserID); err != nil {
		return err
	}
	if _, ok := validReasons[req.Reason]; !ok {
		return platformerrors.Invalid("unknown report reason")
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > maxDetails {
		return platformerrors.Invalid("details must be at most %d characters", maxDetails)
	}
	if err := s.repo.CreateReport(ctx, userID, req.UserID, req.Reason, details); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return platformerrors.NotFound("user not found")
		}
		return err
	}
	if req.Block {
		return s.Block(ctx, userID, req.UserID)
	}
	return nil
}
