package discovery

import (
	"context"
	"errors"
	"net/http"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo     Repository
	profiles *profiles.Service
	notifier notifications.Notifier
	pub      realtime.Publisher
}

func NewService(repo Repository, p *profiles.Service, n notifications.Notifier, pub realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: p, notifier: n, pub: pub}
}

// Deck returns the next profiles to show. Profiles already judged never come back.
func (s *Service) Deck(ctx context.Context, userID string, limit int) (DeckResponse, error) {
	ids, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		return DeckResponse{}, err
	}
	views, err := s.profiles.Views(ctx, userID, ids)
	return DeckResponse{Profiles: views}, err
}

// LikesReceived lists people who liked the user and are still pending.
func (s *Service) LikesReceived(ctx context.Context, userID string, limit int) (DeckResponse, error) {
	ids, err := s.repo.LikesReceived(ctx, userID, limit)
	if err != nil {
		return DeckResponse{}, err
	}
	views, err := s.profiles.Views(ctx, userID, ids)
	return DeckResponse{Profiles: views}, err
}

func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResponse, error) {
	if action != ActionLike && action != ActionPass {
		return SwipeResponse{}, platformerrors.New(http.StatusBadRequest, "validation", "action must be like or pass")
	}
	if userID == targetID {
		return SwipeResponse{}, platformerrors.New(http.StatusBadRequest, "validation", "you cannot swipe yourself")
	}
	out, err := s.repo.Swipe(ctx, userID, targetID, action)
	switch {
	case errors.Is(err, ErrNotEligible):
		return SwipeResponse{}, platformerrors.New(http.StatusNotFound, "not_found", "profile not available")
	case errors.Is(err, ErrProfileIncomplete):
		return SwipeResponse{}, platformerrors.New(http.StatusForbidden, "profile_incomplete", "complete your profile (details, preferences, one photo) first")
	case err != nil:
		return SwipeResponse{}, err
	}

	resp := SwipeResponse{Result: "liked", AlreadyProcessed: !out.Inserted}
	if action == ActionPass {
		resp.Result = "passed"
	}
	if out.Matched {
		resp.Result = "matched"
		info := &MatchInfo{MatchID: out.MatchID, ConversationID: out.ConversationID}
		if views, err := s.profiles.Views(ctx, userID, []string{targetID}); err == nil && len(views) > 0 {
			info.Profile = &views[0]
		}
		resp.Match = info
	}

	if out.Inserted {
		switch {
		case out.Matched:
			s.announceMatch(ctx, userID, targetID, out)
		case action == ActionLike:
			s.notifier.NotifyOnce(ctx, targetID, notifications.TypeLike, "Quelqu'un vous a liké",
				"Découvrez qui s'intéresse à vous.", map[string]string{"kind": "like"}, "kind")
		}
	}
	return resp, nil
}

// announceMatch tells both users, each with the other's first name.
func (s *Service) announceMatch(ctx context.Context, a, b string, out swipeOutcome) {
	for _, pair := range [][2]string{{a, b}, {b, a}} {
		recipient, other := pair[0], pair[1]
		name := "Quelqu'un"
		views, err := s.profiles.Views(ctx, recipient, []string{other})
		if err == nil && len(views) > 0 {
			name = views[0].FirstName
		}
		data := map[string]string{"matchId": out.MatchID, "conversationId": out.ConversationID, "userId": other}
		s.notifier.Notify(ctx, recipient, notifications.TypeMatch, "C'est un match !",
			"Vous et "+name+" vous plaisez mutuellement. Dites bonjour !", data)
		s.pub.Publish(recipient, realtime.Event{Type: "match.new", Data: data})
	}
}
