package profiles

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrNotFound         = errors.New("profile not found")
	ErrProfileRequired  = errors.New("create your profile first")
	ErrBirthDateLocked  = errors.New("birth date cannot be changed")
	errValidationMarker = errors.New("validation")
)

// ValidationError carries a message that is safe to show to the user.
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }
func (e ValidationError) Is(target error) bool {
	return target == errValidationMarker
}

func invalid(format string, args ...any) error {
	return ValidationError{Message: fmt.Sprintf(format, args...)}
}

// IsValidation reports whether err is a user-facing validation failure.
func IsValidation(err error) bool { return errors.Is(err, errValidationMarker) }

type PhotoURLFunc func(photoID string) string

type Service struct {
	repo     Repository
	photoURL PhotoURLFunc
	now      func() time.Time

	catalogMu sync.Mutex
	catalog   []Interest
	labels    map[string]string
}

func NewService(repo Repository, photoURL PhotoURLFunc) *Service {
	return &Service{repo: repo, photoURL: photoURL, now: time.Now}
}

// AgeOn returns the completed years between birth and the given day.
func AgeOn(birth, today time.Time) int {
	age := today.Year() - birth.Year()
	if today.Month() < birth.Month() || (today.Month() == birth.Month() && today.Day() < birth.Day()) {
		age--
	}
	return age
}

// ApproxDistance turns an exact distance into a deliberately blurry one: 1 km
// resolution below 10 km, 5 km resolution above, never below 1.
func ApproxDistance(km float64) int {
	if km <= 1 {
		return 1
	}
	if km <= 10 {
		return int(math.Ceil(km))
	}
	return int(math.Ceil(km/5.0)) * 5
}

// RoundCoordinate keeps two decimals (~1.1 km), which is all matching needs.
func RoundCoordinate(v float64) float64 {
	return math.Round(v*100) / 100
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()
	if s.catalog != nil {
		return s.catalog, nil
	}
	list, err := s.repo.ListInterests(ctx)
	if err != nil {
		return nil, err
	}
	labels := make(map[string]string, len(list))
	for _, i := range list {
		labels[i.Slug] = i.Label
	}
	s.catalog, s.labels = list, labels
	return list, nil
}

func (s *Service) label(slug string) (string, bool) {
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()
	l, ok := s.labels[slug]
	return l, ok
}

func (s *Service) photoRefs(ids []string) []PhotoRef {
	refs := make([]PhotoRef, 0, len(ids))
	for _, id := range ids {
		refs = append(refs, PhotoRef{ID: id, URL: s.photoURL(id)})
	}
	return refs
}

func (s *Service) Me(ctx context.Context, userID string) (MeProfileResponse, error) {
	resp := MeProfileResponse{Photos: []PhotoRef{}, Missing: []string{}}

	photoIDs, err := s.repo.PhotoIDs(ctx, userID)
	if err != nil {
		return resp, err
	}
	resp.Photos = s.photoRefs(photoIDs)

	stored, err := s.repo.GetProfile(ctx, userID)
	switch {
	case err == nil:
		resp.Profile = ownProfile(stored, s.now())
	case errors.Is(err, ErrRepositoryNotFound):
		resp.Missing = append(resp.Missing, "profile")
	default:
		return resp, err
	}

	prefs, err := s.repo.GetPreferences(ctx, userID)
	switch {
	case err == nil:
		resp.Preferences = &PreferencesResponse{InterestedIn: prefs.InterestedIn, MinAge: prefs.MinAge, MaxAge: prefs.MaxAge, MaxDistanceKm: prefs.MaxDistanceKm}
	case errors.Is(err, ErrRepositoryNotFound):
		resp.Missing = append(resp.Missing, "preferences")
	default:
		return resp, err
	}

	if len(photoIDs) == 0 {
		resp.Missing = append(resp.Missing, "photos")
	}
	resp.IsComplete = len(resp.Missing) == 0
	return resp, nil
}

func ownProfile(p StoredProfile, now time.Time) *OwnProfile {
	interests := p.Interests
	if interests == nil {
		interests = []string{}
	}
	return &OwnProfile{
		FirstName:    p.FirstName,
		BirthDate:    p.BirthDate.Format("2006-01-02"),
		Age:          AgeOn(p.BirthDate, now),
		Gender:       p.Gender,
		Bio:          p.Bio,
		Occupation:   p.Occupation,
		LookingFor:   p.LookingFor,
		City:         p.City,
		Latitude:     p.Latitude,
		Longitude:    p.Longitude,
		Discoverable: p.Discoverable,
		ShowDistance: p.ShowDistance,
		Interests:    interests,
	}
}

// SaveProfile validates every field server-side and creates or updates the profile.
func (s *Service) SaveProfile(ctx context.Context, userID string, in ProfileInput) error {
	today := s.now().UTC()

	firstName, err := cleanText(in.FirstName, "first name", 1, 40, false)
	if err != nil {
		return err
	}
	bio, err := cleanText(in.Bio, "bio", 0, 500, true)
	if err != nil {
		return err
	}
	occupation, err := cleanText(in.Occupation, "occupation", 0, 80, false)
	if err != nil {
		return err
	}
	city, err := cleanText(in.City, "city", 0, 80, false)
	if err != nil {
		return err
	}
	if _, ok := genders[in.Gender]; !ok {
		return invalid("gender must be woman, man or nonbinary")
	}
	lookingFor := in.LookingFor
	if lookingFor == "" {
		lookingFor = "open"
	}
	if _, ok := lookingForValues[lookingFor]; !ok {
		return invalid("lookingFor must be relationship, casual, friendship or open")
	}

	birth, err := time.Parse("2006-01-02", strings.TrimSpace(in.BirthDate))
	if err != nil {
		return invalid("birth date must use the YYYY-MM-DD format")
	}
	if age := AgeOn(birth, today); age < MinAge {
		return invalid("you must be at least %d years old", MinAge)
	} else if age > 120 {
		return invalid("birth date is not valid")
	}

	if math.IsNaN(in.Latitude) || math.IsNaN(in.Longitude) ||
		in.Latitude < -90 || in.Latitude > 90 || in.Longitude < -180 || in.Longitude > 180 {
		return invalid("location is not valid")
	}

	interests, err := s.validateInterests(ctx, in.Interests)
	if err != nil {
		return err
	}

	existing, err := s.repo.GetProfile(ctx, userID)
	switch {
	case err == nil:
		// The age gate is only meaningful if the date of birth cannot be edited afterwards.
		if !existing.BirthDate.Equal(birth) {
			return ErrBirthDateLocked
		}
	case !errors.Is(err, ErrRepositoryNotFound):
		return err
	}

	return s.repo.UpsertProfile(ctx, StoredProfile{
		UserID:     userID,
		FirstName:  firstName,
		BirthDate:  birth,
		Gender:     in.Gender,
		Bio:        bio,
		Occupation: occupation,
		LookingFor: lookingFor,
		City:       city,
		Latitude:   RoundCoordinate(in.Latitude),
		Longitude:  RoundCoordinate(in.Longitude),
		Interests:  interests,
	})
}

func (s *Service) validateInterests(ctx context.Context, slugs []string) ([]string, error) {
	if len(slugs) > 10 {
		return nil, invalid("choose at most 10 interests")
	}
	if _, err := s.Interests(ctx); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(slugs))
	out := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		if _, ok := s.label(slug); !ok {
			return nil, invalid("unknown interest %q", slug)
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, slug)
	}
	return out, nil
}

func (s *Service) SavePreferences(ctx context.Context, userID string, in PreferencesInput) error {
	seen := map[string]struct{}{}
	interestedIn := make([]string, 0, 3)
	for _, g := range in.InterestedIn {
		if _, ok := genders[g]; !ok {
			return invalid("interestedIn must only contain woman, man or nonbinary")
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			interestedIn = append(interestedIn, g)
		}
	}
	if len(interestedIn) == 0 {
		return invalid("choose who you want to meet")
	}
	if in.MinAge < MinAge || in.MaxAge > MaxAge || in.MinAge > in.MaxAge {
		return invalid("age range must be between %d and %d", MinAge, MaxAge)
	}
	if in.MaxDistanceKm < 1 || in.MaxDistanceKm > 500 {
		return invalid("distance must be between 1 and 500 km")
	}
	return s.repo.UpsertPreferences(ctx, userID, StoredPreferences{
		InterestedIn: interestedIn, MinAge: in.MinAge, MaxAge: in.MaxAge, MaxDistanceKm: in.MaxDistanceKm,
	})
}

func (s *Service) SavePrivacy(ctx context.Context, userID string, in PrivacyInput) error {
	if in.Discoverable == nil && in.ShowDistance == nil {
		return invalid("nothing to update")
	}
	if err := s.repo.SetPrivacy(ctx, userID, in.Discoverable, in.ShowDistance); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrProfileRequired
		}
		return err
	}
	return nil
}

// Cards hydrates public profiles for many users in one round trip, preserving the
// order of ids. Unknown users are skipped silently.
func (s *Service) Cards(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error) {
	stored, err := s.repo.PublicByIDs(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	if _, err := s.Interests(ctx); err != nil {
		return nil, err
	}
	byID := make(map[string]StoredPublic, len(stored))
	for _, p := range stored {
		byID[p.UserID] = p
	}
	today := s.now()
	out := make([]PublicProfile, 0, len(ids))
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			out = append(out, s.toPublic(p, today))
		}
	}
	return out, nil
}

func (s *Service) toPublic(p StoredPublic, today time.Time) PublicProfile {
	var distance *int
	if p.DistanceKm != nil {
		d := ApproxDistance(*p.DistanceKm)
		distance = &d
	}
	interests := make([]Interest, 0, len(p.Interests))
	for _, slug := range p.Interests {
		if label, ok := s.label(slug); ok {
			interests = append(interests, Interest{Slug: slug, Label: label})
		}
	}
	return PublicProfile{
		ID:         p.UserID,
		FirstName:  p.FirstName,
		Age:        AgeOn(p.BirthDate, today),
		Gender:     p.Gender,
		Bio:        p.Bio,
		Occupation: p.Occupation,
		LookingFor: p.LookingFor,
		City:       p.City,
		DistanceKm: distance,
		Photos:     s.photoRefs(p.PhotoIDs),
		Interests:  interests,
	}
}

// View returns another user's full profile, or ErrNotFound when the viewer is not
// allowed to see it (block, hidden profile, unknown user): the cases are indistinguishable.
func (s *Service) View(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	if viewerID == targetID {
		cards, err := s.Cards(ctx, viewerID, []string{targetID})
		if err != nil || len(cards) == 0 {
			return PublicProfile{}, ErrNotFound
		}
		return cards[0], nil
	}
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if !ok {
		return PublicProfile{}, ErrNotFound
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	if len(cards) == 0 {
		return PublicProfile{}, ErrNotFound
	}
	return cards[0], nil
}

func (s *Service) Touch(ctx context.Context, userID string) error {
	return s.repo.TouchActivity(ctx, userID)
}

// cleanText trims, collapses control characters and enforces rune-length bounds.
// Newlines are only kept when allowed (bio).
func cleanText(raw, field string, min, max int, allowNewlines bool) (string, error) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r == '\n' && allowNewlines:
			b.WriteRune(r)
		case r == '\n' || r == '\t' || r == '\r':
			b.WriteRune(' ')
		case unicode.IsControl(r) || r == utf8.RuneError:
			// dropped
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	n := utf8.RuneCountInString(out)
	if n < min {
		return "", invalid("%s is required", field)
	}
	if n > max {
		return "", invalid("%s must be at most %d characters", field, max)
	}
	return out, nil
}
