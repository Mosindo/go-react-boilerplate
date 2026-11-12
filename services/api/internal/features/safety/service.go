package safety

import (
	"context"
	"log"
	"slices"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

type ProfileProvider interface {
	PublicProfiles(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

type Service struct {
	repo      Repository
	profiles  ProfileProvider
	publisher realtime.Publisher
}

func NewService(repo Repository, profiles ProfileProvider, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, publisher: publisher}
}

func (s *Service) checkTarget(ctx context.Context, actorID, organizationID, targetID string) error {
	if actorID == targetID {
		return ErrSelfAction
	}
	ok, err := s.repo.UserInOrganization(ctx, organizationID, targetID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUserNotFound
	}
	return nil
}

func (s *Service) Block(ctx context.Context, blockerID, organizationID, blockedID string) error {
	if err := s.checkTarget(ctx, blockerID, organizationID, blockedID); err != nil {
		return err
	}
	matchID, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if matchID != "" {
		ev := realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}}
		s.publisher.Publish(blockerID, ev)
		s.publisher.Publish(blockedID, ev)
	}
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
	cards, err := s.profiles.PublicProfiles(ctx, blockerID, ids)
	if err != nil {
		return nil, err
	}
	byID := map[string]profiles.PublicProfile{}
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]BlockedUser, 0, len(stored))
	for _, b := range stored {
		item := BlockedUser{UserID: b.UserID, FirstName: "Membre", BlockedAt: b.BlockedAt}
		if c, ok := byID[b.UserID]; ok {
			item.FirstName = c.FirstName
			if len(c.Photos) > 0 {
				item.PhotoURL = c.Photos[0].URL
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// Report files a moderation report and always blocks the reported member:
// someone who felt the need to report should never see that person again.
func (s *Service) Report(ctx context.Context, reporterID, organizationID, reportedID, reason, details string) error {
	reason = strings.TrimSpace(reason)
	if !slices.Contains(ReportReasons, reason) {
		return ErrInvalidReason
	}
	details = strings.TrimSpace(details)
	if utf8.RuneCountInString(details) > 1000 {
		return ErrDetailsTooLong
	}
	if err := s.checkTarget(ctx, reporterID, organizationID, reportedID); err != nil {
		return err
	}
	if err := s.repo.CreateReport(ctx, reporterID, reportedID, reason, details); err != nil {
		return err
	}
	log.Printf(`{"event":"moderation_report","reason":%q,"reported_user_id":%q}`, reason, reportedID)
	return s.Block(ctx, reporterID, organizationID, reportedID)
}
