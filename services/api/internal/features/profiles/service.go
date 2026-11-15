package profiles

import (
	"context"
	"math"
	"slices"
	"time"
	"unicode"

	"example.com/api/internal/platform/apperr"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/textx"
)

var (
	ErrInvalidName        = apperr.BadRequest("invalid_first_name", "first name must be 1-40 letters")
	ErrInvalidBirthdate   = apperr.BadRequest("invalid_birthdate", "birthdate must be a valid date (YYYY-MM-DD)")
	ErrUnderage           = apperr.BadRequest("underage", "you must be at least 18 years old")
	ErrInvalidGender      = apperr.BadRequest("invalid_gender", "gender must be woman, man or nonbinary")
	ErrBioTooLong         = apperr.BadRequest("bio_too_long", "bio must be at most 500 characters")
	ErrJobTooLong         = apperr.BadRequest("job_title_too_long", "job title must be at most 60 characters")
	ErrCityTooLong        = apperr.BadRequest("city_too_long", "city must be at most 80 characters")
	ErrInvalidHeight      = apperr.BadRequest("invalid_height", "height must be between 120 and 230 cm")
	ErrTooManyInterests   = apperr.BadRequest("too_many_interests", "choose at most 10 interests")
	ErrUnknownInterest    = apperr.BadRequest("unknown_interest", "unknown interest")
	ErrInvalidLocation    = apperr.BadRequest("invalid_location", "invalid coordinates")
	ErrInvalidPreferences = apperr.BadRequest("invalid_preferences", "invalid discovery preferences")
	ErrProfileNotFound    = apperr.NotFound("profile_not_found", "profile not found")
	ErrBirthdateLocked    = apperr.BadRequest("birthdate_locked", "birthdate cannot be changed once set")
)

type Service struct {
	repo   Repository
	signer *storage.URLSigner
	now    func() time.Time
}

func NewService(repo Repository, signer *storage.URLSigner) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) GetOwn(ctx context.Context, userID string) (MeProfileResponse, error) {
	profile, err := s.repo.GetOwnProfile(ctx, userID)
	if err != nil {
		return MeProfileResponse{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return MeProfileResponse{}, err
	}
	return MeProfileResponse{
		Profile:      s.toOwnResponse(profile),
		Preferences:  prefs,
		Completeness: ComputeCompleteness(profile),
	}, nil
}

func (s *Service) UpdateOwn(ctx context.Context, userID string, req UpdateProfileRequest) (MeProfileResponse, error) {
	update, err := ValidateProfileUpdate(req, s.now())
	if err != nil {
		return MeProfileResponse{}, err
	}
	current, err := s.repo.GetOwnProfile(ctx, userID)
	if err != nil {
		return MeProfileResponse{}, err
	}
	// Birthdate gates access to an adults-only product: it is set once.
	if current.Birthdate != nil {
		if update.Birthdate != nil && !update.Birthdate.Equal(*current.Birthdate) {
			return MeProfileResponse{}, ErrBirthdateLocked
		}
		update.Birthdate = current.Birthdate
	}
	if current.Gender != nil && update.Gender == nil {
		update.Gender = current.Gender
	}
	if len(update.InterestIDs) > 0 {
		known, err := s.repo.CountInterests(ctx, update.InterestIDs)
		if err != nil {
			return MeProfileResponse{}, err
		}
		if known != len(update.InterestIDs) {
			return MeProfileResponse{}, ErrUnknownInterest
		}
	}
	if err := s.repo.UpdateProfile(ctx, userID, update); err != nil {
		return MeProfileResponse{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, req UpdateLocationRequest) (MeProfileResponse, error) {
	lat, lng, err := ApproximateCoordinates(*req.Latitude, *req.Longitude)
	if err != nil {
		return MeProfileResponse{}, err
	}
	city := textx.Clean(req.City, false)
	if textx.Len(city) > MaxCityRunes {
		return MeProfileResponse{}, ErrCityTooLong
	}
	if err := s.repo.UpdateLocation(ctx, userID, &lat, &lng, city); err != nil {
		return MeProfileResponse{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) ClearLocation(ctx context.Context, userID string) (MeProfileResponse, error) {
	if err := s.repo.UpdateLocation(ctx, userID, nil, nil, ""); err != nil {
		return MeProfileResponse{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (MeProfileResponse, error) {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return MeProfileResponse{}, err
	}
	if err := s.repo.UpdatePreferences(ctx, userID, prefs); err != nil {
		return MeProfileResponse{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdatePrivacy(ctx context.Context, userID string, req UpdatePrivacyRequest) (MeProfileResponse, error) {
	if err := s.repo.UpdatePrivacy(ctx, userID, *req.IsPaused, *req.HideDistance); err != nil {
		return MeProfileResponse{}, err
	}
	return s.GetOwn(ctx, userID)
}

// GetVisible returns another user's card when the viewer is allowed to see
// it: neither side blocked the other, and the target is discoverable or
// already matched with the viewer.
func (s *Service) GetVisible(ctx context.Context, viewerID, targetID string) (Card, error) {
	if viewerID == targetID {
		return Card{}, ErrProfileNotFound
	}
	visible, err := s.repo.IsVisibleTo(ctx, viewerID, targetID)
	if err != nil {
		return Card{}, err
	}
	if !visible {
		return Card{}, ErrProfileNotFound
	}
	cards, err := s.LoadCards(ctx, viewerID, []string{targetID})
	if err != nil {
		return Card{}, err
	}
	card, ok := cards[targetID]
	if !ok {
		return Card{}, ErrProfileNotFound
	}
	return card, nil
}

// LoadCards builds public cards for several users with a constant number of
// queries (profiles, photos, interests), whatever the page size.
func (s *Service) LoadCards(ctx context.Context, viewerID string, userIDs []string) (map[string]Card, error) {
	result := make(map[string]Card, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := s.repo.LoadCardRows(ctx, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.LoadPhotos(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.LoadInterests(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		card := Card{
			UserID:    row.UserID,
			FirstName: row.FirstName,
			Age:       row.Age,
			Gender:    row.Gender,
			Bio:       row.Bio,
			JobTitle:  row.JobTitle,
			HeightCm:  row.HeightCm,
			City:      row.City,
			IsDemo:    row.IsDemo,
			Interests: interests[row.UserID],
			Photos:    s.photoResponses(photos[row.UserID]),
		}
		if card.Interests == nil {
			card.Interests = []Interest{}
		}
		if row.DistanceKm != nil && !row.HideDistance {
			d := DisplayDistance(*row.DistanceKm)
			card.DistanceKm = &d
		}
		result[row.UserID] = card
	}
	return result, nil
}

func (s *Service) toOwnResponse(p OwnProfile) OwnProfileResponse {
	resp := OwnProfileResponse{
		UserID:       p.UserID,
		FirstName:    p.FirstName,
		Gender:       p.Gender,
		Bio:          p.Bio,
		JobTitle:     p.JobTitle,
		HeightCm:     p.HeightCm,
		City:         p.City,
		HasLocation:  p.HasLocation,
		IsPaused:     p.IsPaused,
		HideDistance: p.HideDistance,
		Interests:    p.Interests,
		Photos:       s.photoResponses(p.Photos),
	}
	if resp.Interests == nil {
		resp.Interests = []Interest{}
	}
	if p.Birthdate != nil {
		formatted := p.Birthdate.Format("2006-01-02")
		age := AgeOn(*p.Birthdate, s.now())
		resp.Birthdate = &formatted
		resp.Age = &age
	}
	return resp
}

func (s *Service) photoResponses(photos []Photo) []PhotoResponse {
	out := make([]PhotoResponse, 0, len(photos))
	for _, p := range photos {
		out = append(out, PhotoResponse{ID: p.ID, URL: s.signer.Sign(p.StorageKey), Width: p.Width, Height: p.Height, Position: p.Position})
	}
	return out
}

// ---------------------------------------------------------------------------
// Pure business rules (unit tested)
// ---------------------------------------------------------------------------

// AgeOn returns the age in full years at the given moment.
func AgeOn(birthdate, now time.Time) int {
	years := now.Year() - birthdate.Year()
	if now.Month() < birthdate.Month() || (now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
		years--
	}
	return years
}

func ValidateProfileUpdate(req UpdateProfileRequest, now time.Time) (ProfileUpdate, error) {
	out := ProfileUpdate{InterestIDs: []int{}}

	out.FirstName = textx.Clean(req.FirstName, false)
	if out.FirstName != "" {
		if textx.Len(out.FirstName) > MaxNameRunes || !validName(out.FirstName) {
			return ProfileUpdate{}, ErrInvalidName
		}
	}

	if req.Birthdate != "" {
		birthdate, err := time.Parse("2006-01-02", req.Birthdate)
		if err != nil {
			return ProfileUpdate{}, ErrInvalidBirthdate
		}
		age := AgeOn(birthdate, now)
		if age < MinAge {
			return ProfileUpdate{}, ErrUnderage
		}
		if age > 110 {
			return ProfileUpdate{}, ErrInvalidBirthdate
		}
		out.Birthdate = &birthdate
	}

	if req.Gender != "" {
		if !slices.Contains(Genders, req.Gender) {
			return ProfileUpdate{}, ErrInvalidGender
		}
		gender := req.Gender
		out.Gender = &gender
	}

	out.Bio = textx.Clean(req.Bio, true)
	if textx.Len(out.Bio) > MaxBioRunes {
		return ProfileUpdate{}, ErrBioTooLong
	}
	out.JobTitle = textx.Clean(req.JobTitle, false)
	if textx.Len(out.JobTitle) > MaxJobRunes {
		return ProfileUpdate{}, ErrJobTooLong
	}
	out.City = textx.Clean(req.City, false)
	if textx.Len(out.City) > MaxCityRunes {
		return ProfileUpdate{}, ErrCityTooLong
	}
	if req.HeightCm != nil {
		if *req.HeightCm < MinHeightCm || *req.HeightCm > MaxHeightCm {
			return ProfileUpdate{}, ErrInvalidHeight
		}
		h := *req.HeightCm
		out.HeightCm = &h
	}

	seen := make(map[int]struct{}, len(req.InterestIDs))
	for _, id := range req.InterestIDs {
		if id <= 0 {
			return ProfileUpdate{}, ErrUnknownInterest
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out.InterestIDs = append(out.InterestIDs, id)
	}
	if len(out.InterestIDs) > MaxInterests {
		return ProfileUpdate{}, ErrTooManyInterests
	}
	return out, nil
}

func validName(name string) bool {
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

func ValidatePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, ErrInvalidPreferences
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > MaxDistanceKm {
		return Preferences{}, ErrInvalidPreferences
	}
	genders := make([]string, 0, len(req.InterestedIn))
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
	return Preferences{InterestedIn: genders, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}

// ApproximateCoordinates validates and rounds coordinates to a ~1.1 km grid
// so the exact position is never stored.
func ApproximateCoordinates(lat, lng float64) (float64, float64, error) {
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, ErrInvalidLocation
	}
	return math.Round(lat*LocationDecimal) / LocationDecimal, math.Round(lng*LocationDecimal) / LocationDecimal, nil
}

// DisplayDistance rounds to whole kilometres with a floor of 1 km so the
// value cannot be used to pinpoint someone.
func DisplayDistance(km float64) int {
	d := int(math.Ceil(km))
	if d < 1 {
		return 1
	}
	return d
}

// ComputeCompleteness lists what is still needed before a profile can appear
// in discovery.
func ComputeCompleteness(p OwnProfile) Completeness {
	missing := []string{}
	if p.FirstName == "" {
		missing = append(missing, "firstName")
	}
	if p.Birthdate == nil {
		missing = append(missing, "birthdate")
	}
	if p.Gender == nil {
		missing = append(missing, "gender")
	}
	if len(p.Photos) == 0 {
		missing = append(missing, "photo")
	}
	return Completeness{IsComplete: len(missing) == 0, Missing: missing}
}
