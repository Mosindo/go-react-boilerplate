package discovery

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/people"
)

var (
	ErrProfileRequired = errors.New("complete your profile to discover people")
	ErrNotFound        = errors.New("profile not found")
)

const (
	DefaultFeedSize = 10
	MaxFeedSize     = 20
)

type URLSigner interface {
	PhotoURL(photoID string) string
}

type Service struct {
	repo   Repository
	signer URLSigner
	now    func() time.Time
}

func NewService(repo Repository, signer URLSigner) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) Feed(ctx context.Context, viewerID string, limit int) ([]Card, error) {
	if limit <= 0 || limit > MaxFeedSize {
		limit = DefaultFeedSize
	}
	ready, err := s.repo.ViewerReady(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if !ready {
		return nil, ErrProfileRequired
	}
	_ = s.repo.TouchActivity(ctx, viewerID)

	stored, err := s.repo.Candidates(ctx, viewerID, limit)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, viewerID, stored)
}

func (s *Service) Profile(ctx context.Context, viewerID, targetID string) (Card, error) {
	stored, err := s.repo.ProfileCard(ctx, viewerID, targetID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Card{}, ErrNotFound
		}
		return Card{}, err
	}
	cards, err := s.hydrate(ctx, viewerID, []storedCard{stored})
	if err != nil {
		return Card{}, err
	}
	return cards[0], nil
}

func (s *Service) hydrate(ctx context.Context, viewerID string, stored []storedCard) ([]Card, error) {
	cards := make([]Card, 0, len(stored))
	if len(stored) == 0 {
		return cards, nil
	}
	ids := make([]string, 0, len(stored))
	for _, c := range stored {
		ids = append(ids, c.UserID)
	}
	photos, err := s.repo.Photos(ctx, ids)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.Interests(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	photosByUser := map[string][]Photo{}
	for _, p := range photos {
		photosByUser[p.UserID] = append(photosByUser[p.UserID], Photo{ID: p.ID, URL: s.signer.PhotoURL(p.ID), Width: p.Width, Height: p.Height})
	}
	interestsByUser := map[string][]Interest{}
	for _, i := range interests {
		interestsByUser[i.UserID] = append(interestsByUser[i.UserID], Interest{Slug: i.Slug, Label: i.Label, Shared: i.Shared})
	}

	now := s.now()
	for _, c := range stored {
		card := Card{
			UserID:           c.UserID,
			FirstName:        c.FirstName,
			Age:              people.Age(c.Birthdate, now),
			Gender:           c.Gender,
			Bio:              c.Bio,
			JobTitle:         c.JobTitle,
			City:             c.City,
			RelationshipGoal: c.RelationshipGoal,
			SharedInterests:  c.SharedInterests,
			Interests:        interestsByUser[c.UserID],
			Photos:           photosByUser[c.UserID],
			IsMatch:          c.IsMatch,
			IsDemo:           c.IsDemo,
		}
		if c.HeightCm != nil {
			h := int(*c.HeightCm)
			card.HeightCm = &h
		}
		if c.DistanceKm != nil && c.ShowDistance && c.UserID != viewerID {
			d := people.DisplayDistanceKm(*c.DistanceKm)
			card.DistanceKm = &d
		}
		if card.Interests == nil {
			card.Interests = []Interest{}
		}
		if card.Photos == nil {
			card.Photos = []Photo{}
		}
		cards = append(cards, card)
	}
	return cards, nil
}
