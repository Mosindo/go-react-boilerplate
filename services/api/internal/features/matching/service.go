package matching

import (
	"context"
	"time"

	"example.com/api/internal/platform/pagination"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) (DiscoverResponse, error) {
	viewer, err := s.repo.LoadViewer(ctx, userID)
	if err != nil {
		return DiscoverResponse{}, err
	}
	list, err := s.repo.Discover(ctx, viewer, pagination.Limit(limit, DiscoverDefaultLimit, DiscoverMaxLimit))
	if err != nil {
		return DiscoverResponse{}, err
	}
	return DiscoverResponse{Profiles: list}, nil
}

// Swipe applies a like or pass. Repeating a swipe is harmless and never
// produces a second match.
func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error) {
	if action != ActionLike && action != ActionPass {
		return SwipeResult{}, ErrInvalidAction
	}
	if _, err := s.repo.LoadViewer(ctx, userID); err != nil {
		return SwipeResult{}, err
	}
	if userID == targetID {
		return SwipeResult{}, ErrTargetUnavailable
	}

	result, created, err := s.repo.Swipe(ctx, userID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	if created == nil {
		return result, nil
	}

	other, err := s.repo.GetCard(ctx, targetID)
	if err != nil {
		return SwipeResult{}, err
	}
	me, err := s.repo.GetCard(ctx, userID)
	if err != nil {
		return SwipeResult{}, err
	}
	result.Match = &Match{ID: created.MatchID, ConversationID: created.ConversationID, CreatedAt: created.CreatedAt, Profile: other}
	s.pub.Publish(targetID, realtime.Event{Type: "match.new", Data: Match{
		ID: created.MatchID, ConversationID: created.ConversationID, CreatedAt: created.CreatedAt, Profile: me,
	}})
	s.pub.Publish(userID, realtime.Event{Type: "notification.new"})
	s.pub.Publish(targetID, realtime.Event{Type: "notification.new"})
	return result, nil
}

func (s *Service) ListMatches(ctx context.Context, userID string, limit int, cursor string) (MatchesResponse, error) {
	limit = pagination.Limit(limit, ListDefaultLimit, ListMaxLimit)
	at, id, err := decode(cursor)
	if err != nil {
		return MatchesResponse{}, err
	}
	items, err := s.repo.ListMatches(ctx, userID, limit+1, at, id)
	if err != nil {
		return MatchesResponse{}, err
	}
	resp := MatchesResponse{}
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		resp.NextCursor = pagination.Encode(last.CreatedAt, last.ID)
	}
	resp.Matches = items
	return resp, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, conversationID, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.pub.Publish(other, realtime.Event{Type: "match.removed", Data: map[string]string{
		"matchId": matchID, "conversationId": conversationID,
	}})
	return nil
}

func (s *Service) ReceivedLikes(ctx context.Context, userID string, limit int, cursor string) (LikesResponse, error) {
	limit = pagination.Limit(limit, ListDefaultLimit, ListMaxLimit)
	at, id, err := decode(cursor)
	if err != nil {
		return LikesResponse{}, err
	}
	items, err := s.repo.ListReceivedLikes(ctx, userID, limit+1, at, id)
	if err != nil {
		return LikesResponse{}, err
	}
	resp := LikesResponse{}
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		resp.NextCursor = pagination.Encode(last.LikedAt, last.UserID)
	}
	resp.Profiles = items
	return resp, nil
}

func decode(cursor string) (*time.Time, string, error) {
	if cursor == "" {
		return nil, "", nil
	}
	at, id, err := pagination.Decode(cursor)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	return &at, id, nil
}
