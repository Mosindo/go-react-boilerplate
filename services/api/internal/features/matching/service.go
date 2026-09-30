package matching

import (
	"context"
	"errors"
	"log"
	"strings"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrCannotSwipeSelf   = apperr.Validation("you cannot swipe on yourself")
	ErrProfileIncomplete = apperr.Conflict("profile_incomplete", "complete your profile before discovering people")
	ErrTargetUnavailable = apperr.NotFound("profile not available")
	ErrAlreadySwiped     = apperr.Conflict("already_swiped", "you already responded to this profile")
	ErrMatchNotFound     = apperr.NotFound("match not found")
)

type SummaryProvider interface {
	Summaries(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Summary, error)
}

type Service struct {
	repo      Repository
	profiles  SummaryProvider
	notifier  notifications.Notifier
	publisher realtime.Publisher
}

func NewService(repo Repository, profiles SummaryProvider, notifier notifications.Notifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, notifier: notifier, publisher: publisher}
}

// Swipe records a like or pass. A like answering an existing like creates a
// match and its conversation atomically, then notifies both members.
func (s *Service) Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeResponse, error) {
	// Canonical form: the self check and the pair lock key compare strings.
	targetID = strings.ToLower(targetID)
	if swiperID == targetID {
		return SwipeResponse{}, ErrCannotSwipeSelf
	}
	if action != ActionLike && action != ActionPass {
		return SwipeResponse{}, apperr.Validation("action must be like or pass")
	}

	result, err := s.repo.Swipe(ctx, swiperID, targetID, action)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepositorySwiperIncomplete):
			return SwipeResponse{}, ErrProfileIncomplete
		case errors.Is(err, ErrRepositoryTargetUnavailable):
			return SwipeResponse{}, ErrTargetUnavailable
		case errors.Is(err, ErrRepositoryAlreadySwiped):
			return SwipeResponse{}, ErrAlreadySwiped
		}
		return SwipeResponse{}, err
	}
	if result == nil {
		return SwipeResponse{Matched: false}, nil
	}

	summaries, err := s.profiles.Summaries(ctx, swiperID, []string{swiperID, targetID})
	if err != nil {
		// The match is committed; degrade gracefully on summary failure.
		log.Printf(`{"event":"match_summary_failed","error":%q}`, err.Error())
		summaries = map[string]profiles.Summary{}
	}

	forSwiper := Match{ID: result.MatchID, ConversationID: result.ConversationID, CreatedAt: result.MatchedAt, User: summaries[targetID]}
	forTarget := Match{ID: result.MatchID, ConversationID: result.ConversationID, CreatedAt: result.MatchedAt, User: summaries[swiperID]}

	s.publisher.Publish(ctx, []string{swiperID}, realtime.Event{Type: "match.created", ConversationID: result.ConversationID, Data: forSwiper})
	s.publisher.Publish(ctx, []string{targetID}, realtime.Event{Type: "match.created", ConversationID: result.ConversationID, Data: forTarget})

	data := map[string]string{"matchId": result.MatchID, "conversationId": result.ConversationID}
	s.notifier.Notify(ctx, targetID, notifications.TypeMatch, "Nouveau match", "Vous avez un match avec "+nameOr(summaries[swiperID])+" !", withUser(data, swiperID))
	s.notifier.Notify(ctx, swiperID, notifications.TypeMatch, "Nouveau match", "Vous avez un match avec "+nameOr(summaries[targetID])+" !", withUser(data, targetID))

	return SwipeResponse{Matched: true, Match: &forSwiper}, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	otherID, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrMatchNotFound
		}
		return err
	}
	s.publisher.Publish(ctx, []string{userID, otherID}, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	return nil
}

func nameOr(summary profiles.Summary) string {
	if summary.FirstName == "" {
		return "quelqu'un"
	}
	return summary.FirstName
}

func withUser(data map[string]string, userID string) map[string]string {
	out := make(map[string]string, len(data)+1)
	for k, v := range data {
		out[k] = v
	}
	out["userId"] = userID
	return out
}
