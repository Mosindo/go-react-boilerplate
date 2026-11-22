package profiles

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrNotVisible = errors.New("profile not visible")

// PhotoURLer issues viewer-scoped photo URLs (see platform/media.Signer).
type PhotoURLer interface {
	PhotoURL(photoID string, now time.Time) string
}

type Service struct {
	repo    Repository
	urls    PhotoURLer
	now     func() time.Time
	mu      sync.RWMutex
	catalog []Interest
}

func NewService(repo Repository, urls PhotoURLer) *Service {
	return &Service{repo: repo, urls: urls, now: time.Now}
}

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	p, err := s.repo.GetOwn(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	interests, err := s.repo.InterestsFor(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	photos, err := s.repo.PhotosFor(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	now := s.now()
	out := OwnProfile{
		UserID:           p.UserID,
		Bio:              p.Bio,
		JobTitle:         p.JobTitle,
		City:             p.City,
		HasLocation:      p.Latitude != nil,
		Discoverable:     p.Discoverable,
		ShowDistance:     p.ShowDistance,
		ModerationStatus: p.ModerationStatus,
		Interests:        nonNil(interests[userID]),
		Photos:           s.photoViews(photos[userID], now),
		Preferences:      p.Preferences,
		MissingFields:    missingFields(p),
	}
	if out.Preferences.InterestedIn == nil {
		out.Preferences.InterestedIn = []string{}
	}
	if p.FirstName != nil {
		out.FirstName = *p.FirstName
	}
	if p.Gender != nil {
		out.Gender = *p.Gender
	}
	if p.Birthdate != nil {
		out.Birthdate = p.Birthdate.Format(birthdateLayout)
		age := AgeAt(*p.Birthdate, now)
		out.Age = &age
	}
	out.IsComplete = len(out.MissingFields) == 0
	return out, nil
}

func (s *Service) Update(ctx context.Context, userID string, patch ProfilePatch) (OwnProfile, error) {
	current, err := s.repo.GetOwn(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	var u profileUpdate
	if patch.FirstName != nil {
		name, err := NormalizeFirstName(*patch.FirstName)
		if err != nil {
			return OwnProfile{}, err
		}
		u.FirstName = &name
	}
	if patch.Birthdate != nil {
		d, err := ParseBirthdate(*patch.Birthdate, s.now())
		if err != nil {
			return OwnProfile{}, err
		}
		// Age is a trust signal: it can be set once, then only support can change it.
		if current.Birthdate != nil && !current.Birthdate.Equal(d) {
			return OwnProfile{}, ErrBirthdateLocked
		}
		u.Birthdate = &d
	}
	if patch.Gender != nil {
		if !ValidGender(*patch.Gender) {
			return OwnProfile{}, ErrInvalidGender
		}
		u.Gender = patch.Gender
	}
	if patch.Bio != nil {
		bio, err := NormalizeText(*patch.Bio, MaxBioRunes, ErrBioTooLong)
		if err != nil {
			return OwnProfile{}, err
		}
		u.Bio = &bio
	}
	if patch.JobTitle != nil {
		job, err := NormalizeText(*patch.JobTitle, MaxJobTitleRunes, ErrJobTitleTooLong)
		if err != nil {
			return OwnProfile{}, err
		}
		u.JobTitle = &job
	}
	u.Discoverable = patch.Discoverable
	u.ShowDistance = patch.ShowDistance

	var slugs []string
	if patch.Interests != nil {
		slugs, err = NormalizeInterestSlugs(*patch.Interests)
		if err != nil {
			return OwnProfile{}, err
		}
	}

	if err := s.repo.UpdateProfile(ctx, userID, u); err != nil {
		return OwnProfile{}, err
	}
	if patch.Interests != nil {
		if err := s.repo.SetInterests(ctx, userID, slugs); err != nil {
			return OwnProfile{}, err
		}
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, req LocationRequest) (OwnProfile, error) {
	lat, lng, err := RoundCoordinates(*req.Latitude, *req.Longitude)
	if err != nil {
		return OwnProfile{}, err
	}
	city, err := NormalizeText(req.City, MaxCityRunes, ErrCityTooLong)
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.UpdateLocation(ctx, userID, lat, lng, city); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req PreferencesRequest) (OwnProfile, error) {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.UpdatePreferences(ctx, userID, prefs); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

// Interests returns the static catalog, cached after the first load.
func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	s.mu.RLock()
	cached := s.catalog
	s.mu.RUnlock()
	if cached != nil {
		return cached, nil
	}
	list, err := s.repo.ListInterests(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.catalog = list
	s.mu.Unlock()
	return list, nil
}

// Cards builds public cards for userIDs (input order kept) in 3 queries,
// whatever the page size: no N+1.
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) ([]Card, error) {
	rows, err := s.repo.Cards(ctx, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.InterestsFor(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.PhotosFor(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	now := s.now()
	cards := make([]Card, 0, len(rows))
	for _, r := range rows {
		c := Card{
			UserID:    r.UserID,
			FirstName: r.FirstName,
			Age:       AgeAt(r.Birthdate, now),
			Gender:    r.Gender,
			Bio:       r.Bio,
			JobTitle:  r.JobTitle,
			City:      r.City,
			Interests: nonNil(interests[r.UserID]),
			Photos:    s.photoViews(photos[r.UserID], now),
			IsDemo:    r.IsDemo,
		}
		if r.ShowDistance && r.DistanceKm != nil {
			d := DisplayDistanceKm(*r.DistanceKm)
			c.DistanceKm = &d
		}
		cards = append(cards, c)
	}
	return cards, nil
}

// PublicProfile returns target's card if viewer is allowed to see it.
func (s *Service) PublicProfile(ctx context.Context, viewerID, targetID string) (Card, error) {
	if viewerID == targetID {
		return Card{}, ErrNotVisible
	}
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return Card{}, err
	}
	if !ok {
		return Card{}, ErrNotVisible
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return Card{}, err
	}
	if len(cards) == 0 {
		return Card{}, ErrNotVisible
	}
	return cards[0], nil
}

func (s *Service) photoViews(stored []storedPhoto, now time.Time) []Photo {
	out := make([]Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, Photo{ID: p.ID, URL: s.urls.PhotoURL(p.ID, now), Position: p.Position, Width: p.Width, Height: p.Height})
	}
	return out
}

func nonNil[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}
