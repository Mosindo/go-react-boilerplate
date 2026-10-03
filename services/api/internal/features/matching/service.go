package matching

import (
	"context"
	"fmt"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

// ProfileReader loads public profiles for display.
type ProfileReader interface {
	PublicProfiles(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

// Notifier stores an in-app notification for a user.
type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error
}

type Service struct {
	repo     Repository
	profiles ProfileReader
	notifier Notifier
	events   realtime.Publisher
}

func NewService(repo Repository, profiles ProfileReader, notifier Notifier, events realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, notifier: notifier, events: events}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	switch {
	case limit <= 0:
		limit = DefaultDiscoverLimit
	case limit > MaxDiscoverLimit:
		limit = MaxDiscoverLimit
	}
	ids, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	return s.profiles.PublicProfiles(ctx, userID, ids)
}

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResponse, error) {
	if action != ActionLike && action != ActionPass {
		return SwipeResponse{}, &ValidationError{Message: "action must be like or pass"}
	}
	if userID == targetID {
		return SwipeResponse{}, &ValidationError{Message: "you cannot swipe on yourself"}
	}
	out, err := s.repo.Swipe(ctx, userID, targetID, action)
	if err != nil {
		return SwipeResponse{}, err
	}

	resp := SwipeResponse{Matched: out.Matched, MatchID: out.MatchID}
	if !out.Matched {
		return resp, nil
	}
	people, err := s.profiles.PublicProfiles(ctx, userID, []string{targetID})
	if err == nil && len(people) == 1 {
		resp.Profile = &people[0]
	}
	if out.NewMatch {
		s.announceMatch(ctx, userID, targetID, out.MatchID)
	}
	return resp, nil
}

// announceMatch tells both users, in-app and in real time. Failures are non-fatal: the match
// itself is already committed and will appear in both conversation lists.
func (s *Service) announceMatch(ctx context.Context, userA, userB, matchID string) {
	for _, pair := range [][2]string{{userA, userB}, {userB, userA}} {
		me, other := pair[0], pair[1]
		name := "someone"
		if people, err := s.profiles.PublicProfiles(ctx, me, []string{other}); err == nil && len(people) == 1 {
			name = people[0].FirstName
		}
		_ = s.notifier.Notify(ctx, me, "match", "It's a match!",
			fmt.Sprintf("You and %s liked each other. Say hello!", name),
			map[string]string{"matchId": matchID, "userId": other, "name": name})
		s.events.Publish(me, realtime.Event{Type: "match.new", Data: map[string]string{"matchId": matchID, "userId": other}})
	}
}
