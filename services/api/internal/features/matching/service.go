package matching

import (
	"context"
	"errors"
	"strings"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
	"github.com/google/uuid"
)

var (
	ErrInvalidKind = errors.New("kind must be like or pass")
	ErrInvalidID   = errors.New("invalid id")
)

// CardLoader assembles public cards (implemented by profiles.Service).
type CardLoader interface {
	Cards(ctx context.Context, viewerID string, ids []string) ([]profiles.Card, error)
}

type Service struct {
	repo   Repository
	cards  CardLoader
	events realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, events realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, events: events}
}

func (s *Service) Feed(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	_ = s.repo.Touch(ctx, userID) // best effort
	ids, err := s.repo.Feed(ctx, userID, clamp(limit))
	if err != nil {
		return nil, err
	}
	return s.cards.Cards(ctx, userID, ids)
}

func (s *Service) ReceivedLikes(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	ids, err := s.repo.ReceivedLikes(ctx, userID, clamp(limit))
	if err != nil {
		return nil, err
	}
	return s.cards.Cards(ctx, userID, ids)
}

// Swipe records a like or pass. Mutual likes become a match exactly once; a
// repeated swipe on the same person is a no-op that reports the first outcome.
func (s *Service) Swipe(ctx context.Context, userID, targetID, kind string) (SwipeResult, error) {
	if kind != KindLike && kind != KindPass {
		return SwipeResult{}, ErrInvalidKind
	}
	id, err := uuid.Parse(strings.TrimSpace(targetID))
	if err != nil {
		return SwipeResult{}, ErrInvalidID
	}
	targetID = id.String()

	out, err := s.repo.Swipe(ctx, userID, targetID, kind)
	if err != nil {
		return SwipeResult{}, err
	}
	res := SwipeResult{AlreadyHandled: out.AlreadyHandled}
	if out.Match == nil {
		return res, nil
	}
	match, err := s.toMatchResponse(ctx, userID, *out.Match)
	if err != nil {
		return SwipeResult{}, err
	}
	res.Matched, res.Match = true, &match
	if !out.AlreadyHandled {
		// Tell the other side in real time, with the card from *their* point of view.
		if theirs, err := s.toMatchResponse(ctx, targetID, StoredMatch{
			ID: out.Match.ID, ConversationID: out.Match.ConversationID, OtherUserID: userID, CreatedAt: out.Match.CreatedAt,
		}); err == nil {
			s.events.Publish(targetID, "match", theirs)
		}
	}
	return res, nil
}

func (s *Service) ListMatches(ctx context.Context, userID string, before *time.Time, limit int) (MatchesResponse, error) {
	limit = clamp(limit)
	stored, err := s.repo.ListMatches(ctx, userID, before, limit+1)
	if err != nil {
		return MatchesResponse{}, err
	}
	resp := MatchesResponse{Matches: []MatchResponse{}}
	if len(stored) > limit {
		stored = stored[:limit]
		next := stored[len(stored)-1].CreatedAt
		resp.NextBefore = &next
	}
	ids := make([]string, len(stored))
	for i, m := range stored {
		ids[i] = m.OtherUserID
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return MatchesResponse{}, err
	}
	byID := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byID[c.ID] = c
	}
	for _, m := range stored {
		if card, ok := byID[m.OtherUserID]; ok {
			resp.Matches = append(resp.Matches, MatchResponse{ID: m.ID, ConversationID: m.ConversationID, CreatedAt: m.CreatedAt, User: card})
		}
	}
	return resp, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.events.Publish(other, "unmatch", map[string]string{"matchId": matchID})
	return nil
}

func (s *Service) toMatchResponse(ctx context.Context, viewerID string, m StoredMatch) (MatchResponse, error) {
	cards, err := s.cards.Cards(ctx, viewerID, []string{m.OtherUserID})
	if err != nil {
		return MatchResponse{}, err
	}
	if len(cards) == 0 {
		return MatchResponse{}, ErrNotFound
	}
	return MatchResponse{ID: m.ID, ConversationID: m.ConversationID, CreatedAt: m.CreatedAt, User: cards[0]}, nil
}

func clamp(limit int) int {
	if limit <= 0 {
		return DefaultFeedSize
	}
	if limit > MaxFeedSize {
		return MaxFeedSize
	}
	return limit
}
