package matching

import (
	"context"
	"strings"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/apperr"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrProfileIncomplete = apperr.Conflict("profile_incomplete", "complete your profile (name, birthdate, gender, one photo) first")
	ErrTargetUnavailable = apperr.NotFound("profile_unavailable", "this profile is no longer available")
	ErrAlreadySwiped     = apperr.Conflict("already_swiped", "you already responded to this profile")
	ErrSelfSwipe         = apperr.BadRequest("self_swipe", "you cannot swipe on yourself")
	ErrMatchNotFound     = apperr.NotFound("match_not_found", "match not found")
)

type Service struct {
	repo          Repository
	cards         *profiles.Service
	ranker        Ranker
	notifications *notifications.Service
	publisher     realtime.Publisher
	now           func() time.Time
}

func NewService(repo Repository, cards *profiles.Service, ranker Ranker, notifier *notifications.Service, publisher realtime.Publisher) *Service {
	if ranker == nil {
		ranker = ScoreRanker{}
	}
	return &Service{repo: repo, cards: cards, ranker: ranker, notifications: notifier, publisher: publisher, now: time.Now}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) (DiscoveryResponse, error) {
	complete, maxDistance, err := s.repo.ViewerStatus(ctx, userID)
	if err != nil {
		return DiscoveryResponse{}, err
	}
	if !complete {
		return DiscoveryResponse{}, ErrProfileIncomplete
	}
	candidates, err := s.repo.Candidates(ctx, userID, candidatePoolSize)
	if err != nil {
		return DiscoveryResponse{}, err
	}
	ranked := s.ranker.Rank(ViewerContext{UserID: userID, MaxDistanceKm: maxDistance, Now: s.now()}, candidates)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	ids := make([]string, len(ranked))
	for i, c := range ranked {
		ids[i] = c.UserID
	}
	cards, err := s.cards.LoadCards(ctx, userID, ids)
	if err != nil {
		return DiscoveryResponse{}, err
	}
	out := make([]profiles.Card, 0, len(ids))
	for _, id := range ids {
		if card, ok := cards[id]; ok {
			out = append(out, card)
		}
	}
	return DiscoveryResponse{Profiles: out}, nil
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResult, error) {
	targetID := strings.ToLower(req.TargetUserID)
	if targetID == userID {
		return SwipeResult{}, ErrSelfSwipe
	}
	complete, _, err := s.repo.ViewerStatus(ctx, userID)
	if err != nil {
		return SwipeResult{}, err
	}
	if !complete {
		return SwipeResult{}, ErrProfileIncomplete
	}
	match, err := s.repo.Swipe(ctx, userID, targetID, req.Action)
	if err != nil {
		return SwipeResult{}, err
	}

	if match == nil {
		if req.Action == ActionLike {
			s.notifications.Notify(ctx, notifications.Input{
				UserID:   targetID,
				Type:     notifications.TypeLike,
				Title:    "Quelqu'un vous a remarqué",
				Body:     "Vous avez reçu de nouveaux likes. Découvrez qui !",
				Data:     map[string]string{},
				GroupKey: "likes",
			})
		}
		return SwipeResult{Matched: false}, nil
	}

	cards, err := s.cards.LoadCards(ctx, userID, []string{targetID})
	if err != nil {
		return SwipeResult{}, err
	}
	otherCards, err := s.cards.LoadCards(ctx, targetID, []string{userID})
	if err != nil {
		return SwipeResult{}, err
	}
	mine := MatchResponse{ID: match.ID, ConversationID: match.ConversationID, CreatedAt: match.CreatedAt, User: cards[targetID]}
	theirs := MatchResponse{ID: match.ID, ConversationID: match.ConversationID, CreatedAt: match.CreatedAt, User: otherCards[userID]}

	for _, side := range []struct {
		userID string
		match  MatchResponse
	}{{userID, mine}, {targetID, theirs}} {
		s.notifications.Notify(ctx, notifications.Input{
			UserID: side.userID,
			Type:   notifications.TypeMatch,
			Title:  "C'est un match !",
			Body:   "Vous et " + side.match.User.FirstName + " vous plaisez mutuellement.",
			Data:   map[string]string{"matchId": match.ID, "conversationId": match.ConversationID, "userId": side.match.User.UserID},
		})
		if s.publisher != nil {
			s.publisher.Publish(side.userID, realtime.Event{Type: "match.new", Data: side.match})
		}
	}
	return SwipeResult{Matched: true, Match: &mine}, nil
}

func (s *Service) ReceivedLikes(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) (LikesResponse, error) {
	likes, err := s.repo.ReceivedLikes(ctx, userID, cursor, limit+1)
	if err != nil {
		return LikesResponse{}, err
	}
	var next *string
	if len(likes) > limit {
		likes = likes[:limit]
		last := likes[limit-1]
		encoded := httpx.EncodeCursor(last.LikedAt, last.UserID)
		next = &encoded
	}
	ids := make([]string, len(likes))
	for i, l := range likes {
		ids[i] = l.UserID
	}
	cards, err := s.cards.LoadCards(ctx, userID, ids)
	if err != nil {
		return LikesResponse{}, err
	}
	items := make([]LikeItem, 0, len(likes))
	for _, l := range likes {
		if card, ok := cards[l.UserID]; ok {
			items = append(items, LikeItem{LikedAt: l.LikedAt, User: card})
		}
	}
	return LikesResponse{Likes: items, NextCursor: next}, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	otherID, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	if s.publisher != nil {
		event := realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}}
		s.publisher.Publish(otherID, event)
		s.publisher.Publish(userID, event)
	}
	return nil
}
