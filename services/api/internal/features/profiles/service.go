package profiles

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/validate"
)

var (
	ErrProfileNotFound = platformerrors.ErrNotFound
)

// CardLoader hydrates public profile cards for a list of users in a fixed number of queries.
// Discovery, matches and chat use it so listings never trigger N+1 queries.
type CardLoader interface {
	LoadCards(ctx context.Context, viewerID string, ids []string) ([]Card, error)
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) LoadCards(ctx context.Context, viewerID string, ids []string) ([]Card, error) {
	if len(ids) == 0 {
		return []Card{}, nil
	}
	stored, err := s.repo.GetStoredMany(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.InterestsFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.PhotosFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Card, len(stored))
	for _, p := range stored {
		byID[p.UserID] = s.toCard(p, interests[p.UserID], photos[p.UserID])
	}
	out := make([]Card, 0, len(ids))
	for _, id := range ids {
		if c, ok := byID[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *Service) toCard(p storedProfile, interests []Interest, photos []PhotoRef) Card {
	if interests == nil {
		interests = []Interest{}
	}
	if photos == nil {
		photos = []PhotoRef{}
	}
	gender := ""
	if p.Gender != nil {
		gender = *p.Gender
	}
	return Card{
		UserID: p.UserID, FirstName: p.FirstName, Age: validate.Age(p.BirthDate, s.now()), Gender: gender,
		Bio: p.Bio, City: p.City, DistanceKm: p.DistanceKm, Interests: interests, Photos: photos,
	}
}

func (s *Service) GetMe(ctx context.Context, userID string) (Me, error) {
	stored, err := s.repo.GetStored(ctx, userID, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Me{}, ErrProfileNotFound
		}
		return Me{}, err
	}
	cards, err := s.LoadCards(ctx, userID, []string{userID})
	if err != nil || len(cards) == 0 {
		return Me{}, err
	}
	missing := []string{}
	if stored.Gender == nil {
		missing = append(missing, "gender")
	}
	if len(stored.InterestedIn) == 0 {
		missing = append(missing, "interestedIn")
	}
	if !stored.HasLocation {
		missing = append(missing, "location")
	}
	if stored.PhotoCount == 0 {
		missing = append(missing, "photos")
	}
	return Me{
		Card:         cards[0],
		BirthDate:    stored.BirthDate.Format("2006-01-02"),
		Discoverable: stored.Discoverable,
		ShowDistance: stored.ShowDistance,
		HasLocation:  stored.HasLocation,
		Complete:     len(missing) == 0,
		Missing:      missing,
	}, nil
}

// GetPublic returns another user's card, or 404 when they are blocked, hidden or unknown
// (the same answer in every case, so existence is not leaked).
func (s *Service) GetPublic(ctx context.Context, viewerID, targetID string) (Card, error) {
	if !isUUID(targetID) {
		return Card{}, ErrProfileNotFound
	}
	if viewerID == targetID {
		me, err := s.GetMe(ctx, viewerID)
		return me.Card, err
	}
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return Card{}, err
	}
	if !ok {
		return Card{}, ErrProfileNotFound
	}
	cards, err := s.LoadCards(ctx, viewerID, []string{targetID})
	if err != nil {
		return Card{}, err
	}
	if len(cards) == 0 {
		return Card{}, ErrProfileNotFound
	}
	return cards[0], nil
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (Me, error) {
	if req.FirstName != nil {
		v, ok := validate.CleanText(*req.FirstName, 40, false)
		if !ok || v == "" {
			return Me{}, platformerrors.Invalid("invalid first name")
		}
		req.FirstName = &v
	}
	if req.Gender != nil && !slices.Contains(Genders, *req.Gender) {
		return Me{}, platformerrors.Invalid("invalid gender")
	}
	if req.Bio != nil {
		v, ok := validate.CleanText(*req.Bio, 500, true)
		if !ok {
			return Me{}, platformerrors.Invalid("bio must be at most 500 characters")
		}
		req.Bio = &v
	}
	if req.City != nil {
		v, ok := validate.CleanText(*req.City, 80, false)
		if !ok {
			return Me{}, platformerrors.Invalid("invalid city")
		}
		req.City = &v
	}
	if req.Interests != nil {
		seen := map[string]bool{}
		clean := make([]string, 0, len(*req.Interests))
		for _, slug := range *req.Interests {
			slug = strings.ToLower(strings.TrimSpace(slug))
			if slug == "" || seen[slug] {
				continue
			}
			seen[slug] = true
			clean = append(clean, slug)
		}
		if len(clean) > MaxInterests {
			return Me{}, platformerrors.Invalid("at most 8 interests")
		}
		req.Interests = &clean
	}
	if err := s.repo.UpdateProfile(ctx, userID, req); err != nil {
		if strings.Contains(err.Error(), "unknown interest") {
			return Me{}, platformerrors.Invalid("unknown interest")
		}
		return Me{}, err
	}
	return s.GetMe(ctx, userID)
}

// UpdateLocation stores coordinates rounded to two decimals (~1 km): precise enough for
// "within N km" filtering, too coarse to locate a person or a home.
func (s *Service) UpdateLocation(ctx context.Context, userID string, req UpdateLocationRequest) (Me, error) {
	if math.IsNaN(req.Latitude) || math.IsNaN(req.Longitude) || req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		return Me{}, platformerrors.Invalid("invalid coordinates")
	}
	var city *string
	if req.City != nil {
		v, ok := validate.CleanText(*req.City, 80, false)
		if !ok {
			return Me{}, platformerrors.Invalid("invalid city")
		}
		city = &v
	}
	lat := math.Round(req.Latitude*100) / 100
	lng := math.Round(req.Longitude*100) / 100
	if err := s.repo.UpdateLocation(ctx, userID, lat, lng, city); err != nil {
		return Me{}, err
	}
	return s.GetMe(ctx, userID)
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	p, err := s.repo.GetPreferences(ctx, userID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return Preferences{}, ErrProfileNotFound
	}
	if p.InterestedIn == nil {
		p.InterestedIn = []string{}
	}
	return p, err
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (Preferences, error) {
	seen := map[string]bool{}
	clean := []string{}
	for _, g := range req.InterestedIn {
		if !slices.Contains(Genders, g) {
			return Preferences{}, platformerrors.Invalid("invalid gender preference")
		}
		if !seen[g] {
			seen[g] = true
			clean = append(clean, g)
		}
	}
	if len(clean) == 0 {
		return Preferences{}, platformerrors.Invalid("choose at least one gender")
	}
	if req.AgeMin < validate.MinAge || req.AgeMax > validate.MaxAge || req.AgeMin > req.AgeMax {
		return Preferences{}, platformerrors.Invalid("invalid age range (18-99)")
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > 500 {
		return Preferences{}, platformerrors.Invalid("distance must be between 1 and 500 km")
	}
	p := Preferences{InterestedIn: clean, AgeMin: req.AgeMin, AgeMax: req.AgeMax, MaxDistanceKm: req.MaxDistanceKm}
	if err := s.repo.UpdatePreferences(ctx, userID, p); err != nil {
		return Preferences{}, err
	}
	return p, nil
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
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

// IsUUID is exported for handlers of other features that take ids in the path.
func IsUUID(s string) bool { return isUUID(s) }

// CanView reports whether viewer may see target's profile and photos (see Repository.CanView).
func (s *Service) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	if viewerID == targetID {
		return true, nil
	}
	return s.repo.CanView(ctx, viewerID, targetID)
}

// IsComplete reports whether the user has everything required to take part in discovery.
func (s *Service) IsComplete(ctx context.Context, userID string) (bool, error) {
	p, err := s.repo.GetStored(ctx, userID, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return false, ErrProfileNotFound
		}
		return false, err
	}
	return p.Gender != nil && len(p.InterestedIn) > 0 && p.HasLocation && p.PhotoCount > 0, nil
}
