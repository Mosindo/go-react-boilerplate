package discovery

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/photos"
	platformerrors "example.com/api/internal/platform/errors"
)

const (
	DefaultLimit = 10
	MaxLimit     = 20
	minAge       = 18
)

type PhotoLister interface {
	ListByUsers(ctx context.Context, userIDs []string) (map[string][]photos.Photo, error)
}

type Service struct {
	repo   Repository
	photos PhotoLister
	now    func() time.Time
}

func NewService(repo Repository, photos PhotoLister) *Service {
	return &Service{repo: repo, photos: photos, now: time.Now}
}

func (s *Service) Discover(ctx context.Context, viewerID string, limit int) ([]Card, error) {
	if limit < 1 || limit > MaxLimit {
		limit = DefaultLimit
	}
	ok, err := s.repo.Onboarded(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, platformerrors.Conflict("onboarding_required", "complete your profile and add a photo first")
	}
	if err := s.repo.TouchActive(ctx, viewerID); err != nil {
		return nil, err
	}
	candidates, err := s.repo.Candidates(ctx, viewerID, limit)
	if err != nil {
		return nil, err
	}
	return s.cards(ctx, viewerID, candidates)
}

func (s *Service) Profile(ctx context.Context, viewerID, targetID string) (Card, error) {
	c, err := s.repo.Visible(ctx, viewerID, targetID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Card{}, platformerrors.NotFound("profile_not_found", "profile not found")
		}
		return Card{}, err
	}
	cards, err := s.cards(ctx, viewerID, []Candidate{c})
	if err != nil {
		return Card{}, err
	}
	return cards[0], nil
}

// cards attaches interests and photos with two batched queries (no N+1).
func (s *Service) cards(ctx context.Context, viewerID string, cs []Candidate) ([]Card, error) {
	ids := make([]string, len(cs))
	for i, c := range cs {
		ids[i] = c.UserID
	}
	interests, err := s.repo.InterestsFor(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	pics, err := s.photos.ListByUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]Card, 0, len(cs))
	for _, c := range cs {
		card := Card{
			UserID:    c.UserID,
			FirstName: c.FirstName,
			Age:       ageOn(c.BirthDate, now),
			Gender:    c.Gender,
			Bio:       c.Bio,
			City:      c.City,
			Interests: interests[c.UserID],
			Photos:    pics[c.UserID],
		}
		if card.Interests == nil {
			card.Interests = []CardInterest{}
		}
		if card.Photos == nil {
			card.Photos = []photos.Photo{}
		}
		if c.DistanceKm != nil && c.ShowDistance {
			d := BucketDistance(*c.DistanceKm)
			card.DistanceKm = &d
		}
		out = append(out, card)
	}
	return out, nil
}

func ageOn(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}
