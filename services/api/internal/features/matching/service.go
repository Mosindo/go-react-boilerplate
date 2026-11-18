package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/people"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrSelfSwipe       = errors.New("you cannot swipe yourself")
	ErrAlreadySwiped   = errors.New("profile already processed")
	ErrNotEligible     = errors.New("profile unavailable")
	ErrProfileRequired = errors.New("create your profile first")
	ErrMatchNotFound   = errors.New("match not found")
)

type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string, coalesce bool)
}

type URLSigner interface {
	PhotoURL(photoID string) string
}

type Service struct {
	repo      Repository
	signer    URLSigner
	notifier  Notifier
	publisher realtime.Publisher
	now       func() time.Time
}

func NewService(repo Repository, signer URLSigner, notifier Notifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, signer: signer, notifier: notifier, publisher: publisher, now: time.Now}
}

func (s *Service) Swipe(ctx context.Context, actorID, targetID, decision string) (SwipeResponse, error) {
	if actorID == targetID {
		return SwipeResponse{}, ErrSelfSwipe
	}
	result, err := s.repo.Swipe(ctx, actorID, targetID, decision)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepositoryAlreadyTaken):
			return SwipeResponse{}, ErrAlreadySwiped
		case errors.Is(err, ErrRepositoryNotEligible):
			return SwipeResponse{}, ErrNotEligible
		case errors.Is(err, ErrRepositoryNoProfile):
			return SwipeResponse{}, ErrProfileRequired
		}
		return SwipeResponse{}, err
	}

	resp := SwipeResponse{Decision: decision, Matched: result.Matched}
	if !result.Matched {
		return resp, nil
	}

	forActor, err := s.summary(ctx, actorID, targetID)
	if err != nil {
		return SwipeResponse{}, err
	}
	resp.Match = &forActor
	s.announceMatch(ctx, actorID, forActor)
	if forTarget, err := s.summary(ctx, targetID, actorID); err == nil {
		s.announceMatch(ctx, targetID, forTarget)
	}
	return resp, nil
}

func (s *Service) announceMatch(ctx context.Context, recipientID string, match Match) {
	s.publisher.Publish(recipientID, realtime.Event{Type: "match.created", Data: match})
	s.notifier.Notify(ctx, recipientID, "match.created", "Nouveau match",
		"Vous et "+match.User.FirstName+" vous plaisez mutuellement. Dites bonjour !",
		map[string]string{"userId": match.User.UserID, "matchId": match.ID}, false)
}

func (s *Service) summary(ctx context.Context, viewerID, otherID string) (Match, error) {
	stored, err := s.repo.Summary(ctx, viewerID, otherID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Match{}, ErrMatchNotFound
		}
		return Match{}, err
	}
	return s.toMatch(stored), nil
}

func (s *Service) toMatch(stored storedSummary) Match {
	m := Match{
		ID:          stored.MatchID,
		MatchedAt:   stored.MatchedAt,
		HasMessages: stored.HasMessages,
		User: UserSummary{
			UserID:    stored.UserID,
			FirstName: stored.FirstName,
			Age:       people.Age(stored.Birthdate, s.now()),
		},
	}
	if stored.PhotoID != nil {
		url := s.signer.PhotoURL(*stored.PhotoID)
		m.User.PhotoURL = &url
	}
	return m
}

func (s *Service) ListMatches(ctx context.Context, userID string, limit, offset int) ([]Match, error) {
	stored, err := s.repo.ListMatches(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]Match, 0, len(stored))
	for _, m := range stored {
		out = append(out, s.toMatch(m))
	}
	return out, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, otherID string) error {
	if err := s.repo.Unmatch(ctx, userID, otherID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrMatchNotFound
		}
		return err
	}
	s.publisher.Publish(otherID, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": userID}})
	s.publisher.Publish(userID, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": otherID}})
	return nil
}
