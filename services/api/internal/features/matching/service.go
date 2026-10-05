package matching

import (
	"context"
	"errors"
	"strings"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var ErrProfileIncomplete = errors.New("complete your profile to start discovering")

// ProfileReader is the part of the profiles feature this feature depends on.
type ProfileReader interface {
	IsComplete(ctx context.Context, userID string) (bool, error)
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Card, error)
}

type Service struct {
	repo     Repository
	profiles ProfileReader
	events   realtime.Publisher
}

func NewService(repo Repository, profiles ProfileReader, events realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, events: events}
}

func (s *Service) requireComplete(ctx context.Context, userID string) error {
	ok, err := s.profiles.IsComplete(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrProfileIncomplete
	}
	return nil
}

func normalizeLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	return min(limit, max)
}

// Discover returns the next batch of profiles. Profiles already liked or passed never come back.
func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	if err := s.requireComplete(ctx, userID); err != nil {
		return nil, err
	}
	ids, err := s.repo.Discover(ctx, userID, normalizeLimit(limit, defaultDiscoverLimit, maxDiscoverLimit))
	if err != nil {
		return nil, err
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	_ = s.repo.TouchActive(ctx, userID)

	out := make([]profiles.Card, 0, len(ids))
	for _, id := range ids {
		if c, ok := cards[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

// Swipe applies a like or pass. When the like is reciprocal both users receive a match event.
func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResponse, error) {
	if action != ActionLike && action != ActionPass {
		return SwipeResponse{}, errors.New("invalid action")
	}
	targetID = strings.ToLower(strings.TrimSpace(targetID))
	if err := s.requireComplete(ctx, userID); err != nil {
		return SwipeResponse{}, err
	}
	match, err := s.repo.Swipe(ctx, userID, targetID, action)
	if err != nil {
		return SwipeResponse{}, err
	}
	resp := SwipeResponse{Action: action}
	if match == nil {
		return resp, nil
	}

	resp.Matched = true
	for _, pair := range [][2]string{{userID, targetID}, {targetID, userID}} {
		self, other := pair[0], pair[1]
		cards, err := s.profiles.Cards(ctx, self, []string{other})
		if err != nil {
			continue // the match is already stored; a failed push must not fail the request
		}
		preview := MatchPreview{ID: match.ID, ConversationID: match.ConversationID, CreatedAt: match.CreatedAt, User: cards[other]}
		if self == userID {
			resp.Match = &preview
		}
		// The swiper learns about the match from the HTTP response; only the other person needs a push.
		if self != userID {
			s.events.Publish(self, realtime.Event{Type: "match.new", Data: preview})
		}
		if n, ok := match.Notifications[self]; ok {
			s.events.Publish(self, realtime.Event{Type: "notification.new", Data: n})
		}
	}
	return resp, nil
}

func (s *Service) toPreviews(ctx context.Context, userID string, rows []matchRow) ([]MatchPreview, error) {
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.OtherUserID)
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]MatchPreview, 0, len(rows))
	for _, r := range rows {
		card, ok := cards[r.OtherUserID]
		if !ok {
			continue
		}
		out = append(out, MatchPreview{
			ID: r.ID, ConversationID: r.ConversationID, CreatedAt: r.CreatedAt,
			User: card, LastMessage: r.Last, UnreadCount: r.Unread,
		})
	}
	return out, nil
}

func (s *Service) ListMatches(ctx context.Context, userID string, limit, offset int) ([]MatchPreview, error) {
	rows, err := s.repo.ListMatches(ctx, userID, normalizeLimit(limit, defaultMatchesLimit, maxMatchesLimit), max(offset, 0))
	if err != nil {
		return nil, err
	}
	return s.toPreviews(ctx, userID, rows)
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.DeleteMatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	for _, id := range []string{userID, other} {
		s.events.Publish(id, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	}
	return nil
}
