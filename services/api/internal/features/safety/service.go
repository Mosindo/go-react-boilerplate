package safety

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

const maxDetailsRunes = 1000

type Service struct {
	repo     Repository
	profiles *profiles.Service
	events   realtime.Publisher
}

func NewService(repo Repository, profileService *profiles.Service, events realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profileService, events: events}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelfAction
	}
	if err := s.repo.Block(ctx, blockerID, blockedID); err != nil {
		return err
	}
	// The blocked member only sees the match disappear, never the block itself.
	s.events.Publish(blockedID, "match.removed", map[string]string{"userId": blockerID})
	s.events.Publish(blockerID, "match.removed", map[string]string{"userId": blockedID})
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	stored, err := s.repo.ListBlocks(ctx, blockerID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(stored))
	for _, b := range stored {
		ids = append(ids, b.UserID)
	}
	cards, err := s.profiles.Cards(ctx, blockerID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]BlockedUser, 0, len(stored))
	for _, b := range stored {
		item := BlockedUser{UserID: b.UserID, FirstName: "Membre", BlockedAt: b.BlockedAt}
		if c, ok := byID[b.UserID]; ok {
			item.FirstName = c.FirstName
			if len(c.Photos) > 0 {
				p := c.Photos[0]
				item.Photo = &p
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) (ReportResponse, error) {
	if reporterID == req.UserID {
		return ReportResponse{}, ErrSelfAction
	}
	reason := strings.TrimSpace(req.Reason)
	if !slices.Contains(ReportReasons, reason) {
		return ReportResponse{}, ErrInvalidReason
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > maxDetailsRunes {
		return ReportResponse{}, ErrDetailsTooLong
	}
	id, err := s.repo.Report(ctx, reporterID, req.UserID, reason, details)
	if err != nil {
		return ReportResponse{}, err
	}
	s.events.Publish(req.UserID, "match.removed", map[string]string{"userId": reporterID})
	s.events.Publish(reporterID, "match.removed", map[string]string{"userId": req.UserID})
	return ReportResponse{ID: id, Status: "open", Blocked: true}, nil
}
