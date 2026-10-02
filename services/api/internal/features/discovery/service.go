package discovery

import (
	"context"
	"errors"
	"strings"

	"example.com/api/internal/features/profiles"
	"github.com/google/uuid"
)

var (
	ErrIncomplete = errors.New("complete your profile (details, preferences and at least one photo) to discover people")
	ErrNotFound   = errors.New("profile not found")
)

type CardSource interface {
	Cards(ctx context.Context, viewerID string, ids []string) ([]profiles.Card, error)
}

type ProfileChecker interface {
	Status(ctx context.Context, userID string) (profiles.Status, error)
}

type Service struct {
	repo     Repository
	cards    CardSource
	profiles ProfileChecker
}

func NewService(repo Repository, cards CardSource, p ProfileChecker) *Service {
	return &Service{repo: repo, cards: cards, profiles: p}
}

func (s *Service) Discover(ctx context.Context, viewerID string, limit int) ([]profiles.Card, error) {
	st, err := s.profiles.Status(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if !st.Complete {
		return nil, ErrIncomplete
	}
	ids, err := s.repo.Candidates(ctx, viewerID, limit)
	if err != nil {
		return nil, err
	}
	return s.cards.Cards(ctx, viewerID, ids)
}

func (s *Service) Profile(ctx context.Context, viewerID, targetID string) (profiles.Card, error) {
	targetID = strings.ToLower(targetID)
	if _, err := uuid.Parse(targetID); err != nil {
		return profiles.Card{}, ErrNotFound
	}
	if targetID != viewerID {
		ok, err := s.repo.CanView(ctx, viewerID, targetID)
		if err != nil {
			return profiles.Card{}, err
		}
		if !ok {
			return profiles.Card{}, ErrNotFound
		}
	}
	cards, err := s.cards.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return profiles.Card{}, err
	}
	if len(cards) == 0 {
		return profiles.Card{}, ErrNotFound
	}
	return cards[0], nil
}
