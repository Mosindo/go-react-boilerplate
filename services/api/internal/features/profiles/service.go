package profiles

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/media"
)

var (
	ErrInvalidFirstName   = errors.New("invalid first name")
	ErrInvalidBirthdate   = errors.New("invalid birthdate")
	ErrUnderage           = errors.New("underage")
	ErrInvalidGender      = errors.New("invalid gender")
	ErrTextTooLong        = errors.New("text too long")
	ErrInvalidPreferences = errors.New("invalid preferences")
	ErrTooManyInterests   = errors.New("too many interests")
	ErrInvalidLocation    = errors.New("invalid location")
	ErrProfileRequired    = errors.New("profile required")
	ErrForbidden          = errors.New("profile not visible")
)

type Service struct {
	repo   Repository
	signer *media.Signer
	now    func() time.Time
}

func NewService(repo Repository, signer *media.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func cleanText(v string) string {
	v = strings.TrimSpace(v)
	// Drop control characters except newlines.
	return strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, v)
}

func validateProfile(userID string, req ProfileRequest, now time.Time) (Profile, error) {
	firstName := cleanText(strings.ReplaceAll(req.FirstName, "\n", " "))
	if firstName == "" || utf8.RuneCountInString(firstName) > 40 {
		return Profile{}, ErrInvalidFirstName
	}
	birth, err := time.Parse("2006-01-02", req.Birthdate)
	if err != nil {
		return Profile{}, ErrInvalidBirthdate
	}
	age := AgeOn(birth, now)
	if age < MinAge {
		return Profile{}, ErrUnderage
	}
	if age > 100 {
		return Profile{}, ErrInvalidBirthdate
	}
	if !isGender(req.Gender) {
		return Profile{}, ErrInvalidGender
	}
	bio := cleanText(req.Bio)
	job := cleanText(strings.ReplaceAll(req.JobTitle, "\n", " "))
	city := cleanText(strings.ReplaceAll(req.City, "\n", " "))
	if utf8.RuneCountInString(bio) > 500 || utf8.RuneCountInString(job) > 60 || utf8.RuneCountInString(city) > 60 {
		return Profile{}, ErrTextTooLong
	}
	return Profile{UserID: userID, FirstName: firstName, Birthdate: birth, Gender: req.Gender, Bio: bio, JobTitle: job, City: city}, nil
}

func validatePreferences(req PreferencesRequest) (Preferences, error) {
	seen := map[string]bool{}
	genders := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		if !isGender(g) {
			return Preferences{}, ErrInvalidPreferences
		}
		if !seen[g] {
			seen[g] = true
			genders = append(genders, g)
		}
	}
	if len(genders) == 0 ||
		req.AgeMin < MinAge || req.AgeMax > MaxAge || req.AgeMin > req.AgeMax ||
		req.MaxDistanceKm < MinDistanceKm || req.MaxDistanceKm > MaxDistanceKm {
		return Preferences{}, ErrInvalidPreferences
	}
	return Preferences{InterestedIn: genders, AgeMin: req.AgeMin, AgeMax: req.AgeMax, MaxDistanceKm: req.MaxDistanceKm}, nil
}

func (s *Service) GetMine(ctx context.Context, userID string) (MyProfileResponse, error) {
	resp := MyProfileResponse{Interests: []Interest{}, Photos: []PhotoResponse{}, Discoverable: true, ShowDistance: true}

	p, err := s.repo.GetProfile(ctx, userID)
	switch {
	case err == nil:
		resp.Profile = &ProfileResponse{
			FirstName: p.FirstName, Birthdate: p.Birthdate.Format("2006-01-02"), Age: AgeOn(p.Birthdate, s.now()),
			Gender: p.Gender, Bio: p.Bio, JobTitle: p.JobTitle, City: p.City,
		}
		resp.HasLocation = p.Latitude != nil
		resp.LocationUpdatedAt = p.LocationUpdatedAt
		resp.Discoverable = p.Discoverable
		resp.ShowDistance = p.ShowDistance
	case errors.Is(err, ErrNotFound):
	default:
		return MyProfileResponse{}, err
	}

	prefs, err := s.repo.GetPreferences(ctx, userID)
	switch {
	case err == nil:
		resp.Preferences = &PreferencesResponse{InterestedIn: prefs.InterestedIn, AgeMin: prefs.AgeMin, AgeMax: prefs.AgeMax, MaxDistanceKm: prefs.MaxDistanceKm}
	case errors.Is(err, ErrNotFound):
	default:
		return MyProfileResponse{}, err
	}

	if resp.Interests, err = s.repo.ListUserInterests(ctx, userID); err != nil {
		return MyProfileResponse{}, err
	}
	photos, err := s.repo.ListPhotos(ctx, []string{userID})
	if err != nil {
		return MyProfileResponse{}, err
	}
	resp.Photos = s.photoResponses(photos)
	resp.IsComplete = resp.Profile != nil && resp.Preferences != nil && len(resp.Photos) > 0
	return resp, nil
}

func (s *Service) SaveProfile(ctx context.Context, userID string, req ProfileRequest) error {
	p, err := validateProfile(userID, req, s.now())
	if err != nil {
		return err
	}
	return s.repo.UpsertProfile(ctx, p)
}

func (s *Service) SavePreferences(ctx context.Context, userID string, req PreferencesRequest) error {
	p, err := validatePreferences(req)
	if err != nil {
		return err
	}
	if _, err := s.repo.GetProfile(ctx, userID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrProfileRequired
		}
		return err
	}
	return s.repo.UpsertPreferences(ctx, userID, p)
}

func (s *Service) SaveInterests(ctx context.Context, userID string, slugs []string) error {
	seen := map[string]bool{}
	clean := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		slug = strings.TrimSpace(slug)
		if slug != "" && !seen[slug] {
			seen[slug] = true
			clean = append(clean, slug)
		}
	}
	if len(clean) > MaxInterests {
		return ErrTooManyInterests
	}
	return s.repo.ReplaceUserInterests(ctx, userID, clean)
}

func (s *Service) Catalog(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterestCatalog(ctx)
}

func (s *Service) SetLocation(ctx context.Context, userID string, lat, lng float64) error {
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return ErrInvalidLocation
	}
	err := s.repo.SetLocation(ctx, userID, RoundCoordinate(lat), RoundCoordinate(lng))
	if errors.Is(err, ErrNotFound) {
		return ErrProfileRequired
	}
	return err
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return s.repo.ClearLocation(ctx, userID)
}

func (s *Service) SetVisibility(ctx context.Context, userID string, req VisibilityRequest) error {
	err := s.repo.SetVisibility(ctx, userID, req.Discoverable, req.ShowDistance)
	if errors.Is(err, ErrNotFound) {
		return ErrProfileRequired
	}
	return err
}

func (s *Service) TouchActive(ctx context.Context, userID string) error {
	return s.repo.TouchActive(ctx, userID)
}

// View returns the public card of target when the viewer is allowed to see it.
func (s *Service) View(ctx context.Context, viewerID, targetID string) (Card, error) {
	if viewerID == targetID {
		return Card{}, ErrForbidden
	}
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return Card{}, err
	}
	if !ok {
		return Card{}, ErrForbidden
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return Card{}, err
	}
	card, found := cards[targetID]
	if !found {
		return Card{}, ErrForbidden
	}
	return card, nil
}

// Cards builds public cards for many users with a constant number of queries
// (profiles, photos, interests) regardless of how many ids are requested.
// Authorization is the caller's responsibility.
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]Card, error) {
	out := make(map[string]Card, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	ids := append([]string{viewerID}, userIDs...)
	profiles, err := s.repo.ListProfiles(ctx, ids)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.ListPhotos(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.ListInterestsForUsers(ctx, ids)
	if err != nil {
		return nil, err
	}

	var viewer *Profile
	for i := range profiles {
		if profiles[i].UserID == viewerID {
			viewer = &profiles[i]
		}
	}
	viewerInterests := map[string]bool{}
	for _, i := range interests[viewerID] {
		viewerInterests[i.Slug] = true
	}
	photosByUser := map[string][]Photo{}
	for _, p := range photos {
		photosByUser[p.UserID] = append(photosByUser[p.UserID], p)
	}

	now := s.now()
	for _, p := range profiles {
		if p.UserID == viewerID {
			continue
		}
		card := Card{
			UserID: p.UserID, FirstName: p.FirstName, Age: AgeOn(p.Birthdate, now), Gender: p.Gender,
			Bio: p.Bio, JobTitle: p.JobTitle, City: p.City,
			Interests: make([]CardInterest, 0, len(interests[p.UserID])),
			Photos:    s.photoResponses(photosByUser[p.UserID]),
		}
		for _, i := range interests[p.UserID] {
			card.Interests = append(card.Interests, CardInterest{Slug: i.Slug, Label: i.Label, Shared: viewerInterests[i.Slug]})
		}
		if p.ShowDistance && viewer != nil && viewer.Latitude != nil && p.Latitude != nil {
			d := DisplayDistanceKm(HaversineKm(*viewer.Latitude, *viewer.Longitude, *p.Latitude, *p.Longitude))
			card.DistanceKm = &d
		}
		out[p.UserID] = card
	}
	return out, nil
}

func (s *Service) photoResponses(photos []Photo) []PhotoResponse {
	out := make([]PhotoResponse, 0, len(photos))
	for _, p := range photos {
		out = append(out, PhotoResponse{ID: p.ID, URL: s.signer.URL(p.StorageKey), Position: p.Position, Width: p.Width, Height: p.Height})
	}
	return out
}
