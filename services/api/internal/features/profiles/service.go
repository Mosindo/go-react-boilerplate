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
)

const (
	MinAge          = 18
	MaxAge          = 99
	maxInterests    = 10
	maxFirstName    = 40
	maxBio          = 500
	maxCity         = 80
	distanceStepKm  = 5
	coordinateScale = 100 // 2 decimals ≈ 1.1 km grid
)

var (
	errNotFound      = platformerrors.NotFound("profile_not_found", "profile not found")
	errProfileHidden = platformerrors.NotFound("not_found", "profile not found")
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) Own(ctx context.Context, userID string) (OwnProfile, error) {
	p, err := s.repo.GetOwn(ctx, userID)
	if errors.Is(err, ErrRepoNotFound) {
		return OwnProfile{}, errNotFound
	}
	return p, err
}

// Save creates or updates the profile. The birth date is fixed once set (age cannot be edited later).
func (s *Service) Save(ctx context.Context, userID string, in ProfileInput) (OwnProfile, error) {
	cleaned, err := normalizeInput(in)
	if err != nil {
		return OwnProfile{}, err
	}
	birth, err := time.Parse("2006-01-02", in.BirthDate)
	if err != nil {
		return OwnProfile{}, platformerrors.BadRequest("invalid_birth_date", "birth date must be a valid date (YYYY-MM-DD)")
	}
	if age := AgeAt(birth, s.now()); age < MinAge {
		return OwnProfile{}, platformerrors.BadRequest("underage", "you must be at least 18 years old")
	} else if age > MaxAge {
		return OwnProfile{}, platformerrors.BadRequest("invalid_birth_date", "birth date is not valid")
	}

	existing, err := s.repo.GetBirthDate(ctx, userID)
	switch {
	case err == nil:
		if !existing.Equal(birth) {
			return OwnProfile{}, platformerrors.BadRequest("birth_date_locked", "birth date cannot be changed")
		}
	case !errors.Is(err, ErrRepoNotFound):
		return OwnProfile{}, err
	}

	if err := s.repo.Upsert(ctx, userID, birth, cleaned); err != nil {
		if errors.Is(err, ErrRepoUnknownInterest) {
			return OwnProfile{}, platformerrors.BadRequest("unknown_interest", "one of the interests does not exist")
		}
		return OwnProfile{}, err
	}
	return s.Own(ctx, userID)
}

// SaveLocation stores the position on a ~1 km grid; precise coordinates are never persisted.
func (s *Service) SaveLocation(ctx context.Context, userID string, in LocationInput) (OwnProfile, error) {
	city, err := cleanText(in.City, maxCity, false)
	if err != nil {
		return OwnProfile{}, platformerrors.BadRequest("invalid_city", "city is not valid")
	}
	if err := s.repo.SetLocation(ctx, userID, coarsen(in.Latitude), coarsen(in.Longitude), city); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return OwnProfile{}, errNotFound
		}
		return OwnProfile{}, err
	}
	return s.Own(ctx, userID)
}

func (s *Service) Preferences(ctx context.Context, userID string) (Preferences, error) {
	p, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return Preferences{}, err
	}
	if p == nil {
		return Preferences{
			InterestedIn:  []Gender{GenderMan, GenderWoman, GenderNonBinary},
			MinAge:        MinAge,
			MaxAge:        MaxAge,
			MaxDistanceKm: 50,
		}, nil
	}
	return *p, nil
}

func (s *Service) SavePreferences(ctx context.Context, userID string, p Preferences) (Preferences, error) {
	if p.MinAge > p.MaxAge {
		return Preferences{}, platformerrors.BadRequest("invalid_age_range", "minimum age cannot exceed maximum age")
	}
	seen := make(map[Gender]bool, len(p.InterestedIn))
	unique := make([]Gender, 0, len(p.InterestedIn))
	for _, g := range p.InterestedIn {
		if !validGender(g) {
			return Preferences{}, platformerrors.BadRequest("invalid_gender", "unknown gender value")
		}
		if !seen[g] {
			seen[g] = true
			unique = append(unique, g)
		}
	}
	p.InterestedIn = unique
	if err := s.repo.UpsertPreferences(ctx, userID, p); err != nil {
		return Preferences{}, err
	}
	return p, nil
}

// Public returns another user's profile if the visibility rule allows it.
func (s *Service) Public(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if !ok {
		return PublicProfile{}, errProfileHidden
	}
	list, err := s.repo.PublicByIDs(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	if len(list) == 0 {
		return PublicProfile{}, errProfileHidden
	}
	return list[0], nil
}

// PublicByIDs is used by discovery and chat to hydrate lists of already-authorised candidates.
func (s *Service) PublicByIDs(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error) {
	return s.repo.PublicByIDs(ctx, viewerID, ids)
}

// CanView exposes the visibility rule (used by photo downloads).
func (s *Service) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	return s.repo.CanView(ctx, viewerID, targetID)
}

func normalizeInput(in ProfileInput) (ProfileInput, error) {
	name, err := cleanText(in.FirstName, maxFirstName, true)
	if err != nil || name == "" {
		return in, platformerrors.BadRequest("invalid_first_name", "first name must be 1 to 40 characters")
	}
	bio, err := cleanText(in.Bio, maxBio, false)
	if err != nil {
		return in, platformerrors.BadRequest("invalid_bio", "bio must be at most 500 characters")
	}
	city, err := cleanText(in.City, maxCity, false)
	if err != nil {
		return in, platformerrors.BadRequest("invalid_city", "city must be at most 80 characters")
	}
	if !validGender(in.Gender) {
		return in, platformerrors.BadRequest("invalid_gender", "unknown gender value")
	}
	ids, err := uniqueInterestIDs(in.InterestIDs)
	if err != nil {
		return in, err
	}
	in.FirstName, in.Bio, in.City, in.InterestIDs = name, bio, city, ids
	return in, nil
}

func uniqueInterestIDs(ids []int) ([]int, error) {
	seen := make(map[int]bool, len(ids))
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id < 1 || id > math.MaxInt16 {
			return nil, platformerrors.BadRequest("unknown_interest", "one of the interests does not exist")
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	if len(out) > maxInterests {
		return nil, platformerrors.BadRequest("too_many_interests", "choose at most 10 interests")
	}
	return out, nil
}

// cleanText trims, rejects control characters (newlines allowed unless singleLine) and enforces a rune limit.
func cleanText(raw string, maxRunes int, singleLine bool) (string, error) {
	text := strings.TrimSpace(raw)
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) > maxRunes {
		return "", errors.New("invalid text")
	}
	for _, r := range text {
		if unicode.IsControl(r) && (singleLine || (r != '\n' && r != '\t')) {
			return "", errors.New("control characters")
		}
	}
	return text, nil
}

func coarsen(v float64) float64 { return math.Round(v*coordinateScale) / coordinateScale }

// AgeAt computes whole years between birth and now (calendar-accurate).
func AgeAt(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// ApproxDistance rounds a distance up to 5 km steps; anything within 1 km is reported as 1. nil stays nil.
func ApproxDistance(km *float64) *int {
	if km == nil || math.IsNaN(*km) {
		return nil
	}
	var out int
	if *km <= 1 {
		out = 1
	} else {
		out = int(math.Ceil(*km/distanceStepKm)) * distanceStepKm
	}
	return &out
}
