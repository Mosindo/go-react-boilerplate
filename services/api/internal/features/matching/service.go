package matching

import (
	"context"
	"strings"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

type ProfileReader interface {
	RequireComplete(ctx context.Context, userID string) error
	PublicProfiles(ctx context.Context, viewerID string, userIDs []string) ([]profiles.PublicProfile, error)
}

type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string)
}

type Service struct {
	repo      Repository
	profiles  ProfileReader
	notifier  Notifier
	publisher realtime.Publisher
}

func NewService(repo Repository, profileReader ProfileReader, notifier Notifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profileReader, notifier: notifier, publisher: publisher}
}

func (s *Service) Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeResponse, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	if action != ActionLike && action != ActionPass {
		return SwipeResponse{}, ErrInvalidAction
	}
	if swiperID == targetID {
		return SwipeResponse{}, ErrCannotSwipeSelf
	}
	if err := s.profiles.RequireComplete(ctx, swiperID); err != nil {
		return SwipeResponse{}, err
	}
	ok, err := s.repo.TargetSwipeable(ctx, swiperID, targetID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !ok {
		return SwipeResponse{}, ErrTargetNotFound
	}

	outcome, err := s.repo.Swipe(ctx, swiperID, targetID, action)
	if err != nil {
		return SwipeResponse{}, err
	}
	res := SwipeResponse{Action: action, Matched: outcome.Matched}
	if !outcome.Matched {
		return res, nil
	}

	match := Match{ID: outcome.MatchID, ConversationID: outcome.ConversationID, CreatedAt: outcome.MatchedAt}
	swiperView, err := s.matchFor(ctx, swiperID, targetID, match)
	if err != nil {
		return SwipeResponse{}, err
	}
	swiperView.InitiatedByMe = true
	res.Match = swiperView
	s.announce(ctx, swiperID, *swiperView)
	if targetView, err := s.matchFor(ctx, targetID, swiperID, match); err == nil {
		s.announce(ctx, targetID, *targetView)
	}
	return res, nil
}

func (s *Service) matchFor(ctx context.Context, viewerID, otherID string, m Match) (*MatchWithProfile, error) {
	cards, err := s.profiles.PublicProfiles(ctx, viewerID, []string{otherID})
	if err != nil {
		return nil, err
	}
	if len(cards) == 0 {
		return nil, ErrTargetNotFound
	}
	m.OtherUserID = otherID
	return &MatchWithProfile{Match: m, Profile: cards[0]}, nil
}

func (s *Service) announce(ctx context.Context, userID string, m MatchWithProfile) {
	if s.publisher != nil {
		s.publisher.Publish(userID, "match.created", m)
	}
	if s.notifier != nil {
		s.notifier.Notify(ctx, userID, notifications.TypeMatch,
			"Nouveau match !",
			"Toi et "+m.Profile.FirstName+" vous plaisez mutuellement. Lance la conversation.",
			map[string]string{"matchId": m.ID, "conversationId": m.ConversationID, "userId": m.OtherUserID})
	}
}

func (s *Service) UndoLastPass(ctx context.Context, userID string) (UndoResponse, error) {
	targetID, err := s.repo.UndoLastPass(ctx, userID, undoWindow)
	if err != nil {
		return UndoResponse{}, err
	}
	cards, err := s.profiles.PublicProfiles(ctx, userID, []string{targetID})
	if err != nil {
		return UndoResponse{}, err
	}
	if len(cards) == 0 {
		return UndoResponse{}, ErrNothingToUndo
	}
	return UndoResponse{Profile: cards[0]}, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	otherID, conversationID, err := s.repo.EndMatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	payload := map[string]string{"matchId": matchID, "conversationId": conversationID}
	if s.publisher != nil {
		s.publisher.Publish(otherID, "match.ended", payload)
		s.publisher.Publish(userID, "match.ended", payload)
	}
	return nil
}
