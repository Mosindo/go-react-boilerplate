package profiles

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/media"
)

const (
	MinAge           = 18
	maxAge           = 100
	maxFirstName     = 40
	maxBio           = 500
	maxCity          = 80
	maxInterests     = 10
	locationDecimals = 100.0 // 2 decimals ≈ 1.1 km
)

var (
	ErrProfileNotFound = platformerrors.NotFound("profile not found")
	errUnderage        = platformerrors.New(400, platformerrors.CodeUnderage, "you must be at least 18 years old")
)

type Service struct {
	repo   *Repository
	signer *media.Signer
	now    func() time.Time
}

func NewService(repo *Repository, signer *media.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

// AgeOn returns the full years between birth and the reference day.
func AgeOn(birth, ref time.Time) int {
	years := ref.Year() - birth.Year()
	if ref.Month() < birth.Month() || (ref.Month() == birth.Month() && ref.Day() < birth.Day()) {
		years--
	}
	return years
}

func (s *Service) GetSelf(ctx context.Context, userID string) (SelfProfile, error) {
	stored, err := s.repo.Get(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return SelfProfile{}, ErrProfileNotFound
		}
		return SelfProfile{}, err
	}
	interests, err := s.repo.InterestsByUsers(ctx, []string{userID})
	if err != nil {
		return SelfProfile{}, err
	}
	photoRows, err := s.repo.PhotosByUsers(ctx, []string{userID})
	if err != nil {
		return SelfProfile{}, err
	}
	photos := s.photos(photoRows[userID])
	profile := SelfProfile{
		UserID:      stored.UserID,
		FirstName:   stored.FirstName,
		BirthDate:   stored.BirthDate.Format("2006-01-02"),
		Age:         AgeOn(stored.BirthDate, s.now().UTC()),
		Gender:      stored.Gender,
		Bio:         stored.Bio,
		City:        stored.City,
		HasLocation: stored.HasLocation,
		Interests:   interests[userID],
		Photos:      photos,
		Complete:    len(photos) > 0,
	}
	if profile.Interests == nil {
		profile.Interests = []Interest{}
	}
	return profile, nil
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (SelfProfile, error) {
	firstName := strings.TrimSpace(req.FirstName)
	if err := validateFirstName(firstName); err != nil {
		return SelfProfile{}, err
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return SelfProfile{}, platformerrors.Validation("birthDate must be formatted YYYY-MM-DD")
	}
	age := AgeOn(birth, s.now().UTC())
	if age < MinAge {
		return SelfProfile{}, errUnderage
	}
	if age > maxAge {
		return SelfProfile{}, platformerrors.Validation("birthDate is not plausible")
	}
	gender := strings.ToLower(strings.TrimSpace(req.Gender))
	if _, ok := validGenders[gender]; !ok {
		return SelfProfile{}, platformerrors.Validation("gender must be man, woman or nonbinary")
	}
	bio := strings.TrimSpace(req.Bio)
	if utf8.RuneCountInString(bio) > maxBio || hasControl(bio, true) {
		return SelfProfile{}, platformerrors.Validation("bio must be at most 500 characters")
	}
	city := strings.TrimSpace(req.City)
	if utf8.RuneCountInString(city) > maxCity || hasControl(city, false) {
		return SelfProfile{}, platformerrors.Validation("city must be at most 80 characters")
	}
	slugs, err := s.validateInterests(ctx, req.Interests)
	if err != nil {
		return SelfProfile{}, err
	}

	if err := s.repo.Upsert(ctx, userID, firstName, birth, gender, bio, city, slugs); err != nil {
		return SelfProfile{}, err
	}
	return s.GetSelf(ctx, userID)
}

func (s *Service) validateInterests(ctx context.Context, raw []string) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	slugs := make([]string, 0, len(raw))
	for _, slug := range raw {
		slug = strings.ToLower(strings.TrimSpace(slug))
		if slug == "" {
			continue
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}
	if len(slugs) > maxInterests {
		return nil, platformerrors.Validation("choose at most 10 interests")
	}
	if len(slugs) == 0 {
		return slugs, nil
	}
	known, err := s.repo.CountKnownInterests(ctx, slugs)
	if err != nil {
		return nil, err
	}
	if known != len(slugs) {
		return nil, platformerrors.Validation("unknown interest")
	}
	return slugs, nil
}

func (s *Service) SetLocation(ctx context.Context, userID string, lat, lng float64) error {
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) ||
		lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return platformerrors.Validation("invalid coordinates")
	}
	// Only a ~1 km approximation is ever persisted.
	lat = math.Round(lat*locationDecimals) / locationDecimals
	lng = math.Round(lng*locationDecimals) / locationDecimals
	if err := s.repo.SetLocation(ctx, userID, lat, lng); err != nil {
		if errors.Is(err, ErrNotFound) {
			return platformerrors.New(403, platformerrors.CodeProfileIncomplete, "create your profile first")
		}
		return err
	}
	return nil
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	prefs, found, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return Preferences{}, err
	}
	if !found {
		return DefaultPreferences(), nil
	}
	return prefs, nil
}

func (s *Service) SavePreferences(ctx context.Context, userID string, p Preferences) (Preferences, error) {
	seen := map[string]struct{}{}
	genders := make([]string, 0, 3)
	for _, g := range p.InterestedIn {
		g = strings.ToLower(strings.TrimSpace(g))
		if _, ok := validGenders[g]; !ok {
			return Preferences{}, platformerrors.Validation("interestedIn contains an invalid gender")
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			genders = append(genders, g)
		}
	}
	if len(genders) == 0 {
		return Preferences{}, platformerrors.Validation("choose at least one gender you want to meet")
	}
	if p.MinAge < MinAge || p.MaxAge > 99 || p.MinAge > p.MaxAge {
		return Preferences{}, platformerrors.Validation("age range must be between 18 and 99 with minAge <= maxAge")
	}
	if p.MaxDistanceKm < 1 || p.MaxDistanceKm > 500 {
		return Preferences{}, platformerrors.Validation("maxDistanceKm must be between 1 and 500")
	}
	p.InterestedIn = genders
	if err := s.repo.SavePreferences(ctx, userID, p); err != nil {
		return Preferences{}, err
	}
	return p, nil
}

// ---- shared readers used by other features ----

func (s *Service) photos(rows []PhotoRow) []media.Photo {
	out := make([]media.Photo, 0, len(rows))
	for _, row := range rows {
		out = append(out, media.Photo{ID: row.ID, URL: s.signer.URL(row.ID), Position: row.Position})
	}
	return out
}

// Photos returns signed photos grouped by user id.
func (s *Service) Photos(ctx context.Context, userIDs []string) (map[string][]media.Photo, error) {
	rows, err := s.repo.PhotosByUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]media.Photo, len(rows))
	for uid, list := range rows {
		out[uid] = s.photos(list)
	}
	return out, nil
}

func (s *Service) InterestsByUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	return s.repo.InterestsByUsers(ctx, userIDs)
}

// Summaries returns minimal public identities (first name, age, main photo) for the given users.
func (s *Service) Summaries(ctx context.Context, userIDs []string) (map[string]UserSummary, error) {
	rows, err := s.repo.SummaryRows(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	photos, err := s.Photos(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]UserSummary, len(rows))
	for _, row := range rows {
		summary := UserSummary{UserID: row.UserID, FirstName: row.FirstName, Age: row.Age}
		if list := photos[row.UserID]; len(list) > 0 {
			first := list[0]
			summary.Photo = &first
		}
		out[row.UserID] = summary
	}
	return out, nil
}

func validateFirstName(name string) error {
	if name == "" || utf8.RuneCountInString(name) > maxFirstName {
		return platformerrors.Validation("firstName must be 1 to 40 characters")
	}
	hasLetter := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case r == ' ' || r == '-' || r == '\'' || r == '’' || r == '.':
		default:
			return platformerrors.Validation("firstName may only contain letters, spaces, hyphens and apostrophes")
		}
	}
	if !hasLetter {
		return platformerrors.Validation("firstName must contain a letter")
	}
	return nil
}

func hasControl(s string, allowNewline bool) bool {
	for _, r := range s {
		if r == '\n' && allowNewline {
			continue
		}
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
