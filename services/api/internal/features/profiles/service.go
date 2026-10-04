package profiles

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"

	platformerrors "example.com/api/internal/platform/errors"
)

const (
	MinAge        = 18
	MaxAge        = 99
	maxInterests  = 10
	maxBioRunes   = 500
	maxCityRunes  = 80
	minNameRunes  = 2
	maxNameRunes  = 40
	maxDistanceKm = 500
)

var genders = map[string]struct{}{"man": {}, "woman": {}, "nonbinary": {}}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) GetOwn(ctx context.Context, userID string) (*OwnProfile, error) {
	p, err := s.repo.GetOwn(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return nil, platformerrors.NotFound("Profile not created yet")
	}
	return p, err
}

func (s *Service) Save(ctx context.Context, userID string, req SaveProfileRequest) (*OwnProfile, error) {
	fields := map[string]string{}

	name := cleanText(req.DisplayName)
	if n := len([]rune(name)); n < minNameRunes || n > maxNameRunes {
		fields["displayName"] = "Use between 2 and 40 characters"
	}
	if _, ok := genders[req.Gender]; !ok {
		fields["gender"] = "Must be man, woman or nonbinary"
	}
	bio := cleanMultiline(req.Bio)
	if len([]rune(bio)) > maxBioRunes {
		fields["bio"] = "Use at most 500 characters"
	}
	city := cleanText(req.City)
	if len([]rune(city)) > maxCityRunes {
		fields["city"] = "Use at most 80 characters"
	}
	slugs, err := normalizeInterests(req.Interests)
	if err != nil {
		fields["interests"] = err.Error()
	}

	birth, birthErr := time.Parse("2006-01-02", req.BirthDate)
	today := s.now().UTC()
	switch {
	case birthErr != nil:
		fields["birthDate"] = "Use the format YYYY-MM-DD"
	case AgeAt(birth, today) < MinAge:
		fields["birthDate"] = "You must be at least 18 years old"
	case AgeAt(birth, today) > MaxAge:
		fields["birthDate"] = "Enter a valid birth date"
	}

	existing, getErr := s.repo.GetOwn(ctx, userID)
	if getErr != nil && !errors.Is(getErr, ErrNotFound) {
		return nil, getErr
	}
	if existing != nil && birthErr == nil && existing.BirthDate != req.BirthDate {
		fields["birthDate"] = "Birth date cannot be changed"
	}
	if len(fields) > 0 {
		return nil, platformerrors.Validation("Invalid input", fields)
	}

	in := ProfileInput{
		UserID:       userID,
		DisplayName:  name,
		BirthDate:    birth,
		Gender:       req.Gender,
		Bio:          bio,
		City:         city,
		InterestSlug: slugs,
		ShowDistance: boolOr(req.ShowDistance, existingBool(existing, func(p *OwnProfile) bool { return p.ShowDistance }, true)),
		Discoverable: boolOr(req.Discoverable, existingBool(existing, func(p *OwnProfile) bool { return p.Discoverable }, true)),
	}
	if err := s.repo.Save(ctx, in); err != nil {
		if errors.Is(err, ErrUnknownInterests) {
			return nil, platformerrors.FieldInvalid("interests", "Unknown interest")
		}
		return nil, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) SetLocation(ctx context.Context, userID string, req LocationRequest) error {
	if req.Latitude == nil || req.Longitude == nil ||
		math.IsNaN(*req.Latitude) || math.IsNaN(*req.Longitude) ||
		*req.Latitude < -90 || *req.Latitude > 90 || *req.Longitude < -180 || *req.Longitude > 180 {
		return platformerrors.Validation("Invalid coordinates", map[string]string{"latitude": "Invalid coordinates"})
	}
	// ~1.1 km grid: precise enough for "within X km", too coarse to locate a person.
	lat := math.Round(*req.Latitude*100) / 100
	lng := math.Round(*req.Longitude*100) / 100
	if err := s.repo.SetLocation(ctx, userID, lat, lng); err != nil {
		return mapNotFound(err)
	}
	return nil
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return mapNotFound(s.repo.ClearLocation(ctx, userID))
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	p, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return p, nil
}

func (s *Service) SavePreferences(ctx context.Context, userID string, p Preferences) (*Preferences, error) {
	fields := map[string]string{}
	seen := map[string]struct{}{}
	clean := make([]string, 0, 3)
	for _, g := range p.InterestedIn {
		if _, ok := genders[g]; !ok {
			fields["interestedIn"] = "Values must be man, woman or nonbinary"
			break
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			clean = append(clean, g)
		}
	}
	if len(clean) == 0 && fields["interestedIn"] == "" {
		fields["interestedIn"] = "Choose at least one"
	}
	if p.MinAge < MinAge || p.MinAge > MaxAge {
		fields["minAge"] = "Must be between 18 and 99"
	}
	if p.MaxAge < MinAge || p.MaxAge > MaxAge {
		fields["maxAge"] = "Must be between 18 and 99"
	}
	if fields["minAge"] == "" && fields["maxAge"] == "" && p.MinAge > p.MaxAge {
		fields["minAge"] = "Must not exceed the maximum age"
	}
	if p.MaxDistanceKm < 1 || p.MaxDistanceKm > maxDistanceKm {
		fields["maxDistanceKm"] = "Must be between 1 and 500"
	}
	if len(fields) > 0 {
		return nil, platformerrors.Validation("Invalid input", fields)
	}
	p.InterestedIn = clean
	if err := s.repo.SavePreferences(ctx, userID, p); err != nil {
		return nil, mapNotFound(err)
	}
	return &p, nil
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) GetVisible(ctx context.Context, viewerID, targetID string) (*PublicProfile, error) {
	if !validUUID(targetID) {
		return nil, platformerrors.NotFound("Profile not found")
	}
	own, err := s.repo.GetOwn(ctx, viewerID)
	if err != nil || !own.Completed {
		if err == nil || errors.Is(err, ErrNotFound) {
			return nil, platformerrors.ProfileIncomplete()
		}
		return nil, err
	}
	p, err := s.repo.GetVisible(ctx, viewerID, targetID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return p, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return platformerrors.NotFound("Profile not found")
	}
	return err
}

func boolOr(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func existingBool(p *OwnProfile, get func(*OwnProfile) bool, def bool) bool {
	if p == nil {
		return def
	}
	return get(p)
}

func normalizeInterests(in []string) ([]string, error) {
	if len(in) > maxInterests {
		return nil, errors.New("Choose at most 10 interests")
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		slug := strings.ToLower(strings.TrimSpace(s))
		if slug == "" {
			return nil, errors.New("Invalid interest")
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, slug)
	}
	return out, nil
}

// cleanText trims, collapses whitespace and drops control characters (single-line fields).
func cleanText(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		case unicode.IsControl(r):
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// cleanMultiline keeps line breaks (max two in a row) but drops other control characters.
func cleanMultiline(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}
	return out
}

func validUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
				return false
			}
		}
	}
	return true
}

// ValidUUID is shared with other features for path parameter checks.
func ValidUUID(s string) bool { return validUUID(s) }
