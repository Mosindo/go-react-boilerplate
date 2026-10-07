package safety

import (
	"context"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
)

type SummaryReader interface {
	Summaries(ctx context.Context, userIDs []string) (map[string]profiles.UserSummary, error)
}

type Publisher interface {
	Publish(userID, eventType string, data any)
}

type Service struct {
	repo     *Repository
	profiles SummaryReader
	events   Publisher
}

func NewService(repo *Repository, reader SummaryReader, events Publisher) *Service {
	return &Service{repo: repo, profiles: reader, events: events}
}

func (s *Service) target(ctx context.Context, userID, targetID string) error {
	targetID = strings.ToLower(targetID)
	if userID == targetID {
		return platformerrors.Validation("you cannot do this to yourself")
	}
	ok, err := s.repo.UserExists(ctx, targetID)
	if err != nil {
		return err
	}
	if !ok {
		return platformerrors.NotFound("user not found")
	}
	return nil
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	targetID = strings.ToLower(targetID)
	if err := s.target(ctx, userID, targetID); err != nil {
		return err
	}
	conversations, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		return err
	}
	for _, id := range conversations {
		s.events.Publish(targetID, "unmatch", map[string]string{"conversationId": id})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, strings.ToLower(targetID))
}

func (s *Service) Blocked(ctx context.Context, userID string) ([]profiles.UserSummary, error) {
	ids, err := s.repo.Blocked(ctx, userID)
	if err != nil {
		return nil, err
	}
	summaries, err := s.profiles.Summaries(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]profiles.UserSummary, 0, len(ids))
	for _, id := range ids {
		if summary, ok := summaries[id]; ok {
			out = append(out, summary)
		} else {
			out = append(out, profiles.UserSummary{UserID: id, FirstName: "Unknown"})
		}
	}
	return out, nil
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) (string, error) {
	targetID := strings.ToLower(strings.TrimSpace(req.TargetUserID))
	if err := s.target(ctx, userID, targetID); err != nil {
		return "", err
	}
	reason, ok := normalizeReason(req.Reason)
	if !ok {
		return "", platformerrors.Validation("unknown report reason")
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return "", platformerrors.Validation("details must be at most 1000 characters")
	}
	return s.repo.CreateReport(ctx, userID, targetID, reason, details)
}
