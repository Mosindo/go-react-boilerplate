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

	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/media"
)

var (
	ErrProfileNotFound    = errors.New("profile not found")
	ErrInvalidFirstName   = errors.New("first name must be 1 to 30 letters")
	ErrInvalidBirthdate   = errors.New("invalid birthdate")
	ErrUnderage           = errors.New("you must be at least 18 years old")
	ErrBirthdateLocked    = errors.New("birthdate cannot be changed once set")
	ErrInvalidGender      = errors.New("invalid gender")
	ErrInvalidGoal        = errors.New("invalid relationship goal")
	ErrInvalidHeight      = errors.New("height must be between 120 and 230 cm")
	ErrTooLong            = errors.New("a text field is too long")
	ErrTooManyInterests   = errors.New("at most 8 interests")
	ErrUnknownInterest    = errors.New("unknown interest")
	ErrInvalidLocation    = errors.New("invalid coordinates")
	ErrInvalidPreferences = errors.New("invalid preferences")
	ErrNotVisible         = errors.New("profile not available")
)

type Service struct {
	repo   Repository
	signer *media.Signer
	now    func() time.Time
}

func NewService(repo Repository, signer *media.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

// AgeAt returns the age in full years at the given instant.
func AgeAt(birthdate, now time.Time) int {
	years := now.Year() - birthdate.Year()
	if now.Month() < birthdate.Month() || (now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
		years--
	}
	return years
}

// OwnProfile bundles everything the owner sees about their account.
type OwnProfile struct {
	Profile     *Profile
	Preferences *Preferences
	Photos      []Photo
	Interests   []Interest
	Missing     []string
}

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	out := OwnProfile{}
	p, err := s.repo.GetProfile(ctx, userID)
	switch {
	case err == nil:
		out.Profile = &p
	case !errors.Is(err, ErrRepoNotFound):
		return OwnProfile{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	switch {
	case err == nil:
		out.Preferences = &prefs
	case !errors.Is(err, ErrRepoNotFound):
		return OwnProfile{}, err
	}
	if out.Profile != nil {
		if out.Photos, err = s.repo.ListPhotos(ctx, userID); err != nil {
			return OwnProfile{}, err
		}
		if out.Interests, err = s.repo.ListUserInterests(ctx, userID); err != nil {
			return OwnProfile{}, err
		}
		_ = s.repo.TouchActivity(ctx, userID)
	}
	out.Missing = Missing(out.Profile, out.Preferences, len(out.Photos))
	return out, nil
}

// Missing lists the onboarding steps still required before discovery.
func Missing(p *Profile, prefs *Preferences, photoCount int) []string {
	missing := []string{}
	if p == nil {
		missing = append(missing, "profile")
	}
	if prefs == nil {
		missing = append(missing, "preferences")
	}
	if photoCount == 0 {
		missing = append(missing, "photos")
	}
	if p == nil || p.Latitude == nil {
		missing = append(missing, "location")
	}
	return missing
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) error {
	in, err := s.ValidateProfile(req)
	if err != nil {
		return err
	}
	existing, err := s.repo.GetProfile(ctx, userID)
	if err == nil && !existing.Birthdate.Equal(in.Birthdate) {
		return ErrBirthdateLocked
	}
	if err != nil && !errors.Is(err, ErrRepoNotFound) {
		return err
	}
	if err := s.repo.UpsertProfile(ctx, userID, in); err != nil {
		if errors.Is(err, ErrRepoUnknownInterest) {
			return ErrUnknownInterest
		}
		return err
	}
	return nil
}

// ValidateProfile normalizes and validates every field server side.
func (s *Service) ValidateProfile(req UpsertProfileRequest) (ProfileInput, error) {
	name := collapseSpaces(req.FirstName)
	if !validName(name) {
		return ProfileInput{}, ErrInvalidFirstName
	}
	birthdate, err := time.Parse("2006-01-02", strings.TrimSpace(req.Birthdate))
	if err != nil {
		return ProfileInput{}, ErrInvalidBirthdate
	}
	age := AgeAt(birthdate, s.now())
	if age < MinAge {
		return ProfileInput{}, ErrUnderage
	}
	if age > 110 {
		return ProfileInput{}, ErrInvalidBirthdate
	}
	if !slices.Contains(Genders, req.Gender) {
		return ProfileInput{}, ErrInvalidGender
	}
	bio := strings.TrimSpace(req.Bio)
	job := collapseSpaces(req.JobTitle)
	city := collapseSpaces(req.City)
	if utf8.RuneCountInString(bio) > MaxBioRunes || utf8.RuneCountInString(job) > MaxJobRunes || utf8.RuneCountInString(city) > MaxCityRunes {
		return ProfileInput{}, ErrTooLong
	}
	if req.HeightCm != nil && (*req.HeightCm < 120 || *req.HeightCm > 230) {
		return ProfileInput{}, ErrInvalidHeight
	}
	var goal *string
	if req.RelationshipGoal != nil && *req.RelationshipGoal != "" {
		if !slices.Contains(RelationshipGoals, *req.RelationshipGoal) {
			return ProfileInput{}, ErrInvalidGoal
		}
		g := *req.RelationshipGoal
		goal = &g
	}
	interests := uniqueInts(req.InterestIDs)
	if len(interests) > MaxInterests {
		return ProfileInput{}, ErrTooManyInterests
	}
	for _, id := range interests {
		if id <= 0 || id > math.MaxInt16 {
			return ProfileInput{}, ErrUnknownInterest
		}
	}
	return ProfileInput{
		FirstName: name, Birthdate: birthdate, Gender: req.Gender, Bio: bio, JobTitle: job, City: city,
		HeightCm: req.HeightCm, RelationshipGoal: goal, InterestIDs: interests,
	}, nil
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lon float64, city *string) error {
	if math.IsNaN(lat) || math.IsNaN(lon) || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return ErrInvalidLocation
	}
	var normalizedCity *string
	if city != nil {
		c := collapseSpaces(*city)
		if utf8.RuneCountInString(c) > MaxCityRunes {
			return ErrTooLong
		}
		if c != "" {
			normalizedCity = &c
		}
	}
	// Only the coarse position is ever persisted.
	err := s.repo.UpdateLocation(ctx, userID, geo.Coarsen(lat), geo.Coarsen(lon), normalizedCity)
	if errors.Is(err, ErrRepoNotFound) {
		return ErrProfileNotFound
	}
	return err
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req PreferencesRequest) error {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return err
	}
	err = s.repo.UpsertPreferences(ctx, userID, prefs)
	if errors.Is(err, ErrRepoNotFound) {
		return ErrProfileNotFound
	}
	return err
}

func ValidatePreferences(req PreferencesRequest) (Preferences, error) {
	genders := []string{}
	for _, g := range req.InterestedIn {
		if !slices.Contains(Genders, g) {
			return Preferences{}, ErrInvalidPreferences
		}
		if !slices.Contains(genders, g) {
			genders = append(genders, g)
		}
	}
	if len(genders) == 0 {
		return Preferences{}, ErrInvalidPreferences
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, ErrInvalidPreferences
	}
	if req.MaxDistanceKm < MinDistanceKm || req.MaxDistanceKm > MaxDistanceKm {
		return Preferences{}, ErrInvalidPreferences
	}
	slices.Sort(genders)
	return Preferences{InterestedIn: genders, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}

func (s *Service) UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error {
	err := s.repo.UpdatePrivacy(ctx, userID, discoverable, showDistance)
	if errors.Is(err, ErrRepoNotFound) {
		return ErrProfileNotFound
	}
	return err
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// View returns someone else's public profile if the viewer may see it.
func (s *Service) View(ctx context.Context, viewerID, targetID string) (Card, error) {
	if viewerID == targetID {
		return Card{}, ErrNotVisible
	}
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return Card{}, err
	}
	if !ok {
		return Card{}, ErrNotVisible
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return Card{}, err
	}
	if len(cards) == 0 {
		return Card{}, ErrNotVisible
	}
	return cards[0], nil
}

// Cards loads public cards for already-authorized ids, preserving order and
// computing the approximate distance from the viewer.
func (s *Service) Cards(ctx context.Context, viewerID string, ids []string) ([]Card, error) {
	if len(ids) == 0 {
		return []Card{}, nil
	}
	var viewerLat, viewerLon *float64
	if viewer, err := s.repo.GetProfile(ctx, viewerID); err == nil {
		viewerLat, viewerLon = viewer.Latitude, viewer.Longitude
	} else if !errors.Is(err, ErrRepoNotFound) {
		return nil, err
	}

	stored, err := s.repo.GetCards(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]StoredCard, len(stored))
	for _, c := range stored {
		byID[c.Profile.UserID] = c
	}
	now := s.now()
	out := make([]Card, 0, len(ids))
	for _, id := range ids {
		c, ok := byID[id]
		if !ok {
			continue
		}
		p := c.Profile
		card := Card{
			UserID: p.UserID, FirstName: p.FirstName, Age: AgeAt(p.Birthdate, now), Gender: p.Gender,
			Bio: p.Bio, JobTitle: p.JobTitle, City: p.City, HeightCm: p.HeightCm,
			RelationshipGoal: p.RelationshipGoal, Photos: c.Photos, Interests: c.Interests,
		}
		if card.Interests == nil {
			card.Interests = []Interest{}
		}
		if p.ShowDistance && viewerLat != nil && p.Latitude != nil {
			km := geo.DisplayKm(geo.DistanceKm(*viewerLat, *viewerLon, *p.Latitude, *p.Longitude))
			card.DistanceKm = &km
		}
		out = append(out, card)
	}
	return out, nil
}

// PhotoURL signs a storage key for display.
func (s *Service) PhotoURL(key string) string { return s.signer.URL(key) }

func (s *Service) ToPhotoResponses(photos []Photo) []PhotoResponse {
	out := make([]PhotoResponse, 0, len(photos))
	for _, p := range photos {
		out = append(out, PhotoResponse{ID: p.ID, URL: s.signer.URL(p.StorageKey), Width: p.Width, Height: p.Height})
	}
	return out
}

func (s *Service) ToCardResponse(c Card) CardResponse {
	return CardResponse{
		UserID: c.UserID, FirstName: c.FirstName, Age: c.Age, Gender: c.Gender, Bio: c.Bio,
		JobTitle: c.JobTitle, City: c.City, HeightCm: c.HeightCm, RelationshipGoal: c.RelationshipGoal,
		DistanceKm: c.DistanceKm, Photos: s.ToPhotoResponses(c.Photos), Interests: c.Interests,
	}
}

func validName(name string) bool {
	n := utf8.RuneCountInString(name)
	if n < 1 || n > MaxNameRunes {
		return false
	}
	letters := 0
	for _, r := range name {
		switch {
		case unicode.IsLetter(r):
			letters++
		case r == ' ' || r == '-' || r == '\'' || r == '’' || r == '.':
		default:
			return false
		}
	}
	return letters > 0
}

func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func uniqueInts(values []int) []int {
	out := make([]int, 0, len(values))
	for _, v := range values {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}
