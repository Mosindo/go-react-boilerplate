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
	ErrInvalid  = errors.New("invalid input")
	ErrUnderage = errors.New("you must be at least 18 years old")
)

// ValidationError carries a user-facing message and maps to a 400.
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

func invalid(msg string) error { return ValidationError{Message: msg} }

type Service struct {
	repo *Repository
	now  func() time.Time
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// AgeAt returns the age in full years at the given instant.
func AgeAt(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

func (s *Service) toMyProfile(ctx context.Context, p storedProfile) (MyProfile, error) {
	interests, err := s.repo.InterestsByUsers(ctx, []string{p.UserID})
	if err != nil {
		return MyProfile{}, err
	}
	photos, err := s.repo.PhotosByUsers(ctx, []string{p.UserID})
	if err != nil {
		return MyProfile{}, err
	}
	out := MyProfile{
		UserID: p.UserID, FirstName: p.FirstName, BirthDate: p.BirthDate.Format("2006-01-02"),
		Age: AgeAt(p.BirthDate, s.now()), Gender: p.Gender, Bio: p.Bio, City: p.City, HasLocation: p.HasLocation,
		Discoverable: p.Discoverable, ShowDistance: p.ShowDistance,
		Interests: interests[p.UserID], Photos: photos[p.UserID],
	}
	if out.Interests == nil {
		out.Interests = []Interest{}
	}
	if out.Photos == nil {
		out.Photos = []Photo{}
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, userID string) (ProfileState, error) {
	state := ProfileState{}
	stored, err := s.repo.getProfile(ctx, userID)
	switch {
	case err == nil:
		p, err := s.toMyProfile(ctx, stored)
		if err != nil {
			return state, err
		}
		state.Profile = &p
	case errors.Is(err, ErrNotFound):
	default:
		return state, err
	}
	prefs, err := s.repo.getPreferences(ctx, userID)
	if err != nil {
		return state, err
	}
	state.Preferences = prefs
	state.Complete = state.Profile != nil && state.Preferences != nil && len(state.Profile.Photos) > 0
	return state, nil
}

func cleanText(raw string, max int, allowNewlines bool) (string, error) {
	text := strings.TrimSpace(raw)
	if !utf8.ValidString(text) {
		return "", invalid("text must be valid UTF-8")
	}
	for _, r := range text {
		if unicode.IsControl(r) && !(allowNewlines && (r == '\n' || r == '\t')) {
			return "", invalid("text contains invalid characters")
		}
	}
	if utf8.RuneCountInString(text) > max {
		return "", invalid("text is too long")
	}
	return text, nil
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (MyProfile, error) {
	name, err := cleanText(strings.Join(strings.Fields(req.FirstName), " "), MaxFirstName, false)
	if err != nil || name == "" {
		return MyProfile{}, invalid("firstName must be 1 to 40 characters")
	}
	if !validGender(req.Gender) {
		return MyProfile{}, invalid("gender must be man, woman or nonbinary")
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return MyProfile{}, invalid("birthDate must be formatted YYYY-MM-DD")
	}
	age := AgeAt(birth, s.now())
	if age < MinAge {
		return MyProfile{}, ErrUnderage
	}
	if age > MaxAge {
		return MyProfile{}, invalid("birthDate is out of range")
	}
	bio, err := cleanText(req.Bio, MaxBio, true)
	if err != nil {
		return MyProfile{}, invalid("bio must be at most 500 characters")
	}
	if len(req.Interests) > MaxInterests {
		return MyProfile{}, invalid("at most 10 interests")
	}
	slugs := make([]string, 0, len(req.Interests))
	seen := map[string]struct{}{}
	for _, slug := range req.Interests {
		slug = strings.TrimSpace(slug)
		if _, dup := seen[slug]; dup || slug == "" {
			continue
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}

	in := ProfileInput{FirstName: name, BirthDate: birth, Gender: req.Gender, Bio: bio, InterestSlug: slugs, Discoverable: true, ShowDistance: true}
	if existing, err := s.repo.getProfile(ctx, userID); err == nil {
		in.Discoverable, in.ShowDistance = existing.Discoverable, existing.ShowDistance
	} else if !errors.Is(err, ErrNotFound) {
		return MyProfile{}, err
	}
	if req.Discoverable != nil {
		in.Discoverable = *req.Discoverable
	}
	if req.ShowDistance != nil {
		in.ShowDistance = *req.ShowDistance
	}

	if err := s.repo.upsertProfile(ctx, userID, in); err != nil {
		if errors.Is(err, ErrUnknownInterests) {
			return MyProfile{}, invalid("unknown interest")
		}
		return MyProfile{}, err
	}
	stored, err := s.repo.getProfile(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}
	return s.toMyProfile(ctx, stored)
}

func (s *Service) SetPreferences(ctx context.Context, userID string, req PreferencesRequest) (Preferences, error) {
	if len(req.InterestedIn) < 1 || len(req.InterestedIn) > 3 {
		return Preferences{}, invalid("interestedIn must contain 1 to 3 genders")
	}
	seen := map[Gender]struct{}{}
	genders := make([]Gender, 0, 3)
	for _, g := range req.InterestedIn {
		if !validGender(g) {
			return Preferences{}, invalid("interestedIn contains an invalid gender")
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			genders = append(genders, g)
		}
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, invalid("age range must be within 18-99 and minAge <= maxAge")
	}
	if req.MaxDistanceKm != nil && (*req.MaxDistanceKm < 1 || *req.MaxDistanceKm > MaxDistanceKm) {
		return Preferences{}, invalid("maxDistanceKm must be between 1 and 500")
	}
	prefs := Preferences{InterestedIn: genders, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}
	if err := s.repo.upsertPreferences(ctx, userID, prefs); err != nil {
		return Preferences{}, err
	}
	return prefs, nil
}

// SetLocation stores a coarse (2 decimals, ~1 km) position; the precise input is never persisted.
func (s *Service) SetLocation(ctx context.Context, userID string, req LocationRequest) (MyProfile, error) {
	if req.Latitude == nil || req.Longitude == nil {
		return MyProfile{}, invalid("latitude and longitude are required")
	}
	lat, lng := *req.Latitude, *req.Longitude
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return MyProfile{}, invalid("coordinates are out of range")
	}
	city, err := cleanText(req.City, MaxCity, false)
	if err != nil {
		return MyProfile{}, invalid("city must be at most 80 characters")
	}
	lat, lng = math.Round(lat*100)/100, math.Round(lng*100)/100
	if err := s.repo.setLocation(ctx, userID, lat, lng, city); err != nil {
		return MyProfile{}, err
	}
	stored, err := s.repo.getProfile(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}
	return s.toMyProfile(ctx, stored)
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return s.repo.clearLocation(ctx, userID)
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.listInterests(ctx)
}

// PublicProfiles builds what viewerID sees of the given users, in a fixed number of queries.
// Callers must have applied their own visibility rules to ids.
func (s *Service) PublicProfiles(ctx context.Context, viewerID string, ids []string) (map[string]PublicProfile, error) {
	out := make(map[string]PublicProfile, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.repo.publicRows(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	all := append([]string{viewerID}, ids...)
	interests, err := s.repo.InterestsByUsers(ctx, all)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.PhotosByUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	mine := map[string]struct{}{}
	for _, i := range interests[viewerID] {
		mine[i.Slug] = struct{}{}
	}
	for _, row := range rows {
		p := PublicProfile{
			UserID: row.UserID, FirstName: row.FirstName, Age: row.Age, Bio: row.Bio, City: row.City, DistanceKm: row.DistanceKm,
			Interests: interests[row.UserID], Photos: photos[row.UserID], SharedInterests: []string{},
		}
		if p.Interests == nil {
			p.Interests = []Interest{}
		}
		if p.Photos == nil {
			p.Photos = []Photo{}
		}
		for _, i := range p.Interests {
			if _, ok := mine[i.Slug]; ok {
				p.SharedInterests = append(p.SharedInterests, i.Slug)
			}
		}
		out[row.UserID] = p
	}
	return out, nil
}

// View returns the profile of target as seen by viewer, or ErrNotFound when it must not be visible.
func (s *Service) View(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if !ok {
		return PublicProfile{}, ErrNotFound
	}
	profiles, err := s.PublicProfiles(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	p, found := profiles[targetID]
	if !found {
		return PublicProfile{}, ErrNotFound
	}
	return p, nil
}

// Summaries returns lightweight cards (name + primary thumbnail) for many users.
func (s *Service) Summaries(ctx context.Context, userIDs []string) (map[string]ProfileSummary, error) {
	out := make(map[string]ProfileSummary, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := s.repo.pool.Query(ctx, `
		SELECT p.user_id, p.first_name, ph.id
		FROM profiles p
		LEFT JOIN photos ph ON ph.user_id = p.user_id AND ph.position = 0
		WHERE p.user_id = ANY($1::uuid[])
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sum ProfileSummary
		var photoID *string
		if err := rows.Scan(&sum.UserID, &sum.FirstName, &photoID); err != nil {
			return nil, err
		}
		if photoID != nil {
			thumb := PhotoURLs(*photoID, 0).ThumbURL
			sum.ThumbURL = &thumb
		}
		out[sum.UserID] = sum
	}
	return out, rows.Err()
}

// CanView lets other features (photos) reuse the profile visibility rule.
func (s *Service) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	return s.repo.CanView(ctx, viewerID, targetID)
}
