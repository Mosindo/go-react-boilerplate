package discovery

import (
	"context"
	"time"

	"example.com/api/internal/features/matches"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

type ProfileReader interface {
	PublicProfiles(ctx context.Context, viewerID string, ids []string) (map[string]profiles.PublicProfile, error)
}

type MatchBuilder interface {
	BuildItem(ctx context.Context, viewerID, matchID, conversationID string, createdAt time.Time, otherID string) (matches.Item, error)
}

// NotificationPusher delivers a stored notification to the user's open sockets.
type NotificationPusher interface {
	Push(ctx context.Context, userID, notificationID string)
}

type Service struct {
	repo     *Repository
	profiles ProfileReader
	matches  MatchBuilder
	events   realtime.Publisher
	notifier NotificationPusher
}

func NewService(repo *Repository, profiles ProfileReader, matches MatchBuilder, events realtime.Publisher, notifier NotificationPusher) *Service {
	return &Service{repo: repo, profiles: profiles, matches: matches, events: events, notifier: notifier}
}

func (s *Service) Feed(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	s.repo.touch(ctx, userID)
	ids, err := s.repo.feed(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	byID, err := s.profiles.PublicProfiles(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]profiles.PublicProfile, 0, len(ids))
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Service) Swipe(ctx context.Context, userID, targetID string, action Action) (SwipeResponse, error) {
	if userID == targetID || (action != ActionLike && action != ActionPass) {
		return SwipeResponse{}, ErrTargetUnavailable
	}
	out, err := s.repo.swipe(ctx, userID, targetID, action)
	if err != nil {
		return SwipeResponse{}, err
	}
	resp := SwipeResponse{Result: out.Result}
	if out.Result != ResultMatched {
		return resp, nil
	}

	mine, err := s.matches.BuildItem(ctx, userID, out.MatchID, out.ConversationID, out.MatchCreatedAt, targetID)
	if err != nil {
		return SwipeResponse{}, err
	}
	resp.Match = &mine
	if theirs, err := s.matches.BuildItem(ctx, targetID, out.MatchID, out.ConversationID, out.MatchCreatedAt, userID); err == nil {
		s.events.Publish(targetID, realtime.Event{Type: "match", Data: theirs})
	}
	s.notifier.Push(ctx, targetID, out.TargetNotificationID)
	return resp, nil
}
