package matching

import (
	"context"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]*profiles.PublicProfile, error) {
	return s.repo.Discover(ctx, userID, limit)
}

func (s *Service) LikesReceived(ctx context.Context, userID string, limit, offset int) ([]*profiles.PublicProfile, error) {
	return s.repo.LikesReceived(ctx, userID, limit, offset)
}

func (s *Service) Swipe(ctx context.Context, userID, targetID string, action Action) (SwipeResponse, error) {
	res, err := s.repo.Swipe(ctx, userID, targetID, action)
	if err != nil {
		return SwipeResponse{}, err
	}
	// Push after commit so clients never see events for rolled-back data.
	for _, mu := range res.Created {
		s.pub.Publish(mu.UserID, realtime.Event{Type: "match.new", Data: map[string]any{
			"matchId":        res.Match.ID,
			"conversationId": res.Match.ConversationID,
			"user":           mu.Other,
		}})
		s.pub.Publish(mu.UserID, realtime.Event{Type: "notification.new", Data: mu.Notification})
	}
	return res.SwipeResponse, nil
}

func (s *Service) ListMatches(ctx context.Context, userID string, limit, offset int) ([]MatchItem, error) {
	return s.repo.ListMatches(ctx, userID, limit, offset)
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	rm, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	for _, id := range rm.UserIDs {
		s.pub.Publish(id, realtime.Event{Type: "match.removed", Data: map[string]string{
			"matchId": rm.MatchID, "conversationId": rm.ConversationID,
		}})
	}
	return nil
}
