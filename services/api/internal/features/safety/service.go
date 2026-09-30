package safety

import (
	"context"
	"errors"
	"log"

	"example.com/api/internal/features/profiles"
	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrSelfAction   = apperr.Validation("you cannot block or report yourself")
	ErrUserNotFound = apperr.NotFound("user not found")
)

type SummaryProvider interface {
	Summaries(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Summary, error)
}

type Service struct {
	repo      Repository
	profiles  SummaryProvider
	publisher realtime.Publisher
}

func NewService(repo Repository, profiles SummaryProvider, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, publisher: publisher}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelfAction
	}
	matchID, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if matchID != "" {
		// The blocked person simply sees the match disappear.
		s.publisher.Publish(ctx, []string{blockerID, blockedID}, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := s.repo.ListBlocked(ctx, blockerID, 200)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.UserID
	}
	summaries, err := s.profiles.Summaries(ctx, blockerID, ids)
	if err != nil {
		return nil, err
	}
	items := make([]BlockedUser, 0, len(rows))
	for _, row := range rows {
		summary, ok := summaries[row.UserID]
		if !ok {
			summary = profiles.Summary{UserID: row.UserID}
		}
		items = append(items, BlockedUser{User: summary, BlockedAt: row.BlockedAt})
	}
	return items, nil
}

// Report files a moderation report and, by default, blocks the person.
func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) error {
	if reporterID == req.UserID {
		return ErrSelfAction
	}
	details, err := profiles.CleanText(req.Details, 1000, "details", true)
	if err != nil {
		return err
	}
	if err := s.repo.CreateReport(ctx, reporterID, req.UserID, req.Reason, details); err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	log.Printf(`{"event":"user_reported","reason":%q}`, req.Reason)
	if req.Block == nil || *req.Block {
		return s.Block(ctx, reporterID, req.UserID)
	}
	return nil
}
