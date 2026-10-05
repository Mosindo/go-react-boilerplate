package profiles

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalidName        = errors.New("first name must be 1-40 letters")
	ErrInvalidBirthDate   = errors.New("birth date must be YYYY-MM-DD")
	ErrUnderage           = errors.New("you must be at least 18 years old")
	ErrBirthDateLocked    = errors.New("birth date cannot be changed")
	ErrInvalidGender      = errors.New("invalid gender")
	ErrInvalidBio         = errors.New("bio must be at most 500 characters")
	ErrInvalidOccupation  = errors.New("occupation must be at most 60 characters")
	ErrTooManyInterests   = errors.New("at most 10 interests")
	ErrInvalidPreferences = errors.New("invalid preferences")
	ErrInvalidLocation    = errors.New("invalid location")
	ErrProfileRequired    = errors.New("create your profile first")
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// RoundDistanceKm hides exact positions: distances are rounded up to the next 5 km step (minimum 5).
func RoundDistanceKm(km float64) int {
	if km <= 5 {
		return 5
	}
	return int(math.Ceil(km/5)) * 5
}

// roundCoordinate keeps ~1.1 km precision, which is all the product needs and avoids storing exact positions.
func roundCoordinate(v float64) float64 { return math.Round(v*100) / 100 }

func (s *Service) Mine(ctx context.Context, userID string) (MyProfileResponse, error) {
	resp := MyProfileResponse{}
	stored, err := s.repo.GetProfile(ctx, userID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return resp, err
	}
	hasProfile := err == nil

	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return resp, err
	}
	resp.Preferences = prefs
	photos, err := s.repo.CountPhotos(ctx, userID)
	if err != nil {
		return resp, err
	}
	resp.Completion = Completion{
		Profile:     hasProfile,
		Preferences: prefs != nil,
		Location:    hasProfile && stored.HasLocation,
		Photos:      photos > 0,
	}
	resp.Completion.Complete = resp.Completion.Profile && resp.Completion.Preferences &&
		resp.Completion.Location && resp.Completion.Photos

	if !hasProfile {
		return resp, nil
	}
	cards, err := s.repo.Cards(ctx, userID, []string{userID})
	if err != nil {
		return resp, err
	}
	card := cards[userID]
	resp.Profile = &OwnProfile{
		ID:             stored.UserID,
		FirstName:      stored.FirstName,
		BirthDate:      stored.BirthDate.Format("2006-01-02"),
		Age:            stored.Age,
		Gender:         stored.Gender,
		Bio:            stored.Bio,
		Occupation:     stored.Occupation,
		City:           stored.City,
		HasLocation:    stored.HasLocation,
		IsDiscoverable: stored.IsDiscoverable,
		ShowDistance:   stored.ShowDistance,
		Interests:      emptyIfNil(card.Interests),
		Photos:         emptyPhotos(card.Photos),
		CreatedAt:      stored.CreatedAt,
	}
	return resp, nil
}

// IsComplete reports whether the user finished onboarding (profile, preferences, location, one photo).
func (s *Service) IsComplete(ctx context.Context, userID string) (bool, error) {
	mine, err := s.Mine(ctx, userID)
	return mine.Completion.Complete, err
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) error {
	in, err := s.validateProfile(req)
	if err != nil {
		return err
	}
	if existing, err := s.repo.GetProfile(ctx, userID); err == nil {
		if !existing.BirthDate.Equal(in.BirthDate) {
			return ErrBirthDateLocked
		}
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	return s.repo.UpsertProfile(ctx, userID, in)
}

func (s *Service) validateProfile(req UpsertProfileRequest) (ProfileInput, error) {
	name := strings.TrimSpace(req.FirstName)
	if n := utf8.RuneCountInString(name); n < 1 || n > 40 {
		return ProfileInput{}, ErrInvalidName
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' && r != '’' && r != '.' {
			return ProfileInput{}, ErrInvalidName
		}
	}

	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return ProfileInput{}, ErrInvalidBirthDate
	}
	age := AgeOn(birth, s.now())
	if age < MinAge {
		return ProfileInput{}, ErrUnderage
	}
	if age > MaxAge {
		return ProfileInput{}, ErrInvalidBirthDate
	}

	if !contains(Genders, req.Gender) {
		return ProfileInput{}, ErrInvalidGender
	}

	bio := cleanText(req.Bio)
	if utf8.RuneCountInString(bio) > 500 {
		return ProfileInput{}, ErrInvalidBio
	}
	occupation := strings.Join(strings.Fields(cleanText(req.Occupation)), " ")
	if utf8.RuneCountInString(occupation) > 60 {
		return ProfileInput{}, ErrInvalidOccupation
	}

	interests := uniqueStrings(req.Interests)
	if len(interests) > MaxInterests {
		return ProfileInput{}, ErrTooManyInterests
	}

	return ProfileInput{FirstName: name, BirthDate: birth, Gender: req.Gender, Bio: bio, Occupation: occupation, Interests: interests}, nil
}

// AgeOn returns the age in full years of someone born on birth at the given date.
func AgeOn(birth, at time.Time) int {
	years := at.Year() - birth.Year()
	if at.Month() < birth.Month() || (at.Month() == birth.Month() && at.Day() < birth.Day()) {
		years--
	}
	return years
}

func (s *Service) SetPreferences(ctx context.Context, userID string, req PreferencesRequest) error {
	genders := uniqueStrings(req.InterestedIn)
	if len(genders) == 0 {
		return ErrInvalidPreferences
	}
	for _, g := range genders {
		if !contains(Genders, g) {
			return ErrInvalidPreferences
		}
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge ||
		req.MaxDistanceKm < 1 || req.MaxDistanceKm > MaxDistanceKm {
		return ErrInvalidPreferences
	}
	return s.repo.UpsertPreferences(ctx, userID, Preferences{
		InterestedIn: genders, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm,
	})
}

func (s *Service) SetLocation(ctx context.Context, userID string, req LocationRequest) error {
	if req.Latitude == nil || req.Longitude == nil {
		return ErrInvalidLocation
	}
	lat, lng := *req.Latitude, *req.Longitude
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return ErrInvalidLocation
	}
	city := strings.Join(strings.Fields(cleanText(req.City)), " ")
	if utf8.RuneCountInString(city) > 80 {
		city = string([]rune(city)[:80])
	}
	err := s.repo.UpdateLocation(ctx, userID, roundCoordinate(lat), roundCoordinate(lng), city)
	if errors.Is(err, ErrNotFound) {
		return ErrProfileRequired
	}
	return err
}

func (s *Service) SetPrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error {
	err := s.repo.UpdatePrivacy(ctx, userID, discoverable, showDistance)
	if errors.Is(err, ErrNotFound) {
		return ErrProfileRequired
	}
	return err
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// View returns another user's public card, enforcing block and visibility rules.
func (s *Service) View(ctx context.Context, viewerID, targetID string) (Card, error) {
	if viewerID == targetID {
		cards, err := s.repo.Cards(ctx, viewerID, []string{viewerID})
		if err != nil {
			return Card{}, err
		}
		if c, ok := cards[viewerID]; ok {
			return c, nil
		}
		return Card{}, ErrNotFound
	}
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return Card{}, err
	}
	if !ok {
		return Card{}, ErrNotFound
	}
	cards, err := s.repo.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return Card{}, err
	}
	c, found := cards[targetID]
	if !found {
		return Card{}, ErrNotFound
	}
	return c, nil
}

// Cards exposes the batched card loader to other features (discovery, matches, chat).
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]Card, error) {
	return s.repo.Cards(ctx, viewerID, userIDs)
}

func cleanText(v string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(v) {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func emptyIfNil(in []Interest) []Interest {
	if in == nil {
		return []Interest{}
	}
	return in
}

func emptyPhotos(in []Photo) []Photo {
	if in == nil {
		return []Photo{}
	}
	return in
}
