package profiles

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/media"
)

type Service struct {
	repo   Repository
	signer *media.Signer
	now    func() time.Time
}

func NewService(repo Repository, signer *media.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func invalid(msg string) error { return platformerrors.New(http.StatusBadRequest, "validation", msg) }

var errNotFound = platformerrors.New(http.StatusNotFound, "not_found", "not found")

// AgeOn computes a full-years age on the given day.
func AgeOn(birth, today time.Time) int {
	age := today.Year() - birth.Year()
	if today.Month() < birth.Month() || (today.Month() == birth.Month() && today.Day() < birth.Day()) {
		age--
	}
	return age
}

func cleanText(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || r >= 0x20 && r != 0x7f {
			return r
		}
		return -1
	}, s))
}

func validateName(name string) (string, error) {
	name = strings.Join(strings.Fields(cleanText(name)), " ")
	if n := utf8.RuneCountInString(name); n < 1 || n > 40 {
		return "", invalid("first name must be between 1 and 40 characters")
	}
	return name, nil
}

// ValidatePreferences enforces ranges and normalises the gender list.
func ValidatePreferences(p Preferences) (Preferences, error) {
	seen := map[string]bool{}
	list := []string{}
	for _, g := range p.InterestedIn {
		if !validGenders[g] {
			return p, invalid("invalid gender in interestedIn")
		}
		if !seen[g] {
			seen[g] = true
			list = append(list, g)
		}
	}
	if len(list) == 0 {
		return p, invalid("choose at least one gender you are interested in")
	}
	if p.MinAge < MinAge || p.MaxAge > MaxAge || p.MinAge > p.MaxAge {
		return p, invalid("age range must be within 18-99 and minAge <= maxAge")
	}
	if p.MaxDistanceKm < 1 || p.MaxDistanceKm > 500 {
		return p, invalid("maxDistanceKm must be between 1 and 500")
	}
	p.InterestedIn = list
	return p, nil
}

// UpsertProfile creates the profile (birthDate required, must be 18+) or updates it.
func (s *Service) UpsertProfile(ctx context.Context, userID string, req UpsertProfileRequest) (OwnProfile, error) {
	name, err := validateName(req.FirstName)
	if err != nil {
		return OwnProfile{}, err
	}
	if !validGenders[req.Gender] {
		return OwnProfile{}, invalid("gender must be woman, man or nonbinary")
	}
	var bio, city *string
	if req.Bio != nil {
		b := cleanText(*req.Bio)
		if utf8.RuneCountInString(b) > 500 {
			return OwnProfile{}, invalid("bio must be at most 500 characters")
		}
		bio = &b
	}
	if req.City != nil {
		c := strings.Join(strings.Fields(cleanText(*req.City)), " ")
		if utf8.RuneCountInString(c) > 80 {
			return OwnProfile{}, invalid("city must be at most 80 characters")
		}
		city = &c
	}
	if req.Interests != nil {
		if len(*req.Interests) > 10 {
			return OwnProfile{}, invalid("choose at most 10 interests")
		}
	}

	existing, err := s.repo.GetProfile(ctx, userID)
	switch {
	case errors.Is(err, ErrNotFound):
		if req.BirthDate == nil {
			return OwnProfile{}, invalid("birthDate is required")
		}
		birth, perr := time.Parse("2006-01-02", *req.BirthDate)
		if perr != nil {
			return OwnProfile{}, invalid("birthDate must be YYYY-MM-DD")
		}
		age := AgeOn(birth, s.now().UTC())
		if age < MinAge {
			return OwnProfile{}, platformerrors.New(http.StatusForbidden, "underage", "you must be at least 18 years old")
		}
		if age > 120 {
			return OwnProfile{}, invalid("invalid birth date")
		}
		sp := storedProfile{FirstName: name, BirthDate: birth, Gender: req.Gender, IsVisible: true}
		if bio != nil {
			sp.Bio = *bio
		}
		if city != nil {
			sp.City = *city
		}
		if req.IsVisible != nil {
			sp.IsVisible = *req.IsVisible
		}
		if err := s.repo.CreateProfile(ctx, userID, sp); err != nil {
			return OwnProfile{}, err
		}
	case err != nil:
		return OwnProfile{}, err
	default:
		if req.BirthDate != nil && *req.BirthDate != existing.BirthDate.Format("2006-01-02") {
			return OwnProfile{}, invalid("birth date cannot be changed")
		}
		if err := s.repo.UpdateProfile(ctx, userID, name, req.Gender, bio, city, req.IsVisible); err != nil {
			return OwnProfile{}, err
		}
	}

	if req.Interests != nil {
		slugs := dedupe(*req.Interests)
		n, err := s.repo.SetInterests(ctx, userID, slugs)
		if err != nil {
			return OwnProfile{}, err
		}
		if n != len(slugs) {
			return OwnProfile{}, invalid("unknown interest")
		}
	}
	return s.Own(ctx, userID)
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.ToLower(strings.TrimSpace(v))
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func (s *Service) SetPreferences(ctx context.Context, userID string, req PreferencesRequest) (OwnProfile, error) {
	if _, err := s.repo.GetProfile(ctx, userID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return OwnProfile{}, invalid("create your profile first")
		}
		return OwnProfile{}, err
	}
	p, err := ValidatePreferences(Preferences{InterestedIn: req.InterestedIn, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm})
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, p); err != nil {
		return OwnProfile{}, err
	}
	return s.Own(ctx, userID)
}

// SetLocation stores a coarsened position: the exact coordinates are discarded.
func (s *Service) SetLocation(ctx context.Context, userID string, req LocationRequest) (OwnProfile, error) {
	if !geo.ValidLatLng(req.Latitude, req.Longitude) {
		return OwnProfile{}, invalid("invalid coordinates")
	}
	var city *string
	if req.City != nil {
		c := strings.Join(strings.Fields(cleanText(*req.City)), " ")
		if utf8.RuneCountInString(c) > 80 {
			return OwnProfile{}, invalid("city must be at most 80 characters")
		}
		city = &c
	}
	err := s.repo.SetLocation(ctx, userID, geo.Coarsen(req.Latitude), geo.Coarsen(req.Longitude), city)
	if errors.Is(err, ErrNotFound) {
		return OwnProfile{}, invalid("create your profile first")
	}
	if err != nil {
		return OwnProfile{}, err
	}
	return s.Own(ctx, userID)
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// Own returns the private profile with completeness information.
func (s *Service) Own(ctx context.Context, userID string) (OwnProfile, error) {
	sp, err := s.repo.GetProfile(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return OwnProfile{UserID: userID, Interests: []Interest{}, Photos: []Photo{}, Missing: []string{"profile", "preferences", "photos"}}, nil
	}
	if err != nil {
		return OwnProfile{}, err
	}
	interests, err := s.repo.UserInterests(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	photos, err := s.repo.UserPhotos(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	out := OwnProfile{
		UserID: userID, FirstName: sp.FirstName, BirthDate: sp.BirthDate.Format("2006-01-02"),
		Age: AgeOn(sp.BirthDate, s.now().UTC()), Gender: sp.Gender, Bio: sp.Bio, City: sp.City,
		HasLocation: sp.HasLocation, IsVisible: sp.IsVisible, Interests: orEmptyInterests(interests[userID]),
		Photos: s.toPhotos(photos), Preferences: prefs, Missing: []string{},
	}
	if prefs == nil {
		out.Missing = append(out.Missing, "preferences")
	}
	if len(out.Photos) == 0 {
		out.Missing = append(out.Missing, "photos")
	}
	out.IsComplete = len(out.Missing) == 0
	return out, nil
}

func orEmptyInterests(in []Interest) []Interest {
	if in == nil {
		return []Interest{}
	}
	return in
}

func (s *Service) toPhotos(in []storedPhoto) []Photo {
	out := make([]Photo, 0, len(in))
	for _, p := range in {
		out = append(out, Photo{ID: p.ID, URL: s.signer.URL(p.ID), Position: p.Position})
	}
	return out
}

// Views hydrates public profiles for the given ids using three queries total
// (no N+1). The result keeps the order of ids and silently drops blocked users.
func (s *Service) Views(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error) {
	if len(ids) == 0 {
		return []PublicProfile{}, nil
	}
	rows, err := s.repo.PublicViews(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	visible := make([]string, 0, len(rows))
	byID := make(map[string]publicRow, len(rows))
	for _, r := range rows {
		byID[r.UserID] = r
		visible = append(visible, r.UserID)
	}
	if len(visible) == 0 {
		return []PublicProfile{}, nil
	}
	interests, err := s.repo.UserInterests(ctx, visible)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.UserPhotos(ctx, visible)
	if err != nil {
		return nil, err
	}
	photosByUser := map[string][]storedPhoto{}
	for _, p := range photos {
		photosByUser[p.UserID] = append(photosByUser[p.UserID], p)
	}
	today := s.now().UTC()
	out := make([]PublicProfile, 0, len(ids))
	for _, id := range ids {
		r, ok := byID[id]
		if !ok {
			continue
		}
		pp := PublicProfile{
			UserID: r.UserID, FirstName: r.FirstName, Age: AgeOn(r.BirthDate, today), Gender: r.Gender,
			Bio: r.Bio, City: r.City, Interests: orEmptyInterests(interests[id]), Photos: s.toPhotos(photosByUser[id]),
		}
		if r.DistanceKm != nil {
			d := geo.ApproxKm(*r.DistanceKm)
			pp.DistanceKm = &d
		}
		out = append(out, pp)
	}
	return out, nil
}

// Public returns another user's profile if the viewer is allowed to see it.
func (s *Service) Public(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if !ok {
		return PublicProfile{}, errNotFound
	}
	views, err := s.Views(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	if len(views) == 0 {
		return PublicProfile{}, errNotFound
	}
	return views[0], nil
}
