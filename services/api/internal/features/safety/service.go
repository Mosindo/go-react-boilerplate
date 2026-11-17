package safety

import (
	"context"
	"slices"
	"strings"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrCannotTargetYou
	}
	ended, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	s.announceEnded(blockerID, blockedID, ended)
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, blockerID)
}

func (s *Service) Report(ctx context.Context, reporterID, reportedID, reason, details string) error {
	if reporterID == reportedID {
		return ErrCannotTargetYou
	}
	reason = strings.TrimSpace(reason)
	if !slices.Contains(ReportReasons, reason) {
		return ErrInvalidReason
	}
	normalized, err := profiles.NormalizeFreeText(details, maxReportDetailsRunes, ErrDetailsTooLong)
	if err != nil {
		return err
	}
	ended, err := s.repo.Report(ctx, reporterID, reportedID, reason, normalized)
	if err != nil {
		return err
	}
	s.announceEnded(reporterID, reportedID, ended)
	return nil
}

func (s *Service) announceEnded(actorID, otherID string, ended *EndedMatch) {
	if ended == nil || s.publisher == nil {
		return
	}
	payload := map[string]string{"matchId": ended.MatchID, "conversationId": ended.ConversationID}
	s.publisher.Publish(actorID, "match.ended", payload)
	s.publisher.Publish(otherID, "match.ended", payload)
}
