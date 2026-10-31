package profiles

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"example.com/api/internal/platform/validate"
)

var (
	ErrInvalidName      = errors.New("invalid first name")
	ErrInvalidGender    = errors.New("invalid gender")
	ErrInvalidBio       = errors.New("invalid bio")
	ErrInvalidCity      = errors.New("invalid city")
	ErrInvalidLocation  = errors.New("invalid location")
	ErrTooManyInterests = errors.New("too many interests")
	ErrInvalidPrefs     = errors.New("invalid preferences")
	ErrProfileMissing   = errors.New("create your profile first")
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	stored, err := s.repo.GetOwn(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return OwnProfile{UserID: userID, Interests: []Interest{}, Photos: []PhotoRef{}, Preferences: prefs}, nil
	}
	if err != nil {
		return OwnProfile{}, err
	}
	interests, err := s.repo.InterestsFor(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	photoIDs, err := s.repo.PhotoIDsFor(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	photos := toPhotoRefs(photoIDs[userID])
	own := OwnProfile{
		UserID:         userID,
		FirstName:      stored.FirstName,
		Age:            validate.AgeAt(stored.BirthDate, s.now()),
		Gender:         stored.Gender,
		Bio:            stored.Bio,
		City:           stored.City,
		Location:       stored.Location,
		IsDiscoverable: stored.IsDiscoverable,
		ShowDistance:   stored.ShowDistance,
		Interests:      orEmptyInterests(interests[userID]),
		Photos:         photos,
		Preferences:    prefs,
		Exists:         true,
	}
	own.Complete = own.FirstName != "" && own.Gender != "" && len(photos) > 0 && len(prefs.InterestedIn) > 0
	return own, nil
}

func (s *Service) Update(ctx context.Context, userID string, req UpdateProfileRequest) (OwnProfile, error) {
	name, ok := validate.CleanText(req.FirstName, MaxNameRunes)
	if !ok || name == "" {
		return OwnProfile{}, ErrInvalidName
	}
	if req.Gender != "" && !validGenders[req.Gender] {
		return OwnProfile{}, ErrInvalidGender
	}
	bio, ok := validate.CleanText(req.Bio, MaxBioRunes)
	if !ok {
		return OwnProfile{}, ErrInvalidBio
	}
	city, ok := validate.CleanText(req.City, MaxCityRunes)
	if !ok {
		return OwnProfile{}, ErrInvalidCity
	}
	loc, err := normalizeLocation(req.Location)
	if err != nil {
		return OwnProfile{}, err
	}
	slugs := dedupe(req.Interests)
	if len(slugs) > MaxInterests {
		return OwnProfile{}, ErrTooManyInterests
	}
	in := ProfileInput{
		FirstName: name, Gender: req.Gender, Bio: bio, City: city, Location: loc, InterestSlugs: slugs,
		IsDiscoverable: req.IsDiscoverable == nil || *req.IsDiscoverable,
		ShowDistance:   req.ShowDistance == nil || *req.ShowDistance,
	}
	if err := s.repo.Upsert(ctx, userID, in); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (OwnProfile, error) {
	interested := dedupe(req.InterestedIn)
	for _, g := range interested {
		if !validGenders[g] {
			return OwnProfile{}, ErrInvalidPrefs
		}
	}
	if len(interested) == 0 || req.AgeMin < validate.MinAge || req.AgeMax > validate.MaxAge || req.AgeMin > req.AgeMax ||
		req.MaxDistanceKm < 1 || req.MaxDistanceKm > 500 {
		return OwnProfile{}, ErrInvalidPrefs
	}
	if _, err := s.repo.GetOwn(ctx, userID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return OwnProfile{}, ErrProfileMissing
		}
		return OwnProfile{}, err
	}
	err := s.repo.SavePreferences(ctx, userID, Preferences{
		InterestedIn: interested, AgeMin: req.AgeMin, AgeMax: req.AgeMax, MaxDistanceKm: req.MaxDistanceKm,
	})
	if err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// Card returns the public profile of target as seen by viewer.
func (s *Service) Card(ctx context.Context, viewerID, targetID string) (Card, error) {
	if viewerID != targetID {
		ok, err := s.repo.CanView(ctx, viewerID, targetID)
		if err != nil {
			return Card{}, err
		}
		if !ok {
			return Card{}, ErrNotFound
		}
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return Card{}, err
	}
	if len(cards) == 0 {
		return Card{}, ErrNotFound
	}
	return cards[0], nil
}

// Cards assembles public cards for ids (in the given order) with three
// batched queries, regardless of len(ids). Callers must already have checked
// that viewer may see those users.
func (s *Service) Cards(ctx context.Context, viewerID string, ids []string) ([]Card, error) {
	rows, err := s.repo.LoadCards(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.InterestsFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	photoIDs, err := s.repo.PhotoIDsFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]Card, 0, len(ids))
	for _, id := range ids {
		row, ok := rows[id]
		if !ok {
			continue
		}
		birth := time.Date(row.BirthYear, time.Month(row.BirthMonth), row.BirthDay, 0, 0, 0, 0, time.UTC)
		card := Card{
			ID: row.ID, FirstName: row.FirstName, Age: validate.AgeAt(birth, now), Gender: row.Gender,
			Bio: row.Bio, City: row.City, Interests: orEmptyInterests(interests[id]), Photos: toPhotoRefs(photoIDs[id]),
		}
		if row.DistanceKm != nil {
			d := ApproxDistance(*row.DistanceKm)
			card.DistanceKm = &d
		}
		out = append(out, card)
	}
	return out, nil
}

// ApproxDistance buckets a distance so exact positions cannot be recovered:
// 1 km steps up to 10 km, then 5 km steps. The minimum shown is 1 km.
func ApproxDistance(km float64) int {
	if km <= 1 {
		return 1
	}
	if km <= 10 {
		return int(math.Ceil(km))
	}
	return int(math.Ceil(km/5)) * 5
}

func normalizeLocation(l *Location) (*Location, error) {
	if l == nil {
		return nil, nil
	}
	if math.IsNaN(l.Latitude) || math.IsNaN(l.Longitude) || l.Latitude < -90 || l.Latitude > 90 || l.Longitude < -180 || l.Longitude > 180 {
		return nil, ErrInvalidLocation
	}
	// Two decimals ≈ 1.1 km: coarse enough that the precise position is never stored.
	return &Location{Latitude: math.Round(l.Latitude*100) / 100, Longitude: math.Round(l.Longitude*100) / 100}, nil
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.ToLower(strings.TrimSpace(v))
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func toPhotoRefs(ids []string) []PhotoRef {
	out := make([]PhotoRef, 0, len(ids))
	for _, id := range ids {
		out = append(out, PhotoRef{ID: id, URL: photoURL(id)})
	}
	return out
}

func orEmptyInterests(in []Interest) []Interest {
	if in == nil {
		return []Interest{}
	}
	return in
}
