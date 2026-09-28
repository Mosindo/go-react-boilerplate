package profiles

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/validate"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) Me(ctx context.Context, userID string) (Me, error) {
	u, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Me{}, httpx.Unauthorized("account no longer exists")
		}
		return Me{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return Me{}, err
	}
	me := Me{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt, Preferences: prefs}
	profile, err := s.loadProfile(ctx, userID)
	if err != nil {
		return Me{}, err
	}
	me.Profile = profile
	me.ProfileComplete = profile != nil && profile.HasLocation && len(profile.Photos) >= 1
	return me, nil
}

// loadProfile returns nil when the user has no profile yet.
func (s *Service) loadProfile(ctx context.Context, userID string) (*Profile, error) {
	rec, err := s.repo.GetProfile(ctx, userID)
	if err != nil || rec == nil {
		return nil, err
	}
	ids := []string{userID}
	interests, err := s.repo.InterestsFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.PhotosFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	p := &Profile{
		UserID:         rec.UserID,
		FirstName:      rec.FirstName,
		Age:            validate.AgeOn(rec.BirthDate, s.now()),
		BirthDate:      rec.BirthDate.Format("2006-01-02"),
		Gender:         rec.Gender,
		Bio:            rec.Bio,
		LocationLabel:  rec.LocationLabel,
		HasLocation:    rec.HasLocation,
		ShowDistance:   rec.ShowDistance,
		IsDiscoverable: rec.IsDiscoverable,
		Interests:      interests[userID],
		Photos:         photos[userID],
	}
	if p.Interests == nil {
		p.Interests = []Interest{}
	}
	if p.Photos == nil {
		p.Photos = []Photo{}
	}
	return p, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*Profile, error) {
	name, err := validate.Text(req.FirstName, 1, MaxFirstName, false)
	if err != nil {
		return nil, httpx.BadRequest("firstName must be 1 to 50 characters without control characters")
	}
	bio, err := validate.Text(req.Bio, 0, MaxBio, true)
	if err != nil {
		return nil, httpx.BadRequest("bio must be at most 500 characters")
	}
	if !ValidGender(req.Gender) {
		return nil, httpx.BadRequest("gender must be one of man, woman, non_binary")
	}
	birth, err := validate.ParseBirthDate(req.BirthDate)
	if err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	now := s.now()
	if birth.After(now) || birth.Year() < 1900 {
		return nil, httpx.BadRequest("birthDate is not a plausible date")
	}
	if !validate.IsAdult(birth, now) {
		return nil, httpx.Underage()
	}
	if len(req.InterestIDs) > MaxInterests {
		return nil, httpx.BadRequest("at most 10 interests are allowed")
	}
	seen := make(map[int16]struct{}, len(req.InterestIDs))
	for _, id := range req.InterestIDs {
		if id < 1 {
			return nil, httpx.BadRequest("invalid interest id")
		}
		if _, dup := seen[id]; dup {
			return nil, httpx.BadRequest("interestIds must be unique")
		}
		seen[id] = struct{}{}
	}

	err = s.repo.UpsertProfile(ctx, userID, ProfileInput{
		FirstName: name, BirthDate: birth, Gender: req.Gender, Bio: bio,
		InterestIDs: req.InterestIDs, ShowDistance: req.ShowDistance, IsDiscoverable: req.IsDiscoverable,
	})
	switch {
	case errors.Is(err, ErrBirthDateChange):
		return nil, httpx.Unprocessable("birthDate cannot be changed once set")
	case errors.Is(err, ErrUnknownInterest):
		return nil, httpx.BadRequest("unknown interest id")
	case err != nil:
		return nil, err
	}
	return s.loadProfile(ctx, userID)
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, req UpdateLocationRequest) (*Profile, error) {
	if req.Latitude == nil || req.Longitude == nil {
		return nil, httpx.BadRequest("latitude and longitude are required")
	}
	if err := geo.ValidateCoordinates(*req.Latitude, *req.Longitude); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	var label *string
	if req.Label != nil {
		clean, err := validate.Text(*req.Label, 0, MaxLabel, false)
		if err != nil {
			return nil, httpx.BadRequest("label must be at most 80 characters")
		}
		label = &clean
	}
	// Precision policy: only ~1 km resolution is ever stored.
	lat, lon := geo.Round2(*req.Latitude), geo.Round2(*req.Longitude)
	if err := s.repo.SetLocation(ctx, userID, lat, lon, label); err != nil {
		if errors.Is(err, ErrNoProfile) {
			return nil, &httpx.Error{Status: 409, Code: httpx.CodeProfileIncomplete, Message: "create your profile before setting a location"}
		}
		return nil, err
	}
	return s.loadProfile(ctx, userID)
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	return s.repo.GetPreferences(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (Preferences, error) {
	if n := len(req.InterestedIn); n < 1 || n > 3 {
		return Preferences{}, httpx.BadRequest("interestedIn must contain 1 to 3 genders")
	}
	seen := map[string]struct{}{}
	for _, g := range req.InterestedIn {
		if !ValidGender(g) {
			return Preferences{}, httpx.BadRequest("interestedIn contains an unknown gender")
		}
		if _, dup := seen[g]; dup {
			return Preferences{}, httpx.BadRequest("interestedIn must not contain duplicates")
		}
		seen[g] = struct{}{}
	}
	if req.MinAge < 18 || req.MinAge > 99 {
		return Preferences{}, httpx.BadRequest("minAge must be between 18 and 99")
	}
	if req.MaxAge < req.MinAge || req.MaxAge > 99 {
		return Preferences{}, httpx.BadRequest("maxAge must be between minAge and 99")
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > 500 {
		return Preferences{}, httpx.BadRequest("maxDistanceKm must be between 1 and 500")
	}
	p := Preferences{InterestedIn: req.InterestedIn, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}
	if err := s.repo.UpsertPreferences(ctx, userID, p); err != nil {
		return Preferences{}, err
	}
	return p, nil
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}
