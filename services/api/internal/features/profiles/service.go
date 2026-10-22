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
	"example.com/api/internal/platform/geo"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) GetSelf(ctx context.Context, userID string) (SelfProfile, error) {
	return s.repo.GetSelf(ctx, userID)
}

func (s *Service) UpdateSelf(ctx context.Context, userID string, req UpdateProfileRequest) (SelfProfile, error) {
	stored, err := s.validateProfile(userID, req)
	if err != nil {
		return SelfProfile{}, err
	}
	if existing, err := s.repo.GetSelf(ctx, userID); err == nil {
		// Privacy toggles keep their value when the client omits them.
		if req.IsDiscoverable == nil {
			stored.IsDiscoverable = existing.IsDiscoverable
		}
		if req.ShowDistance == nil {
			stored.ShowDistance = existing.ShowDistance
		}
	} else if !errors.Is(err, ErrNotFound) {
		return SelfProfile{}, err
	}
	if err := s.repo.Upsert(ctx, stored); err != nil {
		if errors.Is(err, ErrUnknownInterests) {
			return SelfProfile{}, platformerrors.Invalid("interestIds", "centre d'intérêt inconnu")
		}
		return SelfProfile{}, err
	}
	return s.repo.GetSelf(ctx, userID)
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	return s.repo.GetPreferences(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (Preferences, error) {
	prefs, err := validatePreferences(req)
	if err != nil {
		return Preferences{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, prefs); err != nil {
		return Preferences{}, err
	}
	return prefs, nil
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	return s.repo.GetPublic(ctx, viewerID, targetID)
}

func (s *Service) validateProfile(userID string, req UpdateProfileRequest) (StoredProfile, error) {
	name := strings.TrimSpace(req.FirstName)
	if name == "" || utf8.RuneCountInString(name) > MaxNameRunes {
		return StoredProfile{}, platformerrors.Invalid("firstName", "le prénom doit contenir entre 1 et 40 caractères")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' && r != '’' && r != '.' {
			return StoredProfile{}, platformerrors.Invalid("firstName", "le prénom contient des caractères non autorisés")
		}
	}

	birthdate, err := time.Parse("2006-01-02", strings.TrimSpace(req.Birthdate))
	if err != nil {
		return StoredProfile{}, platformerrors.Invalid("birthdate", "date de naissance invalide (AAAA-MM-JJ)")
	}
	if age := AgeOn(birthdate, s.now()); age < MinAge {
		return StoredProfile{}, platformerrors.Invalid("birthdate", "vous devez avoir au moins 18 ans")
	} else if age > MaxAge {
		return StoredProfile{}, platformerrors.Invalid("birthdate", "date de naissance invalide")
	}

	if !contains(Genders, req.Gender) {
		return StoredProfile{}, platformerrors.Invalid("gender", "genre invalide")
	}

	bio := cleanText(req.Bio)
	if utf8.RuneCountInString(bio) > MaxBioRunes {
		return StoredProfile{}, platformerrors.Invalid("bio", "la bio ne peut pas dépasser 500 caractères")
	}
	city := cleanText(req.City)
	if utf8.RuneCountInString(city) > MaxCityRunes {
		return StoredProfile{}, platformerrors.Invalid("city", "nom de ville trop long")
	}
	if math.IsNaN(req.Latitude) || math.IsNaN(req.Longitude) ||
		req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		return StoredProfile{}, platformerrors.Invalid("latitude", "position invalide")
	}

	ids, err := uniqueInterestIDs(req.InterestIDs)
	if err != nil {
		return StoredProfile{}, err
	}

	stored := StoredProfile{
		UserID:         userID,
		FirstName:      name,
		Birthdate:      birthdate,
		Gender:         req.Gender,
		Bio:            bio,
		City:           city,
		Latitude:       geo.RoundCoordinate(req.Latitude),
		Longitude:      geo.RoundCoordinate(req.Longitude),
		IsDiscoverable: true,
		ShowDistance:   true,
		InterestIDs:    ids,
	}
	if req.IsDiscoverable != nil {
		stored.IsDiscoverable = *req.IsDiscoverable
	}
	if req.ShowDistance != nil {
		stored.ShowDistance = *req.ShowDistance
	}
	return stored, nil
}

func validatePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	seen := map[string]bool{}
	var interested []string
	for _, g := range req.InterestedIn {
		if !contains(Genders, g) {
			return Preferences{}, platformerrors.Invalid("interestedIn", "genre recherché invalide")
		}
		if !seen[g] {
			seen[g] = true
			interested = append(interested, g)
		}
	}
	if len(interested) == 0 {
		return Preferences{}, platformerrors.Invalid("interestedIn", "choisissez au moins un genre")
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, platformerrors.Invalid("minAge", "tranche d'âge invalide (18 à 99 ans)")
	}
	distance := req.MaxDistanceKm
	if distance == 0 {
		distance = DefaultDistance
	}
	if distance < 1 || distance > MaxDistanceKm {
		return Preferences{}, platformerrors.Invalid("maxDistanceKm", "distance invalide (1 à 500 km)")
	}
	return Preferences{InterestedIn: interested, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: distance}, nil
}

// AgeOn returns the age in whole years of someone born on birthdate, at the given date.
func AgeOn(birthdate, at time.Time) int {
	years := at.Year() - birthdate.Year()
	if at.Month() < birthdate.Month() || (at.Month() == birthdate.Month() && at.Day() < birthdate.Day()) {
		years--
	}
	return years
}

func uniqueInterestIDs(ids []int16) ([]int16, error) {
	if len(ids) > MaxInterests {
		return nil, platformerrors.Invalid("interestIds", "10 centres d'intérêt maximum")
	}
	seen := map[int16]bool{}
	out := make([]int16, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, platformerrors.Invalid("interestIds", "centre d'intérêt invalide")
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, nil
}

// cleanText trims text and strips control characters (keeping newlines and tabs as spaces).
func cleanText(v string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(v) {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			b.WriteRune(' ')
		case unicode.IsControl(r):
		default:
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
