package matching

import (
	"context"
	"errors"
	"strings"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

const (
	DefaultLimit = 30
	MaxLimit     = 100
)

var ErrProfileIncomplete = errors.New("complete your profile before swiping")

type CardLoader interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.Card, error)
}

type ViewerChecker interface {
	Me(ctx context.Context, userID string) (profiles.MeResponse, error)
}

type MatchNotifier interface {
	NotifyMatch(ctx context.Context, userID, actorID, matchID string)
}

type Service struct {
	repo      Repository
	cards     CardLoader
	viewer    ViewerChecker
	notifier  MatchNotifier
	publisher realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, viewer ViewerChecker, notifier MatchNotifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, viewer: viewer, notifier: notifier, publisher: publisher}
}

func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResponse, error) {
	targetID = strings.ToLower(targetID)
	if userID == targetID {
		return SwipeResponse{}, ErrSelfSwipe
	}
	me, err := s.viewer.Me(ctx, userID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !me.Completeness.IsComplete {
		return SwipeResponse{}, ErrProfileIncomplete
	}
	ok, err := s.repo.TargetSwipeable(ctx, userID, targetID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !ok {
		return SwipeResponse{}, ErrTargetUnavailable
	}
	outcome, err := s.repo.RecordSwipe(ctx, userID, targetID, action)
	if err != nil {
		return SwipeResponse{}, err
	}
	resp := SwipeResponse{Action: outcome.Action}
	if outcome.MatchID == "" {
		return resp, nil
	}
	match, err := s.Get(ctx, userID, outcome.MatchID)
	if err != nil {
		return SwipeResponse{}, err
	}
	resp.Matched = true
	resp.Match = &match

	if outcome.MatchCreated {
		s.notifier.NotifyMatch(ctx, targetID, userID, outcome.MatchID)
		s.notifier.NotifyMatch(ctx, userID, targetID, outcome.MatchID)
		if other, err := s.Get(ctx, targetID, outcome.MatchID); err == nil {
			s.publisher.Publish(targetID, realtime.Event{Type: "match.created", Data: other})
		}
	}
	return resp, nil
}

func (s *Service) Get(ctx context.Context, userID, matchID string) (Match, error) {
	stored, err := s.repo.GetMatch(ctx, userID, matchID)
	if err != nil {
		return Match{}, err
	}
	matches, err := s.hydrate(ctx, userID, []storedMatch{stored})
	if err != nil {
		return Match{}, err
	}
	if len(matches) == 0 {
		return Match{}, ErrMatchNotFound
	}
	return matches[0], nil
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (ListResponse, error) {
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	stored, err := s.repo.ListMatches(ctx, userID, limit+1, offset)
	if err != nil {
		return ListResponse{}, err
	}
	hasMore := len(stored) > limit
	if hasMore {
		stored = stored[:limit]
	}
	matches, err := s.hydrate(ctx, userID, stored)
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Matches: matches, HasMore: hasMore}, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.publisher.Publish(other, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	s.publisher.Publish(userID, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	return nil
}

// hydrate attaches the other user's card to each match with one batched load.
func (s *Service) hydrate(ctx context.Context, userID string, stored []storedMatch) ([]Match, error) {
	ids := make([]string, len(stored))
	for i, m := range stored {
		ids[i] = m.OtherUserID
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]Match, 0, len(stored))
	for _, m := range stored {
		card, ok := byID[m.OtherUserID]
		if !ok {
			continue
		}
		out = append(out, Match{
			ID:            m.ID,
			CreatedAt:     m.CreatedAt,
			LastMessageAt: m.LastMessageAt,
			User:          card,
			LastMessage:   m.LastMessage,
			UnreadCount:   m.UnreadCount,
		})
	}
	return out, nil
}
