package discovery

import (
	"context"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

// ProfileHydrator loads full profiles in the given order.
type ProfileHydrator interface {
	Hydrate(ctx context.Context, viewerID string, ids []string) ([]profiles.Profile, error)
}

// Notifier records and pushes a notification without failing the caller.
type Notifier interface {
	NotifyBestEffort(ctx context.Context, userID, kind, actorID string, data map[string]string)
}

type Service struct {
	repo     Repository
	profiles ProfileHydrator
	notifier Notifier
	events   realtime.Publisher
}

func NewService(repo Repository, hydrator ProfileHydrator, notifier Notifier, events realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: hydrator, notifier: notifier, events: events}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.Profile, error) {
	ready, err := s.repo.HasDiscoverableSetup(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !ready {
		return nil, platformerrors.Conflict("complete your profile (details, photo and location) to discover people")
	}
	ids, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	return s.profiles.Hydrate(ctx, userID, ids)
}

func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error) {
	if !profiles.IsUUID(targetID) {
		return SwipeResult{}, platformerrors.NotFound("profile not found")
	}
	if targetID == userID {
		return SwipeResult{}, platformerrors.BadRequest("you cannot react to yourself")
	}
	created, err := s.repo.Swipe(ctx, userID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	if created == nil {
		return SwipeResult{}, nil
	}

	mine, err := s.repo.GetMatchItem(ctx, created.MatchID, userID)
	if err != nil {
		return SwipeResult{}, err
	}
	theirs, err := s.repo.GetMatchItem(ctx, created.MatchID, targetID)
	if err != nil {
		return SwipeResult{}, err
	}
	s.events.Publish(userID, realtime.Event{"type": "match.new", "match": mine})
	s.events.Publish(targetID, realtime.Event{"type": "match.new", "match": theirs})
	s.notifier.NotifyBestEffort(ctx, userID, "match", targetID, map[string]string{"matchId": created.MatchID, "conversationId": created.ConversationID})
	s.notifier.NotifyBestEffort(ctx, targetID, "match", userID, map[string]string{"matchId": created.MatchID, "conversationId": created.ConversationID})
	return SwipeResult{Matched: true, Match: &mine}, nil
}

func (s *Service) ListMatches(ctx context.Context, userID string, cursor *httpx.TimeCursor, limit int) (MatchesResult, error) {
	items, err := s.repo.ListMatches(ctx, userID, cursor, limit+1)
	if err != nil {
		return MatchesResult{}, err
	}
	res := MatchesResult{}
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		next := httpx.EncodeTimeCursor(last.CreatedAt, last.ID)
		res.NextCursor = &next
	}
	res.Matches = items
	return res, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	if !profiles.IsUUID(matchID) {
		return platformerrors.NotFound("match not found")
	}
	other, conversationID, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	removed := realtime.Event{"type": "match.removed", "matchId": matchID, "conversationId": conversationID}
	s.events.Publish(other, removed)
	s.events.Publish(userID, removed)
	return nil
}
