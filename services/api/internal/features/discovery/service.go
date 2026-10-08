package discovery

import (
	"context"
	"errors"
	"strings"
	"time"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/pagination"
	"example.com/api/internal/platform/realtime"
)

const (
	DefaultBatch = 10
	MaxBatch     = 20
)

type CardLoader interface {
	Cards(ctx context.Context, viewerID string, ids []string) (map[string]profiles.PublicProfile, error)
}

type Service struct {
	repo      Repository
	cards     CardLoader
	publisher realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, publisher: publisher}
}

var errProfileRequired = platformerrors.Conflict("profile_incomplete", "complete your profile and add a photo first")

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	ok, err := s.repo.HasCompleteProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errProfileRequired
	}
	ids, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	return s.orderedCards(ctx, userID, ids)
}

func (s *Service) orderedCards(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error) {
	cards, err := s.cards.Cards(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]profiles.PublicProfile, 0, len(ids))
	for _, id := range ids {
		if c, ok := cards[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResult, error) {
	target := lower(req.UserID)
	if !platformerrors.IsUUID(target) {
		return SwipeResult{}, platformerrors.BadRequest("invalid_user", "invalid user id")
	}
	if target == userID {
		return SwipeResult{}, platformerrors.BadRequest("invalid_user", "you cannot swipe on yourself")
	}
	ok, err := s.repo.HasCompleteProfile(ctx, userID)
	if err != nil {
		return SwipeResult{}, err
	}
	if !ok {
		return SwipeResult{}, errProfileRequired
	}
	out, err := s.repo.Swipe(ctx, userID, target, req.Action)
	if err != nil {
		if errors.Is(err, ErrNotEligible) {
			return SwipeResult{}, platformerrors.NotFound("not_found", "profile not available")
		}
		return SwipeResult{}, err
	}
	res := SwipeResult{Action: req.Action, Matched: out.Matched, AlreadyDone: out.AlreadyDone, MatchID: out.MatchID, ConversationID: out.ConversationID}
	if out.Matched && !out.AlreadyDone {
		s.publisher.Publish(userID, realtime.Event{Type: "match.new", Data: map[string]string{"matchId": out.MatchID, "conversationId": out.ConversationID, "userId": target}})
		s.publisher.Publish(target, realtime.Event{Type: "match.new", Data: map[string]string{"matchId": out.MatchID, "conversationId": out.ConversationID, "userId": userID}})
	}
	return res, nil
}

func (s *Service) ListMatches(ctx context.Context, userID string, limit int, cursor string) (MatchesResponse, error) {
	cur, err := pagination.Decode(cursor)
	if err != nil {
		return MatchesResponse{}, err
	}
	var afterAt *time.Time
	afterID := ""
	if cur != nil {
		afterAt, afterID = &cur.At, cur.ID
	}
	rows, err := s.repo.ListMatches(ctx, userID, limit+1, afterAt, afterID)
	if err != nil {
		return MatchesResponse{}, err
	}
	res := MatchesResponse{Matches: []MatchItem{}}
	if len(rows) > limit {
		last := rows[limit-1]
		res.NextCursor = pagination.Cursor{At: last.CreatedAt, ID: last.ID}.Encode()
		rows = rows[:limit]
	}
	ids := make([]string, len(rows))
	for i, m := range rows {
		ids[i] = m.OtherID
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return MatchesResponse{}, err
	}
	for _, m := range rows {
		card, ok := cards[m.OtherID]
		if !ok {
			continue
		}
		res.Matches = append(res.Matches, MatchItem{ID: m.ID, ConversationID: m.ConversationID, CreatedAt: m.CreatedAt, HasMessages: m.HasMessages, User: card})
	}
	return res, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, ErrNoMatch) {
			return platformerrors.NotFound("not_found", "match not found")
		}
		return err
	}
	s.publisher.Publish(other, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	return nil
}

func lower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
