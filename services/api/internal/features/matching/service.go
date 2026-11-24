package matching

import (
	"context"
	"errors"
	"log"
	"time"

	"example.com/api/internal/features/profiles"
	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrProfileIncomplete = apperr.Conflict("profile_incomplete", "complete your profile (photo and location) to discover people")
	ErrSelfSwipe         = apperr.BadRequest("self_swipe", "you cannot swipe on yourself")
	ErrAlreadySwiped     = apperr.Conflict("already_swiped", "you already responded to this profile")
	ErrTargetUnavailable = apperr.NotFound("profile_unavailable", "this profile is no longer available")
	ErrMatchNotFound     = apperr.NotFound("match_not_found", "match not found")
)

type CardSource interface {
	Cards(ctx context.Context, viewerID string, targetIDs []string) ([]profiles.Card, error)
}

type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]any) error
}

type Service struct {
	repo        Repository
	cards       CardSource
	notifier    Notifier
	publisher   realtime.Publisher
	ranker      Ranker
	includeDemo bool
	now         func() time.Time
}

func NewService(repo Repository, cards CardSource, notifier Notifier, publisher realtime.Publisher, ranker Ranker, includeDemo bool) *Service {
	return &Service{repo: repo, cards: cards, notifier: notifier, publisher: publisher, ranker: ranker, includeDemo: includeDemo, now: time.Now}
}

// Discover returns the next ranked profiles for the viewer. Already swiped
// profiles are excluded server-side, so the client simply asks again for
// more: no offset, no duplicate profiles.
func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	viewer, err := s.repo.Viewer(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryIncomplete) {
			return nil, ErrProfileIncomplete
		}
		return nil, err
	}
	_ = s.repo.TouchUser(ctx, userID)
	pool, err := s.repo.CandidatePool(ctx, userID, s.includeDemo, candidatePoolSize)
	if err != nil {
		return nil, err
	}
	ranked := s.ranker.Rank(pool, viewer.MaxDistanceKm, s.now())
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	ids := make([]string, 0, len(ranked))
	for _, c := range ranked {
		ids = append(ids, c.UserID)
	}
	return s.cards.Cards(ctx, userID, ids)
}

func (s *Service) Swipe(ctx context.Context, userID, targetID, decision string) (SwipeResponse, error) {
	if userID == targetID {
		return SwipeResponse{}, ErrSelfSwipe
	}
	if decision != DecisionLike && decision != DecisionPass {
		return SwipeResponse{}, apperr.BadRequest("invalid_decision", "decision must be like or pass")
	}
	if _, err := s.repo.Viewer(ctx, userID); err != nil {
		if errors.Is(err, ErrRepositoryIncomplete) {
			return SwipeResponse{}, ErrProfileIncomplete
		}
		return SwipeResponse{}, err
	}
	ok, err := s.repo.CanSwipe(ctx, userID, targetID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !ok {
		return SwipeResponse{}, ErrTargetUnavailable
	}
	match, err := s.repo.Swipe(ctx, userID, targetID, decision)
	if err != nil {
		if errors.Is(err, ErrRepositoryAlreadySwiped) {
			return SwipeResponse{}, ErrAlreadySwiped
		}
		return SwipeResponse{}, err
	}
	resp := SwipeResponse{Decision: decision}
	if match == nil {
		return resp, nil
	}

	resp.Matched = true
	cards, err := s.cards.Cards(ctx, userID, []string{targetID, userID})
	if err != nil {
		return SwipeResponse{}, err
	}
	byID := map[string]profiles.Card{}
	for _, c := range cards {
		byID[c.UserID] = c
	}
	resp.Match = &MatchResponse{Match: *match, User: byID[targetID]}
	s.announceMatch(ctx, userID, targetID, *match, byID)
	return resp, nil
}

func (s *Service) announceMatch(ctx context.Context, userID, targetID string, match Match, byID map[string]profiles.Card) {
	for _, pair := range [][2]string{{userID, targetID}, {targetID, userID}} {
		recipient, other := pair[0], pair[1]
		otherCard := byID[other]
		s.publisher.Publish(recipient, realtime.Event{Type: "match.created", Data: MatchResponse{Match: match, User: otherCard}})
		name := otherCard.FirstName
		if name == "" {
			name = "Quelqu'un"
		}
		if err := s.notifier.Notify(ctx, recipient, "match", "Nouveau match ✨", name+" et toi vous êtes plu. Lance la conversation !", map[string]any{
			"matchId":        match.ID,
			"conversationId": match.ConversationID,
			"userId":         other,
		}); err != nil {
			log.Printf(`{"event":"match_notification_failed","error":%q}`, err.Error())
		}
	}
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrMatchNotFound
		}
		return err
	}
	s.publisher.Publish(other, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	s.publisher.Publish(userID, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	return nil
}
