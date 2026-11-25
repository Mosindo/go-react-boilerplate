package matching

import (
	"context"
	"errors"
	"log"
	"net/http"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrProfileIncomplete = apperr.New(http.StatusConflict, "profile_incomplete", "complete your profile (photo, location, preferences) to discover people")
	ErrSelfSwipe         = apperr.Validation("self_swipe", "you cannot swipe on yourself")
	ErrTargetUnavailable = apperr.New(http.StatusNotFound, "profile_unavailable", "this profile is no longer available")
	ErrAlreadySwiped     = apperr.New(http.StatusConflict, "already_swiped", "you already answered this profile")
	ErrMatchNotFound     = apperr.New(http.StatusNotFound, "match_not_found", "match not found")
)

type ProfileService interface {
	IsComplete(ctx context.Context, userID string) (bool, error)
	Hydrate(ctx context.Context, viewerID string, rows []profiles.CardRow) ([]profiles.PublicProfile, error)
	Public(ctx context.Context, viewerID, targetID string) (profiles.PublicProfile, error)
}

type Notifier interface {
	Notify(ctx context.Context, userID, typ, title, body string, data map[string]string)
}

type Service struct {
	repo        Repository
	profiles    ProfileService
	notifier    Notifier
	pub         realtime.Publisher
	includeDemo bool
}

// NewService wires the matching service. includeDemo controls whether seeded
// demo accounts appear in discovery (never in production).
func NewService(repo Repository, p ProfileService, n Notifier, pub realtime.Publisher, includeDemo bool) *Service {
	return &Service{repo: repo, profiles: p, notifier: n, pub: pub, includeDemo: includeDemo}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	complete, err := s.profiles.IsComplete(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !complete {
		return nil, ErrProfileIncomplete
	}
	if err := s.repo.TouchActivity(ctx, userID); err != nil {
		log.Printf(`{"event":"touch_activity_failed","error":%q}`, err.Error())
	}
	rows, err := s.repo.Discover(ctx, userID, limit, s.includeDemo)
	if err != nil {
		return nil, err
	}
	return s.profiles.Hydrate(ctx, userID, rows)
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResponse, error) {
	if req.TargetUserID == userID {
		return SwipeResponse{}, ErrSelfSwipe
	}
	if req.Action != ActionLike && req.Action != ActionPass {
		return SwipeResponse{}, apperr.ErrInvalidInput
	}
	complete, err := s.profiles.IsComplete(ctx, userID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !complete {
		return SwipeResponse{}, ErrProfileIncomplete
	}

	res, err := s.repo.Swipe(ctx, userID, req.TargetUserID, req.Action)
	switch {
	case errors.Is(err, errRepoTargetUnavailable):
		return SwipeResponse{}, ErrTargetUnavailable
	case errors.Is(err, errRepoConflict):
		return SwipeResponse{}, ErrAlreadySwiped
	case err != nil:
		return SwipeResponse{}, err
	}

	resp := SwipeResponse{Action: res.action, AlreadySwiped: res.alreadySwiped}
	if res.matchID == "" {
		return resp, nil
	}

	targetProfile, err := s.profiles.Public(ctx, userID, req.TargetUserID)
	if err != nil {
		return SwipeResponse{}, err
	}
	resp.Match = &MatchInfo{MatchID: res.matchID, ConversationID: res.conversationID, Profile: targetProfile}

	if !res.alreadySwiped {
		s.announceMatch(ctx, userID, req.TargetUserID, res)
	}
	return resp, nil
}

func (s *Service) announceMatch(ctx context.Context, userID, targetID string, res swipeResult) {
	data := func(otherID string) map[string]string {
		return map[string]string{"matchId": res.matchID, "conversationId": res.conversationID, "userId": otherID}
	}
	s.notifier.Notify(ctx, userID, notifications.TypeMatch, "Nouveau match !",
		"Vous et "+res.targetName+" vous êtes plu mutuellement.", data(targetID))
	s.notifier.Notify(ctx, targetID, notifications.TypeMatch, "Nouveau match !",
		"Vous et "+res.swiperName+" vous êtes plu mutuellement.", data(userID))
	s.pub.Publish(userID, realtime.Event{Type: "match.created", Data: data(targetID)})
	s.pub.Publish(targetID, realtime.Event{Type: "match.created", Data: data(userID)})
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.EndMatch(ctx, userID, matchID)
	if errors.Is(err, errRepoNotFound) {
		return ErrMatchNotFound
	}
	if err != nil {
		return err
	}
	s.pub.Publish(other, realtime.Event{Type: "match.ended", Data: map[string]string{"matchId": matchID}})
	s.pub.Publish(userID, realtime.Event{Type: "match.ended", Data: map[string]string{"matchId": matchID}})
	return nil
}
