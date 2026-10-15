package matching

import (
	"context"
	"errors"
	"log"
	"strings"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/users"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrProfileNotFound = platformerrors.NotFound("profile_not_found", "this profile is not available")
	ErrAlreadySwiped   = platformerrors.Conflict("already_swiped", "you already made a choice for this person")
	ErrSelfSwipe       = platformerrors.BadRequest("invalid_target", "you cannot choose yourself")
	ErrNotMatched      = platformerrors.NotFound("match_not_found", "match not found")
	ErrInvalidExclude  = platformerrors.BadRequest("invalid_exclude", "exclude must be a comma separated list of user ids")
)

// Profiles is the part of the users module that matching needs.
type Profiles interface {
	RequireComplete(ctx context.Context, userID string) error
	BuildCards(ctx context.Context, viewerID string, bases []users.CardBase) ([]users.Card, error)
	TouchActive(ctx context.Context, userID string)
}

type Notifier interface {
	Notify(ctx context.Context, userID string, n notifications.NewNotification) error
}

type Publisher interface {
	Publish(userID string, event realtime.Event)
}

type Service struct {
	repo      Repository
	profiles  Profiles
	notifier  Notifier
	publisher Publisher
}

func NewService(repo Repository, profiles Profiles, notifier Notifier, publisher Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, notifier: notifier, publisher: publisher}
}

// Discover returns the next profiles for the viewer. exclude lists ids the
// client already holds so a prefetch never repeats a card.
func (s *Service) Discover(ctx context.Context, viewerID string, exclude []string, limit int) ([]users.Card, error) {
	if err := s.profiles.RequireComplete(ctx, viewerID); err != nil {
		return nil, err
	}
	s.profiles.TouchActive(ctx, viewerID)
	if limit < 1 {
		limit = defaultDiscoverLimit
	}
	if limit > maxDiscoverLimit {
		limit = maxDiscoverLimit
	}
	bases, err := s.repo.Candidates(ctx, viewerID, exclude, limit)
	if err != nil {
		return nil, err
	}
	return s.profiles.BuildCards(ctx, viewerID, bases)
}

func (s *Service) Profile(ctx context.Context, viewerID, targetID string) (users.Card, error) {
	if viewerID == targetID {
		return users.Card{}, ErrProfileNotFound
	}
	if err := s.profiles.RequireComplete(ctx, viewerID); err != nil {
		return users.Card{}, err
	}
	base, err := s.repo.Candidate(ctx, viewerID, targetID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotEligible) {
			return users.Card{}, ErrProfileNotFound
		}
		return users.Card{}, err
	}
	cards, err := s.profiles.BuildCards(ctx, viewerID, []users.CardBase{base})
	if err != nil || len(cards) == 0 {
		return users.Card{}, err
	}
	return cards[0], nil
}

// Swipe records a like or a pass. A like that completes a mutual like creates a match.
func (s *Service) Swipe(ctx context.Context, viewerID, targetID, action string) (SwipeResponse, error) {
	targetID = strings.ToLower(targetID)
	if viewerID == targetID {
		return SwipeResponse{}, ErrSelfSwipe
	}
	if err := s.profiles.RequireComplete(ctx, viewerID); err != nil {
		return SwipeResponse{}, err
	}
	result, err := s.repo.Swipe(ctx, viewerID, targetID, action)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepositoryNotEligible):
			return SwipeResponse{}, ErrProfileNotFound
		case errors.Is(err, ErrRepositoryAlreadySwiped):
			return SwipeResponse{}, ErrAlreadySwiped
		}
		return SwipeResponse{}, err
	}
	if !result.Matched {
		return SwipeResponse{}, nil
	}

	s.announceMatch(ctx, viewerID, targetID, result)
	return SwipeResponse{
		Matched: true,
		Match: &MatchInfo{
			ID:             result.MatchID,
			ConversationID: result.ConversationID,
			UserID:         targetID,
			FirstName:      result.TargetName,
		},
	}, nil
}

// announceMatch tells both people. The match is already committed, so
// notification failures are logged and never fail the swipe.
func (s *Service) announceMatch(ctx context.Context, viewerID, targetID string, r SwipeResult) {
	for _, side := range []struct{ user, peer, peerName string }{
		{viewerID, targetID, r.TargetName},
		{targetID, viewerID, r.ViewerName},
	} {
		data := map[string]any{
			"matchId":        r.MatchID,
			"conversationId": r.ConversationID,
			"userId":         side.peer,
			"firstName":      side.peerName,
		}
		s.publisher.Publish(side.user, realtime.Event{Type: "match.new", Data: data})
		err := s.notifier.Notify(ctx, side.user, notifications.NewNotification{
			Type:      notifications.TypeMatch,
			Title:     "C'est un match !",
			Body:      "Vous et " + side.peerName + " vous êtes plu. Dites bonjour !",
			Data:      data,
			DedupeKey: "match:" + r.MatchID,
		})
		if err != nil {
			log.Printf(`{"event":"match_notification_failed","error":%q}`, err.Error())
		}
	}
}

func (s *Service) Unmatch(ctx context.Context, viewerID, targetID string) error {
	targetID = strings.ToLower(targetID)
	conversationID, err := s.repo.Unmatch(ctx, viewerID, targetID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotMatched) {
			return ErrNotMatched
		}
		return err
	}
	event := realtime.Event{Type: "match.removed", Data: map[string]any{"userId": viewerID, "conversationId": conversationID}}
	s.publisher.Publish(targetID, event)
	return nil
}
