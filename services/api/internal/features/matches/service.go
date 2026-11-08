package matches

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrTargetUnavailable = platformerrors.ErrNotFound
	ErrAlreadySwiped     = platformerrors.Conflict("you already reacted to this profile")
	ErrProfileIncomplete = platformerrors.New(403, "complete your profile before liking people")
	ErrMatchNotFound     = platformerrors.ErrNotFound
)

type ProfileGate interface {
	IsComplete(ctx context.Context, userID string) (bool, error)
}

type Service struct {
	repo     Repository
	cards    profiles.CardLoader
	gate     ProfileGate
	notifier notifications.Notifier
	events   realtime.Publisher
}

func NewService(repo Repository, cards profiles.CardLoader, gate ProfileGate, notifier notifications.Notifier, events realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, gate: gate, notifier: notifier, events: events}
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResponse, error) {
	if req.Action != ActionLike && req.Action != ActionPass {
		return SwipeResponse{}, platformerrors.Invalid("action must be 'like' or 'pass'")
	}
	if !profiles.IsUUID(req.UserID) {
		return SwipeResponse{}, platformerrors.Invalid("invalid userId")
	}
	if req.UserID == userID {
		return SwipeResponse{}, platformerrors.Invalid("you cannot react to yourself")
	}
	ok, err := s.gate.IsComplete(ctx, userID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !ok {
		return SwipeResponse{}, ErrProfileIncomplete
	}

	out, err := s.repo.Swipe(ctx, userID, req.UserID, req.Action)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepositoryTargetUnavailable):
			return SwipeResponse{}, ErrTargetUnavailable
		case errors.Is(err, ErrRepositoryConflict):
			return SwipeResponse{}, ErrAlreadySwiped
		}
		return SwipeResponse{}, err
	}
	res := SwipeResponse{Action: req.Action}
	if out.Match == nil {
		return res, nil
	}
	res.Matched = true

	// Each side sees the other person's card.
	mine, err := s.matchFor(ctx, userID, *out.Match)
	if err != nil {
		return SwipeResponse{}, err
	}
	res.Match = &mine

	if out.Created {
		theirs, err := s.matchFor(ctx, req.UserID, storedMatch{ID: out.Match.ID, ConversationID: out.Match.ConversationID, OtherUserID: userID, CreatedAt: out.Match.CreatedAt})
		if err == nil {
			s.events.Publish(req.UserID, realtime.Event{Type: "match.new", Data: theirs})
		}
		s.events.Publish(userID, realtime.Event{Type: "match.new", Data: mine})
		matchID := out.Match.ID
		s.notifier.Notify(ctx, req.UserID, notifications.TypeMatch, &userID, &matchID)
		other := req.UserID
		s.notifier.Notify(ctx, userID, notifications.TypeMatch, &other, &matchID)
	}
	return res, nil
}

func (s *Service) matchFor(ctx context.Context, viewerID string, m storedMatch) (Match, error) {
	cards, err := s.cards.LoadCards(ctx, viewerID, []string{m.OtherUserID})
	if err != nil {
		return Match{}, err
	}
	if len(cards) == 0 {
		return Match{}, ErrMatchNotFound
	}
	return Match{ID: m.ID, ConversationID: m.ConversationID, CreatedAt: m.CreatedAt, HasMessages: m.HasMessages, User: cards[0]}, nil
}

func (s *Service) List(ctx context.Context, userID string, before *time.Time, limit int) (MatchesResponse, error) {
	rows, err := s.repo.ListMatches(ctx, userID, before, limit+1)
	if err != nil {
		return MatchesResponse{}, err
	}
	var next *time.Time
	if len(rows) > limit {
		rows = rows[:limit]
		t := rows[len(rows)-1].CreatedAt
		next = &t
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.OtherUserID
	}
	cards, err := s.cards.LoadCards(ctx, userID, ids)
	if err != nil {
		return MatchesResponse{}, err
	}
	byID := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]Match, 0, len(rows))
	for _, r := range rows {
		if c, ok := byID[r.OtherUserID]; ok {
			out = append(out, Match{ID: r.ID, ConversationID: r.ConversationID, CreatedAt: r.CreatedAt, HasMessages: r.HasMessages, User: c})
		}
	}
	return MatchesResponse{Matches: out, NextBefore: next}, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	if !profiles.IsUUID(matchID) {
		return ErrMatchNotFound
	}
	if _, err := s.repo.Unmatch(ctx, userID, matchID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrMatchNotFound
		}
		return err
	}
	return nil
}
