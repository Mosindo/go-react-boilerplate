package profiles

import (
	"context"
	"errors"
	"slices"
	"time"
)

var ErrProfileHidden = errors.New("profile not available")

// PhotoURLSigner issues short-lived URLs for private photos.
type PhotoURLSigner interface {
	PhotoURL(photoID string) string
}

type Service struct {
	repo   Repository
	signer PhotoURLSigner
	now    func() time.Time
}

func NewService(repo Repository, signer PhotoURLSigner) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	p, err := s.repo.GetOwn(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	photos, err := s.repo.LoadPhotos(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	interests, err := s.repo.LoadInterests(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	_ = s.repo.TouchActivity(ctx, userID)

	out := OwnProfile{
		UserID:           p.UserID,
		FirstName:        p.FirstName,
		Gender:           p.Gender,
		Bio:              p.Bio,
		JobTitle:         p.JobTitle,
		HeightCm:         p.HeightCm,
		RelationshipGoal: p.RelationshipGoal,
		City:             p.City,
		HasLocation:      p.HasLocation,
		IsPaused:         p.IsPaused,
		ShowDistance:     p.ShowDistance,
		IsComplete:       p.IsComplete,
		Missing:          missingFields(p, len(photos)),
		Photos:           s.photoViews(photos),
		Interests:        nonNilInterests(interests[userID]),
		Preferences:      p.Preferences,
	}
	if p.Birthdate != nil {
		formatted := p.Birthdate.Format("2006-01-02")
		age := AgeOn(*p.Birthdate, s.now())
		out.Birthdate = &formatted
		out.Age = &age
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, userID string, req UpdateProfileRequest) (OwnProfile, error) {
	patch, err := validatePatch(req, s.now())
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.ApplyPatch(ctx, userID, patch); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lon float64, city string) (OwnProfile, error) {
	rLat, rLon, c, err := validateLocation(lat, lon, city)
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.UpdateLocation(ctx, userID, rLat, rLon, c); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (OwnProfile, error) {
	current, err := s.repo.GetOwn(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	next, err := applyPreferences(current.Preferences, req)
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.UpdatePreferences(ctx, userID, next); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// GetPublic returns another member's profile if the viewer may see it.
func (s *Service) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if !ok {
		return PublicProfile{}, ErrProfileHidden
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	if len(cards) == 0 {
		return PublicProfile{}, ErrProfileHidden
	}
	return cards[0], nil
}

// Cards batch-loads public profiles (3 queries regardless of the number of
// users) and preserves the order of userIDs. Callers are responsible for
// authorization; this is used by discovery and chat after their own checks.
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) ([]PublicProfile, error) {
	if len(userIDs) == 0 {
		return []PublicProfile{}, nil
	}
	stored, err := s.repo.LoadCards(ctx, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.LoadPhotos(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.LoadInterests(ctx, append(slices.Clone(userIDs), viewerID))
	if err != nil {
		return nil, err
	}

	photosByUser := make(map[string][]storedPhoto, len(userIDs))
	for _, p := range photos {
		photosByUser[p.UserID] = append(photosByUser[p.UserID], p)
	}
	viewerInterests := make(map[string]struct{})
	for _, i := range interests[viewerID] {
		viewerInterests[i.Slug] = struct{}{}
	}

	byID := make(map[string]PublicProfile, len(stored))
	now := s.now()
	for _, c := range stored {
		card := PublicProfile{
			UserID:           c.UserID,
			FirstName:        c.FirstName,
			Age:              AgeOn(c.Birthdate, now),
			Gender:           c.Gender,
			Bio:              c.Bio,
			JobTitle:         c.JobTitle,
			HeightCm:         c.HeightCm,
			RelationshipGoal: c.RelationshipGoal,
			City:             c.City,
			Photos:           s.photoViews(photosByUser[c.UserID]),
			Interests:        nonNilInterests(interests[c.UserID]),
			SharedInterests:  []string{},
			IsDemo:           c.IsDemo,
			LastActiveAt:     c.LastActiveAt,
		}
		if c.DistanceKm != nil && c.ShowDistance {
			d := ApproximateDistance(*c.DistanceKm)
			card.DistanceKm = &d
		}
		for _, i := range card.Interests {
			if _, ok := viewerInterests[i.Slug]; ok {
				card.SharedInterests = append(card.SharedInterests, i.Slug)
			}
		}
		byID[c.UserID] = card
	}

	out := make([]PublicProfile, 0, len(userIDs))
	for _, id := range userIDs {
		if card, ok := byID[id]; ok {
			out = append(out, card)
		}
	}
	return out, nil
}

func (s *Service) photoViews(photos []storedPhoto) []PhotoView {
	out := make([]PhotoView, 0, len(photos))
	for _, p := range photos {
		out = append(out, PhotoView{
			ID:       p.ID,
			URL:      s.signer.PhotoURL(p.ID),
			Position: p.Position,
			Width:    p.Width,
			Height:   p.Height,
		})
	}
	return out
}

func nonNilInterests(in []Interest) []Interest {
	if in == nil {
		return []Interest{}
	}
	return in
}
