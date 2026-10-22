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
	repo     Repository
	realtime realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, realtime: publisher}
}

func (s *Service) validateTarget(userID, targetID string) error {
	if !profiles.IsUUID(targetID) {
		return ErrUserNotFound
	}
	if userID == targetID {
		return platformerrors.Invalid("userId", "action impossible sur votre propre compte")
	}
	return nil
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if err := s.validateTarget(userID, targetID); err != nil {
		return err
	}
	matchID, conversationID, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		return err
	}
	if matchID != "" {
		payload := map[string]string{"matchId": matchID, "conversationId": conversationID}
		s.realtime.Publish(targetID, realtime.Event{Type: "match.removed", Data: payload})
		s.realtime.Publish(userID, realtime.Event{Type: "match.removed", Data: payload})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	if err := s.validateTarget(userID, targetID); err != nil {
		return err
	}
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) ListBlocked(ctx context.Context, userID string) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, userID)
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	if err := s.validateTarget(userID, req.UserID); err != nil {
		return err
	}
	valid := false
	for _, reason := range ReportReasons {
		if reason == req.Reason {
			valid = true
		}
	}
	if !valid {
		return platformerrors.Invalid("reason", "motif de signalement invalide")
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return platformerrors.Invalid("details", "1000 caractères maximum")
	}
	if err := s.repo.CreateReport(ctx, userID, req.UserID, req.Reason, details); err != nil {
		return err
	}
	if req.AlsoBlock {
		return s.Block(ctx, userID, req.UserID)
	}
	return nil
}
