package matching

import (
	"context"
	"errors"
	"strings"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var ErrInvalidAction = errors.New("action must be 'like' or 'pass'")
var ErrSelfSwipe = errors.New("you cannot swipe yourself")

type Service struct {
	repo          Repository
	profiles      *profiles.Service
	notifications *notifications.Service
	publisher     realtime.Publisher
}

func NewService(repo Repository, profilesService *profiles.Service, notificationsService *notifications.Service, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.Nop{}
	}
	return &Service{repo: repo, profiles: profilesService, notifications: notificationsService, publisher: publisher}
}

func NormalizeAction(action string) (string, error) {
	a := strings.ToLower(strings.TrimSpace(action))
	if a != ActionLike && a != ActionPass {
		return "", ErrInvalidAction
	}
	return a, nil
}

func (s *Service) Swipe(ctx context.Context, actorID, targetID, rawAction string) (SwipeResponse, error) {
	action, err := NormalizeAction(rawAction)
	if err != nil {
		return SwipeResponse{}, err
	}
	if actorID == targetID {
		return SwipeResponse{}, ErrSelfSwipe
	}
	result, err := s.repo.Swipe(ctx, actorID, targetID, action)
	if err != nil {
		return SwipeResponse{}, err
	}
	resp := SwipeResponse{Action: action, Matched: result.Matched}
	if !result.Matched || result.Match == nil {
		return resp, nil
	}

	summaries, err := s.profiles.Summaries(ctx, actorID, []string{actorID, targetID})
	if err != nil {
		return SwipeResponse{}, err
	}
	actor, target := summaries[actorID], summaries[targetID]
	resp.Match = &MatchResponse{ID: result.Match.ID, User: target, CreatedAt: result.Match.CreatedAt}

	// Both members are told in realtime and get a persistent notification.
	s.publisher.Publish(actorID, realtime.Event{Type: "match.new", Data: MatchResponse{ID: result.Match.ID, User: target, CreatedAt: result.Match.CreatedAt}})
	s.publisher.Publish(targetID, realtime.Event{Type: "match.new", Data: MatchResponse{ID: result.Match.ID, User: actor, CreatedAt: result.Match.CreatedAt}})
	s.notifications.Notify(ctx, actorID, notifications.TypeMatch, "Nouveau match ✨", "Vous et "+target.FirstName+" vous plaisez mutuellement.", map[string]string{"userId": targetID, "matchId": result.Match.ID})
	s.notifications.Notify(ctx, targetID, notifications.TypeMatch, "Nouveau match ✨", "Vous et "+actor.FirstName+" vous plaisez mutuellement.", map[string]string{"userId": actorID, "matchId": result.Match.ID})
	return resp, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, otherID string) error {
	matchID, err := s.repo.Unmatch(ctx, userID, otherID)
	if err != nil {
		return err
	}
	s.publisher.Publish(otherID, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": userID, "matchId": matchID}})
	s.publisher.Publish(userID, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": otherID, "matchId": matchID}})
	return nil
}
