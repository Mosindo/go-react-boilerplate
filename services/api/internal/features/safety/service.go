package safety

import (
	"context"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

const maxDetailsLength = 1000

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

// Block hides both users from each other everywhere (discovery, profile,
// matches, chat). The blocked user only sees the match disappear, exactly as
// after an unmatch, so blocking is not disclosed.
func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	blockedID = strings.ToLower(blockedID)
	if blockerID == blockedID {
		return ErrSelf
	}
	matchID, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if matchID != "" {
		ev := realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}}
		s.publisher.Publish(blockedID, ev)
		s.publisher.Publish(blockerID, ev)
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, strings.ToLower(blockedID))
}

func (s *Service) ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	items, err := s.repo.ListBlocks(ctx, blockerID)
	if items == nil {
		items = []BlockedUser{}
	}
	return items, err
}

// Report records a moderation report and also blocks the reported user, the
// safest default for the reporter.
func (s *Service) Report(ctx context.Context, reporterID, reportedID, reason, details string) error {
	reportedID = strings.ToLower(reportedID)
	if reporterID == reportedID {
		return ErrSelf
	}
	details = strings.TrimSpace(details)
	if utf8.RuneCountInString(details) > maxDetailsLength {
		details = string([]rune(details)[:maxDetailsLength])
	}
	if err := s.Block(ctx, reporterID, reportedID); err != nil {
		return err
	}
	return s.repo.UpsertReport(ctx, reporterID, reportedID, reason, details)
}
