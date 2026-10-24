package profiles

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/geo"
)

// ValidationError is a user-fixable input problem; Message is safe to show to the client.
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

func invalid(msg string) error { return ValidationError{Message: msg} }

const (
	maxFirstNameRunes = 40
	maxBioRunes       = 500
	maxCityRunes      = 80
	dateLayout        = "2006-01-02"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// Own returns the caller's private profile; ErrNotFound while onboarding has not created it yet.
func (s *Service) Own(ctx context.Context, userID string) (Own, error) {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return Own{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return Own{}, err
	}
	interests, err := s.repo.InterestsForUsers(ctx, []string{userID})
	if err != nil {
		return Own{}, err
	}
	photos, err := s.repo.PhotosForUsers(ctx, []string{userID})
	if err != nil {
		return Own{}, err
	}
	own := Own{
		UserID:       userID,
		FirstName:    p.FirstName,
		BirthDate:    p.BirthDate.Format(dateLayout),
		Age:          AgeAt(p.BirthDate, s.now()),
		Gender:       p.Gender,
		Bio:          p.Bio,
		City:         p.City,
		HasLocation:  p.Latitude != nil,
		Discoverable: p.Discoverable,
		ShowDistance: p.ShowDistance,
		Interests:    nonNilInterests(interests[userID]),
		Photos:       nonNilPhotos(photos[userID]),
		Preferences:  prefs,
	}
	own.Complete = prefs != nil && len(own.Photos) > 0
	return own, nil
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (Own, error) {
	in, err := s.validateProfile(req)
	if err != nil {
		return Own{}, err
	}
	if err := s.repo.UpsertProfile(ctx, userID, in); err != nil {
		switch {
		case errors.Is(err, ErrUnknownInterest):
			return Own{}, invalid("unknown interest")
		case errors.Is(err, ErrBirthDateChanged):
			return Own{}, invalid("birth date cannot be changed")
		}
		return Own{}, err
	}
	return s.Own(ctx, userID)
}

func (s *Service) validateProfile(req UpsertProfileRequest) (ProfileInput, error) {
	name := cleanText(req.FirstName, false)
	if n := utf8.RuneCountInString(name); n < 1 || n > maxFirstNameRunes {
		return ProfileInput{}, invalid("first name must be 1 to 40 characters")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' && r != '’' {
			return ProfileInput{}, invalid("first name contains invalid characters")
		}
	}
	birth, err := time.Parse(dateLayout, strings.TrimSpace(req.BirthDate))
	if err != nil {
		return ProfileInput{}, invalid("birth date must use YYYY-MM-DD")
	}
	age := AgeAt(birth, s.now())
	if age < MinAge {
		return ProfileInput{}, invalid("you must be at least 18 years old")
	}
	if age > 120 {
		return ProfileInput{}, invalid("invalid birth date")
	}
	if _, ok := validGenders[req.Gender]; !ok {
		return ProfileInput{}, invalid("invalid gender")
	}
	bio := cleanText(req.Bio, true)
	if utf8.RuneCountInString(bio) > maxBioRunes {
		return ProfileInput{}, invalid("bio must be at most 500 characters")
	}
	city := cleanText(req.City, false)
	if utf8.RuneCountInString(city) > maxCityRunes {
		return ProfileInput{}, invalid("city must be at most 80 characters")
	}
	slugs, err := normalizeSlugs(req.Interests)
	if err != nil {
		return ProfileInput{}, err
	}
	discoverable, showDistance := true, true
	if req.Discoverable != nil {
		discoverable = *req.Discoverable
	}
	if req.ShowDistance != nil {
		showDistance = *req.ShowDistance
	}
	return ProfileInput{
		FirstName: name, BirthDate: birth, Gender: req.Gender, Bio: bio, City: city,
		InterestSlugs: slugs, Discoverable: discoverable, ShowDistance: showDistance,
	}, nil
}

func normalizeSlugs(raw []string) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		slug := strings.ToLower(strings.TrimSpace(s))
		if slug == "" || len(slug) > 40 {
			return nil, invalid("invalid interest")
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, slug)
	}
	if len(out) > MaxInterests {
		return nil, invalid("at most 10 interests")
	}
	return out, nil
}

func (s *Service) SetPreferences(ctx context.Context, userID string, p Preferences) (Preferences, error) {
	if _, err := s.repo.GetProfile(ctx, userID); err != nil {
		return Preferences{}, err
	}
	if p.AgeMin < MinAge || p.AgeMax > MaxAge || p.AgeMin > p.AgeMax {
		return Preferences{}, invalid("age range must be within 18-99 and min <= max")
	}
	if p.MaxDistanceKm != nil && (*p.MaxDistanceKm < 1 || *p.MaxDistanceKm > 20000) {
		return Preferences{}, invalid("max distance must be between 1 and 20000 km")
	}
	if n := len(p.InterestedIn); n < 1 || n > 3 {
		return Preferences{}, invalid("choose at least one gender")
	}
	seen := map[string]struct{}{}
	genders := make([]string, 0, 3)
	for _, g := range p.InterestedIn {
		if _, ok := validGenders[g]; !ok {
			return Preferences{}, invalid("invalid gender")
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			genders = append(genders, g)
		}
	}
	sort.Strings(genders)
	p.InterestedIn = genders
	if err := s.repo.UpsertPreferences(ctx, userID, p); err != nil {
		return Preferences{}, err
	}
	return p, nil
}

// SetLocation stores only a coarsened position.
func (s *Service) SetLocation(ctx context.Context, userID string, lat, lon float64) error {
	if !geo.ValidCoordinates(lat, lon) {
		return invalid("invalid coordinates")
	}
	cl, cn := geo.Coarsen(lat), geo.Coarsen(lon)
	return s.repo.SetLocation(ctx, userID, &cl, &cn)
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return s.repo.SetLocation(ctx, userID, nil, nil)
}

// Card returns the public profile of target as seen by viewer, or ErrNotFound when not visible.
func (s *Service) Card(ctx context.Context, viewerID, targetID string) (Card, error) {
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return Card{}, err
	}
	c, ok := cards[targetID]
	if !ok {
		return Card{}, ErrNotFound
	}
	return c, nil
}

// Cards loads many visible public profiles with a constant number of queries.
func (s *Service) Cards(ctx context.Context, viewerID string, ids []string) (map[string]Card, error) {
	out := make(map[string]Card, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	stored, err := s.repo.CardProfiles(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return out, nil
	}
	visible := make([]string, len(stored))
	for i, p := range stored {
		visible[i] = p.UserID
	}
	interests, err := s.repo.InterestsForUsers(ctx, visible)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.PhotosForUsers(ctx, visible)
	if err != nil {
		return nil, err
	}
	viewer, err := s.repo.GetProfile(ctx, viewerID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	now := s.now()
	for _, p := range stored {
		card := Card{
			UserID:    p.UserID,
			FirstName: p.FirstName,
			Age:       AgeAt(p.BirthDate, now),
			Gender:    p.Gender,
			Bio:       p.Bio,
			City:      p.City,
			Interests: nonNilInterests(interests[p.UserID]),
			Photos:    nonNilPhotos(photos[p.UserID]),
		}
		if p.UserID != viewerID && p.ShowDistance && viewer.Latitude != nil && p.Latitude != nil {
			d := geo.BucketKm(geo.Haversine(*viewer.Latitude, *viewer.Longitude, *p.Latitude, *p.Longitude))
			card.DistanceKm = &d
		}
		out[p.UserID] = card
	}
	return out, nil
}

// AgeAt computes the full years elapsed between birth and now.
func AgeAt(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// cleanText trims, drops control characters (keeping newlines when asked) and collapses blank runs.
func cleanText(v string, keepNewlines bool) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(v) {
		switch {
		case r == '\n' && keepNewlines:
			b.WriteRune(r)
		case r == '\n' || r == '\t' || r == '\r':
			b.WriteRune(' ')
		case unicode.IsControl(r):
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	if keepNewlines {
		for strings.Contains(out, "\n\n\n") {
			out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
		}
	}
	return strings.TrimSpace(out)
}

func nonNilInterests(v []Interest) []Interest {
	if v == nil {
		return []Interest{}
	}
	return v
}

func nonNilPhotos(v []PhotoRef) []PhotoRef {
	if v == nil {
		return []PhotoRef{}
	}
	return v
}
