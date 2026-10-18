package profiles

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/photos"
)

var (
	ErrInvalidName      = errors.New("first name may only contain letters, spaces, hyphens and apostrophes")
	ErrInvalidBirthDate = errors.New("birthDate must be a valid date (YYYY-MM-DD)")
	ErrUnderage         = errors.New("you must be at least 18 years old")
	ErrTooOld           = errors.New("birthDate is out of range")
	ErrBioTooLong       = errors.New("bio must be at most 500 characters")
	ErrCityTooLong      = errors.New("city must be at most 80 characters")
	ErrUnknownInterest  = errors.New("unknown interest")
	ErrTooManyInterests = errors.New("you can pick up to 8 interests")
	ErrInvalidPrefs     = errors.New("invalid preferences")
)

var nameRe = regexp.MustCompile(`^[\p{L}][\p{L}\p{M} '’.\-]*$`)

// PhotoLister is the slice of the photos service profiles depend on.
type PhotoLister interface {
	ListForUsers(ctx context.Context, userIDs []string) (map[string][]photos.Photo, error)
}

type Service struct {
	repo   Repository
	photos PhotoLister
	now    func() time.Time
}

func NewService(repo Repository, photos PhotoLister) *Service {
	return &Service{repo: repo, photos: photos, now: time.Now}
}

// AgeOn computes completed years between birth and now.
func AgeOn(birth, now time.Time) int {
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return age
}

// ApproxDistanceKm turns an exact distance into a coarse bucket so that
// repeated queries cannot be triangulated into a precise position:
// "1" below 2 km, then 1 km steps up to 10 km, then 5 km steps.
func ApproxDistanceKm(km float64) int {
	switch {
	case km < 2:
		return 1
	case km <= 10:
		return int(math.Ceil(km))
	default:
		return int(math.Ceil(km/5)) * 5
	}
}

// RoundCoordinate keeps ~1.1 km precision: enough for "nearby", too coarse to
// locate a home.
func RoundCoordinate(v float64) float64 { return math.Round(v*100) / 100 }

func cleanText(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}

func (s *Service) validate(userID string, req UpsertRequest) (storedProfile, []string, error) {
	name := strings.Join(strings.Fields(cleanText(req.FirstName)), " ")
	if n := utf8.RuneCountInString(name); n < 1 || n > MaxNameRunes || !nameRe.MatchString(name) {
		return storedProfile{}, nil, ErrInvalidName
	}

	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return storedProfile{}, nil, ErrInvalidBirthDate
	}
	age := AgeOn(birth, s.now())
	if age < MinAge {
		return storedProfile{}, nil, ErrUnderage
	}
	if age > 120 {
		return storedProfile{}, nil, ErrTooOld
	}

	bio := cleanText(req.Bio)
	if utf8.RuneCountInString(bio) > MaxBioRunes {
		return storedProfile{}, nil, ErrBioTooLong
	}
	city := strings.Join(strings.Fields(cleanText(req.City)), " ")
	if utf8.RuneCountInString(city) > MaxCityRunes {
		return storedProfile{}, nil, ErrCityTooLong
	}

	prefs := req.Preferences
	if prefs.AgeMin < MinAge || prefs.AgeMax > MaxAge || prefs.AgeMin > prefs.AgeMax || len(prefs.InterestedIn) == 0 {
		return storedProfile{}, nil, ErrInvalidPrefs
	}
	seenGender := map[string]bool{}
	var interestedIn []string
	for _, g := range prefs.InterestedIn {
		if !seenGender[g] {
			seenGender[g] = true
			interestedIn = append(interestedIn, g)
		}
	}
	prefs.InterestedIn = interestedIn

	seen := map[string]bool{}
	var slugs []string
	for _, slug := range req.Interests {
		slug = strings.ToLower(strings.TrimSpace(slug))
		if slug != "" && !seen[slug] {
			seen[slug] = true
			slugs = append(slugs, slug)
		}
	}
	if len(slugs) > MaxInterests {
		return storedProfile{}, nil, ErrTooManyInterests
	}

	return storedProfile{
		UserID:      userID,
		FirstName:   name,
		BirthDate:   birth,
		Gender:      req.Gender,
		Bio:         bio,
		City:        city,
		Latitude:    RoundCoordinate(*req.Latitude),
		Longitude:   RoundCoordinate(*req.Longitude),
		Preferences: prefs,
	}, slugs, nil
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertRequest) (MyProfile, error) {
	p, slugs, err := s.validate(userID, req)
	if err != nil {
		return MyProfile{}, err
	}
	catalog, err := s.repo.ListInterests(ctx)
	if err != nil {
		return MyProfile{}, err
	}
	known := make(map[string]bool, len(catalog))
	for _, i := range catalog {
		known[i.Slug] = true
	}
	for _, slug := range slugs {
		if !known[slug] {
			return MyProfile{}, ErrUnknownInterest
		}
	}
	if err := s.repo.Upsert(ctx, p, slugs); err != nil {
		return MyProfile{}, err
	}
	return s.Get(ctx, userID)
}

func (s *Service) Get(ctx context.Context, userID string) (MyProfile, error) {
	p, err := s.repo.Get(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}
	interests, err := s.repo.InterestsForUsers(ctx, []string{userID})
	if err != nil {
		return MyProfile{}, err
	}
	pics, err := s.photos.ListForUsers(ctx, []string{userID})
	if err != nil {
		return MyProfile{}, err
	}
	out := MyProfile{
		UserID:       p.UserID,
		FirstName:    p.FirstName,
		BirthDate:    p.BirthDate.Format("2006-01-02"),
		Age:          AgeOn(p.BirthDate, s.now()),
		Gender:       p.Gender,
		Bio:          p.Bio,
		City:         p.City,
		Latitude:     p.Latitude,
		Longitude:    p.Longitude,
		IsVisible:    p.IsVisible,
		ShowDistance: p.ShowDistance,
		Interests:    interests[userID],
		Preferences:  p.Preferences,
		Photos:       pics[userID],
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
	if out.Interests == nil {
		out.Interests = []InterestRef{}
	}
	if out.Photos == nil {
		out.Photos = []photos.Photo{}
	}
	return out, nil
}

func (s *Service) UpdateSettings(ctx context.Context, userID string, req SettingsRequest) (MyProfile, error) {
	if err := s.repo.UpdateSettings(ctx, userID, req.IsVisible, req.ShowDistance); err != nil {
		return MyProfile{}, err
	}
	return s.Get(ctx, userID)
}

func (s *Service) Interests(ctx context.Context) ([]InterestRef, error) {
	return s.repo.ListInterests(ctx)
}

// Hydrate turns raw cards into public profiles with two batched queries
// (interests and photos), whatever the number of cards.
func (s *Service) Hydrate(ctx context.Context, cards []Card) ([]PublicProfile, error) {
	ids := make([]string, len(cards))
	for i, c := range cards {
		ids[i] = c.UserID
	}
	interests, err := s.repo.InterestsForUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	pics, err := s.photos.ListForUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]PublicProfile, 0, len(cards))
	for _, c := range cards {
		pp := PublicProfile{
			UserID:    c.UserID,
			FirstName: c.FirstName,
			Age:       AgeOn(c.BirthDate, now),
			Gender:    c.Gender,
			Bio:       c.Bio,
			City:      c.City,
			Interests: interests[c.UserID],
			Photos:    pics[c.UserID],
		}
		if c.DistanceKm != nil {
			d := ApproxDistanceKm(*c.DistanceKm)
			pp.DistanceKm = &d
		}
		if pp.Interests == nil {
			pp.Interests = []InterestRef{}
		}
		if pp.Photos == nil {
			pp.Photos = []photos.Photo{}
		}
		out = append(out, pp)
	}
	return out, nil
}
