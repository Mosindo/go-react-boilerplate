package profiles

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/mediaurl"
)

var (
	ErrNotFound          = errors.New("profile not found")
	ErrValidation        = errors.New("validation failed")
	ErrBirthDateLocked   = errors.New("birth date cannot be changed")
	ErrProfileMissing    = errors.New("create your profile first")
	ErrIncompleteProfile = errors.New("onboarding is incomplete")
)

// ValidationError carries a user-facing message and wraps ErrValidation.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }
func (e *ValidationError) Unwrap() error { return ErrValidation }

func invalid(msg string) error { return &ValidationError{Message: msg} }

type Service struct {
	repo   Repository
	signer *mediaurl.Signer
	now    func() time.Time
}

func NewService(repo Repository, signer *mediaurl.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) photos(ids []string) []Photo {
	out := make([]Photo, 0, len(ids))
	for _, id := range ids {
		out = append(out, Photo{ID: id, URL: s.signer.Path(id, "full"), ThumbURL: s.signer.Path(id, "thumb")})
	}
	return out
}

// Me returns the owner's profile together with onboarding progress.
func (s *Service) Me(ctx context.Context, userID string) (MyProfile, error) {
	res := MyProfile{Interests: []Interest{}, Photos: []Photo{}, Missing: []string{}}

	stored, err := s.repo.GetOwn(ctx, userID)
	switch {
	case err == nil:
		res.Profile = &Own{
			FirstName: stored.FirstName, BirthDate: stored.BirthDate.Format("2006-01-02"), Age: stored.Age,
			Gender: stored.Gender, Bio: stored.Bio, Occupation: stored.Occupation, City: stored.City,
			HasLocation: stored.HasLocation, IsVisible: stored.IsVisible, ShowDistance: stored.ShowDistance,
		}
	case errors.Is(err, ErrRepositoryNotFound):
		res.Missing = append(res.Missing, "profile")
	default:
		return res, err
	}

	if res.Profile != nil {
		if res.Interests, err = s.repo.InterestsOf(ctx, userID); err != nil {
			return res, err
		}
	}
	photoIDs, err := s.repo.PhotoIDs(ctx, []string{userID})
	if err != nil {
		return res, err
	}
	res.Photos = s.photos(photoIDs[userID])
	if len(res.Photos) < MinPhotosNeeded {
		res.Missing = append(res.Missing, "photos")
	}
	if res.Preferences, err = s.repo.GetPreferences(ctx, userID); err != nil {
		return res, err
	}
	if res.Preferences == nil || len(res.Preferences.InterestedIn) == 0 {
		res.Missing = append(res.Missing, "preferences")
	}
	res.OnboardingComplete = len(res.Missing) == 0
	return res, nil
}

// IsOnboarded reports whether the user can take part in discovery.
func (s *Service) IsOnboarded(ctx context.Context, userID string) (bool, error) {
	me, err := s.Me(ctx, userID)
	return me.OnboardingComplete, err
}

func ageOn(birth, now time.Time) int {
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return age
}

func cleanText(raw string, maxRunes int, field string, allowNewlines bool) (string, error) {
	v := strings.TrimSpace(raw)
	if !utf8.ValidString(v) {
		return "", invalid(field + " is not valid text")
	}
	for _, r := range v {
		if unicode.IsControl(r) && !(allowNewlines && (r == '\n' || r == '\r' || r == '\t')) {
			return "", invalid(field + " contains invalid characters")
		}
	}
	if utf8.RuneCountInString(v) > maxRunes {
		return "", invalid(field + " is too long")
	}
	return v, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) error {
	name, err := cleanText(req.FirstName, MaxNameRunes, "first name", false)
	if err != nil {
		return err
	}
	if name == "" {
		return invalid("first name is required")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsMark(r) && r != ' ' && r != '-' && r != '\'' && r != '’' {
			return invalid("first name contains invalid characters")
		}
	}
	if !slices.Contains(Genders, req.Gender) {
		return invalid("gender is invalid")
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return invalid("birth date must use the YYYY-MM-DD format")
	}
	age := ageOn(birth, s.now().UTC())
	if age < MinAge {
		return invalid("you must be at least 18 years old")
	}
	if age > MaxAge {
		return invalid("birth date is not valid")
	}
	bio, err := cleanText(req.Bio, MaxBioRunes, "bio", true)
	if err != nil {
		return err
	}
	occupation, err := cleanText(req.Occupation, MaxShortRunes, "occupation", false)
	if err != nil {
		return err
	}
	city, err := cleanText(req.City, MaxShortRunes, "city", false)
	if err != nil {
		return err
	}
	slugs := dedupe(req.Interests)
	if len(slugs) > MaxInterests {
		return invalid("too many interests")
	}

	// The birth date is locked once set: otherwise the age gate could be bypassed
	// by editing it later.
	if existing, err := s.repo.GetOwn(ctx, userID); err == nil {
		if !existing.BirthDate.Equal(birth) {
			return ErrBirthDateLocked
		}
	} else if !errors.Is(err, ErrRepositoryNotFound) {
		return err
	}

	err = s.repo.UpsertProfile(ctx, userID, ProfileInput{
		FirstName: name, BirthDate: birth, Gender: req.Gender, Bio: bio, Occupation: occupation, City: city,
	}, slugs)
	if errors.Is(err, ErrRepositoryUnknownInterest) {
		return invalid("unknown interest")
	}
	return err
}

func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.ToLower(strings.TrimSpace(v))
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

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) error {
	genders := dedupe(req.InterestedIn)
	if len(genders) == 0 {
		return invalid("choose at least one gender")
	}
	for _, g := range genders {
		if !slices.Contains(Genders, g) {
			return invalid("gender is invalid")
		}
	}
	if req.AgeMin < MinAge || req.AgeMax > MaxAge || req.AgeMin > req.AgeMax {
		return invalid("age range must be within 18-99 and min <= max")
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > MaxDistanceKm {
		return invalid("distance is out of range")
	}
	return s.repo.UpsertPreferences(ctx, userID, Preferences{
		InterestedIn: genders, AgeMin: req.AgeMin, AgeMax: req.AgeMax, MaxDistanceKm: req.MaxDistanceKm,
	})
}

// RoundCoordinate reduces precision to 0.01 degree (~1 km): the exact position is never stored.
func RoundCoordinate(v float64) float64 { return math.Round(v*100) / 100 }

func (s *Service) UpdateLocation(ctx context.Context, userID string, req UpdateLocationRequest) error {
	if math.IsNaN(req.Latitude) || math.IsNaN(req.Longitude) ||
		req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		return invalid("coordinates are out of range")
	}
	var city *string
	if req.City != nil {
		c, err := cleanText(*req.City, MaxShortRunes, "city", false)
		if err != nil {
			return err
		}
		city = &c
	}
	err := s.repo.SetLocation(ctx, userID, RoundCoordinate(req.Latitude), RoundCoordinate(req.Longitude), city)
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrProfileMissing
	}
	return err
}

func (s *Service) UpdatePrivacy(ctx context.Context, userID string, req UpdatePrivacyRequest) error {
	err := s.repo.SetPrivacy(ctx, userID, req.IsVisible, req.ShowDistance)
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrProfileMissing
	}
	return err
}

// DistanceBucket turns an exact distance into a coarse, privacy-preserving figure:
// 1, 2, 5, 10, 25, 50, 100, 250, 500 km ("less than N km").
func DistanceBucket(km float64) int {
	for _, b := range []int{1, 2, 5, 10, 25, 50, 100, 250, 500} {
		if km <= float64(b) {
			return b
		}
	}
	return 1000
}

// Cards returns the public cards for the given users, preserving the input order.
// Users without a (visible, complete) profile are skipped.
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) ([]Card, error) {
	data, err := s.repo.Cards(ctx, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	out := make([]Card, 0, len(userIDs))
	for _, id := range userIDs {
		d, ok := data[id]
		if !ok {
			continue
		}
		out = append(out, s.toCard(d))
	}
	return out, nil
}

func (s *Service) toCard(d *CardData) Card {
	var dist *int
	if d.Profile.DistanceKm != nil {
		b := DistanceBucket(*d.Profile.DistanceKm)
		dist = &b
	}
	return Card{
		UserID: d.Profile.UserID, FirstName: d.Profile.FirstName, Age: d.Profile.Age, Gender: d.Profile.Gender,
		Bio: d.Profile.Bio, Occupation: d.Profile.Occupation, City: d.Profile.City, DistanceKm: dist,
		Interests: d.Interests, Photos: s.photos(d.PhotoIDs),
	}
}

// Card returns a single card or ErrNotFound.
func (s *Service) Card(ctx context.Context, viewerID, userID string) (Card, error) {
	cards, err := s.Cards(ctx, viewerID, []string{userID})
	if err != nil {
		return Card{}, err
	}
	if len(cards) == 0 {
		return Card{}, ErrNotFound
	}
	return cards[0], nil
}

// ViewableCard returns the card only if the viewer is allowed to see it.
func (s *Service) ViewableCard(ctx context.Context, viewerID, targetID string) (Card, error) {
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return Card{}, err
	}
	if !ok {
		return Card{}, ErrNotFound
	}
	return s.Card(ctx, viewerID, targetID)
}

// CanView exposes the visibility rule to other features (swipes, reports).
func (s *Service) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	return s.repo.CanView(ctx, viewerID, targetID)
}
