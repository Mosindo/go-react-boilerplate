package profiles

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/storage"
)

const maxInterests = 10

type Service struct {
	repo   Repository
	signer *storage.Signer
	now    func() time.Time
}

func NewService(repo Repository, signer *storage.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	row, err := s.repo.GetOwn(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return OwnProfile{}, perr.NotFound("profile_not_found", "profile has not been created yet")
		}
		return OwnProfile{}, err
	}
	photos, err := s.repo.PhotosFor(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	p := row.OwnProfile
	p.Age = ageOn(row.BirthDate, s.now())
	p.Photos = s.photoViews(photos)
	if p.Interests == nil {
		p.Interests = []string{}
	}
	return p, nil
}

func (s *Service) Save(ctx context.Context, userID string, in ProfileInput) (OwnProfile, error) {
	if err := s.normalize(ctx, &in); err != nil {
		return OwnProfile{}, err
	}
	lat, lng, err := s.resolveLocation(ctx, userID, in)
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.Upsert(ctx, userID, in, lat, lng); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) normalize(ctx context.Context, in *ProfileInput) error {
	in.FirstName = cleanLine(in.FirstName)
	if n := utf8.RuneCountInString(in.FirstName); n < 1 || n > 40 {
		return perr.Invalid("invalid_first_name", "first name must be between 1 and 40 characters")
	}
	if !contains(Genders, in.Gender) {
		return perr.Invalid("invalid_gender", "gender is not valid")
	}
	in.Bio = strings.TrimSpace(in.Bio)
	if utf8.RuneCountInString(in.Bio) > 500 {
		return perr.Invalid("invalid_bio", "bio must be at most 500 characters")
	}
	in.City = cleanLine(in.City)
	in.Occupation = cleanLine(in.Occupation)
	if utf8.RuneCountInString(in.City) > 80 || utf8.RuneCountInString(in.Occupation) > 80 {
		return perr.Invalid("invalid_field", "city and occupation must be at most 80 characters")
	}
	if in.HeightCM != nil && (*in.HeightCM < 100 || *in.HeightCM > 250) {
		return perr.Invalid("invalid_height", "height must be between 100 and 250 cm")
	}
	if in.LookingFor == "" {
		in.LookingFor = "unsure"
	}
	if !contains(LookingFors, in.LookingFor) {
		return perr.Invalid("invalid_looking_for", "relationship goal is not valid")
	}

	seen := map[string]bool{}
	slugs := make([]string, 0, len(in.Interests))
	for _, slug := range in.Interests {
		slug = strings.TrimSpace(slug)
		if slug != "" && !seen[slug] {
			seen[slug] = true
			slugs = append(slugs, slug)
		}
	}
	if len(slugs) > maxInterests {
		return perr.Invalid("too_many_interests", "you can pick up to 10 interests")
	}
	if len(slugs) > 0 {
		known, err := s.repo.ListInterests(ctx)
		if err != nil {
			return err
		}
		valid := map[string]bool{}
		for _, k := range known {
			valid[k.Slug] = true
		}
		for _, slug := range slugs {
			if !valid[slug] {
				return perr.Invalid("unknown_interest", "unknown interest: "+slug)
			}
		}
	}
	in.Interests = slugs
	return nil
}

// resolveLocation returns the coordinates to store: rounded to ~1 km, kept as is when the caller
// sent none, cleared on request.
func (s *Service) resolveLocation(ctx context.Context, userID string, in ProfileInput) (*float64, *float64, error) {
	if in.ClearLocation {
		return nil, nil, nil
	}
	if in.Latitude == nil && in.Longitude == nil {
		return s.repo.ViewerLocation(ctx, userID)
	}
	if in.Latitude == nil || in.Longitude == nil ||
		*in.Latitude < -90 || *in.Latitude > 90 || *in.Longitude < -180 || *in.Longitude > 180 {
		return nil, nil, perr.Invalid("invalid_location", "latitude and longitude must both be valid")
	}
	lat, lng := geo.Round2(*in.Latitude), geo.Round2(*in.Longitude)
	return &lat, &lng, nil
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	p, err := s.repo.GetPreferences(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return Preferences{}, perr.NotFound("profile_not_found", "create your profile first")
	}
	return p, err
}

func (s *Service) SavePreferences(ctx context.Context, userID string, p Preferences) (Preferences, error) {
	if len(p.InterestedIn) == 0 {
		return Preferences{}, perr.Invalid("invalid_interested_in", "pick at least one gender")
	}
	seen := map[string]bool{}
	deduped := []string{}
	for _, g := range p.InterestedIn {
		if !contains(Genders, g) {
			return Preferences{}, perr.Invalid("invalid_interested_in", "gender preference is not valid")
		}
		if !seen[g] {
			seen[g] = true
			deduped = append(deduped, g)
		}
	}
	p.InterestedIn = deduped
	if p.MinAge < 18 || p.MaxAge > 99 || p.MinAge > p.MaxAge {
		return Preferences{}, perr.Invalid("invalid_age_range", "age range must be within 18-99 and min <= max")
	}
	if p.MaxDistanceKm != nil && (*p.MaxDistanceKm < 1 || *p.MaxDistanceKm > 500) {
		return Preferences{}, perr.Invalid("invalid_distance", "distance must be between 1 and 500 km")
	}
	if err := s.repo.UpdatePreferences(ctx, userID, p); err != nil {
		if errors.Is(err, ErrNotFound) {
			return Preferences{}, perr.NotFound("profile_not_found", "create your profile first")
		}
		return Preferences{}, err
	}
	return p, nil
}

// View returns another member's profile if the viewer is allowed to see it.
func (s *Service) View(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	notFound := perr.NotFound("profile_not_found", "profile not found")
	if !validUUID(targetID) {
		return PublicProfile{}, notFound
	}
	if viewerID != targetID {
		ok, err := s.repo.CanView(ctx, viewerID, targetID)
		if err != nil {
			return PublicProfile{}, err
		}
		if !ok {
			return PublicProfile{}, notFound
		}
	}
	list, err := s.LoadPublic(ctx, viewerID, []string{targetID}, nil)
	if err != nil {
		return PublicProfile{}, err
	}
	if len(list) == 0 {
		return PublicProfile{}, notFound
	}
	return list[0], nil
}

// LoadPublic builds public profiles for already-authorised ids, preserving their order, with a
// constant number of queries. exactDistances may carry SQL-computed distances (km) by user id.
func (s *Service) LoadPublic(ctx context.Context, viewerID string, ids []string, exactDistances map[string]float64) ([]PublicProfile, error) {
	rows, err := s.repo.PublicRows(ctx, ids)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.PhotosFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.InterestsFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	vLat, vLng, err := s.repo.ViewerLocation(ctx, viewerID)
	if err != nil {
		return nil, err
	}

	photosBy := map[string][]storedPhoto{}
	for _, p := range photos {
		photosBy[p.UserID] = append(photosBy[p.UserID], p)
	}
	byID := make(map[string]PublicProfile, len(rows))
	for _, row := range rows {
		p := row.PublicProfile
		p.Age = ageOn(row.BirthDate, s.now())
		p.Photos = s.photoViews(photosBy[p.UserID])
		p.Interests = interests[p.UserID]
		if p.Interests == nil {
			p.Interests = []Interest{}
		}
		if row.ShowDistance && p.UserID != viewerID {
			if km, ok := exactDistances[p.UserID]; ok {
				d := geo.Display(km)
				p.DistanceKm = &d
			} else if vLat != nil && vLng != nil && row.Lat != nil && row.Lng != nil {
				d := geo.Display(geo.DistanceKm(*vLat, *vLng, *row.Lat, *row.Lng))
				p.DistanceKm = &d
			}
		}
		byID[p.UserID] = p
	}
	out := make([]PublicProfile, 0, len(ids))
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Service) photoViews(in []storedPhoto) []Photo {
	out := make([]Photo, 0, len(in))
	for _, p := range in {
		out = append(out, Photo{ID: p.ID, URL: s.signer.URL(p.ID, "full"), ThumbURL: s.signer.URL(p.ID, "thumb"), Position: p.Position})
	}
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// cleanLine trims, collapses whitespace and strips control characters from single-line fields.
func cleanLine(v string) string {
	v = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, v)
	return strings.Join(strings.Fields(v), " ")
}

func ageOn(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

func validUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, r := range v {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if r != '-' {
				return false
			}
		case !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'):
			return false
		}
	}
	return true
}
