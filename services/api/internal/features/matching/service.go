package matching

import (
	"context"
	"strings"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

// Events publishes realtime signals; implemented by realtime.Hub.
type Events interface {
	Publish(userID string, event realtime.Event)
}

type Service struct {
	repo     Repository
	profiles *profiles.Service
	events   Events
}

func NewService(repo Repository, profileService *profiles.Service, events Events) *Service {
	return &Service{repo: repo, profiles: profileService, events: events}
}

// Discover returns the next batch of candidates. Profiles already swiped on
// never come back, so the client simply asks again once the batch is used;
// there is no offset to maintain.
func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	if limit <= 0 {
		limit = DefaultDiscoverLimit
	}
	if limit > MaxDiscoverLimit {
		limit = MaxDiscoverLimit
	}
	v, err := s.repo.Viewer(ctx, userID)
	if err != nil {
		return nil, err
	}
	stored, err := s.repo.Candidates(ctx, userID, v, "", limit)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, stored)
}

func (s *Service) hydrate(ctx context.Context, stored []profiles.StoredPublic) ([]profiles.PublicProfile, error) {
	ids := make([]string, 0, len(stored))
	for _, p := range stored {
		ids = append(ids, p.UserID)
	}
	photos, err := s.repo.PhotoIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.Interests(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]profiles.PublicProfile, 0, len(stored))
	for _, p := range stored {
		out = append(out, s.profiles.ToPublic(p, photos[p.UserID], interests[p.UserID]))
	}
	return out, nil
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResponse, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != ActionLike && action != ActionPass {
		return SwipeResponse{}, platformerrors.Invalid("action", "must be like or pass")
	}
	targetID := strings.ToLower(strings.TrimSpace(req.TargetUserID))
	if targetID == userID {
		return SwipeResponse{}, platformerrors.Invalid("targetUserId", "you cannot swipe on yourself")
	}

	v, err := s.repo.Viewer(ctx, userID)
	if err != nil {
		return SwipeResponse{}, err
	}
	res, err := s.repo.Swipe(ctx, userID, v, targetID, action)
	if err != nil {
		return SwipeResponse{}, err
	}

	resp := SwipeResponse{Action: res.Action, Duplicate: res.Duplicate, Matched: res.MatchID != ""}
	if resp.Matched {
		other, err := s.profiles.GetPublic(ctx, userID, targetID)
		if err != nil {
			return SwipeResponse{}, err
		}
		resp.Match = &MatchInfo{MatchID: res.MatchID, ConversationID: res.ConversationID, User: other}
	}
	if res.NewMatch {
		for _, id := range []string{userID, targetID} {
			s.events.Publish(id, realtime.Event{Type: realtime.TypeMatch, ConversationID: res.ConversationID})
			s.events.Publish(id, realtime.Event{Type: realtime.TypeNotification})
		}
	}
	return resp, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.events.Publish(other, realtime.Event{Type: realtime.TypeUnmatch})
	return nil
}
