package profiles

import (
	"context"
	"errors"
	"time"
)

var ErrNotVisible = errors.New("profile not available")

// URLSigner produces protected photo URLs.
type URLSigner interface {
	Sign(photoID string) string
}

// MaxPhotos is the number of photo slots per profile.
const MaxPhotos = 6

type Service struct {
	repo   Repository
	signer URLSigner
	now    func() time.Time
}

func NewService(repo Repository, signer URLSigner) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) today() time.Time {
	return s.now().UTC()
}

func (s *Service) photos(rows []PhotoRow) []Photo {
	out := make([]Photo, 0, len(rows))
	for _, row := range rows {
		out = append(out, Photo{ID: row.ID, Position: row.Position, URL: s.signer.Sign(row.ID)})
	}
	return out
}

// GetMine returns the owner's profile with completeness information.
func (s *Service) GetMine(ctx context.Context, userID string) (MyProfile, error) {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}
	interests, err := s.repo.ListUserInterests(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}
	photoRows, err := s.repo.ListUserPhotos(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}

	missing := MissingFields(p, prefs, len(photoRows))
	out := MyProfile{
		UserID:       p.UserID,
		Bio:          p.Bio,
		HasLocation:  p.Latitude != nil,
		Interests:    interests,
		Photos:       s.photos(photoRows),
		Preferences:  prefs,
		Settings:     Settings{IsDiscoverable: p.IsDiscoverable, HideDistance: p.HideDistance},
		Missing:      missing,
		IsComplete:   len(missing) == 0,
		MaxPhotos:    MaxPhotos,
		MaxInterests: MaxInterests,
	}
	if p.FirstName != nil {
		out.FirstName = *p.FirstName
	}
	if p.Gender != nil {
		out.Gender = *p.Gender
	}
	if p.City != nil {
		out.City = *p.City
	}
	if p.Birthdate != nil {
		formatted := p.Birthdate.Format("2006-01-02")
		age := AgeOn(*p.Birthdate, s.today())
		out.Birthdate = &formatted
		out.Age = &age
	}
	if out.Interests == nil {
		out.Interests = []Interest{}
	}
	return out, nil
}

// UpdateMine validates and applies a partial profile update.
func (s *Service) UpdateMine(ctx context.Context, userID string, req UpdateProfileRequest) (MyProfile, error) {
	var u ProfileUpdate
	if req.FirstName != nil {
		name, err := NormalizeFirstName(*req.FirstName)
		if err != nil {
			return MyProfile{}, err
		}
		u.FirstName = &name
	}
	if req.Birthdate != nil {
		b, err := ParseBirthdate(*req.Birthdate, s.today())
		if err != nil {
			return MyProfile{}, err
		}
		u.Birthdate = &b
	}
	if req.Gender != nil {
		if !ValidGender(*req.Gender) {
			return MyProfile{}, ErrInvalidGender
		}
		g := *req.Gender
		u.Gender = &g
	}
	if req.Bio != nil {
		bio, err := NormalizeBio(*req.Bio)
		if err != nil {
			return MyProfile{}, err
		}
		u.Bio = &bio
	}
	if req.City != nil {
		city, err := NormalizeCity(*req.City)
		if err != nil {
			return MyProfile{}, err
		}
		u.City = &city
	}
	if req.Interests != nil {
		slugs, err := NormalizeInterests(*req.Interests)
		if err != nil {
			return MyProfile{}, err
		}
		u.Interests = &slugs
	}
	if err := s.repo.UpdateProfile(ctx, userID, u); err != nil {
		return MyProfile{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (MyProfile, error) {
	prefs, err := NormalizePreferences(req)
	if err != nil {
		return MyProfile{}, err
	}
	if err := s.repo.UpdatePreferences(ctx, userID, prefs); err != nil {
		return MyProfile{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) UpdateSettings(ctx context.Context, userID string, settings Settings) (MyProfile, error) {
	if err := s.repo.UpdateSettings(ctx, userID, settings); err != nil {
		return MyProfile{}, err
	}
	return s.GetMine(ctx, userID)
}

// UpdateLocation stores an approximate position (snapped to ~1 km).
func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lon float64, city *string) (MyProfile, error) {
	if err := ValidateCoordinates(lat, lon); err != nil {
		return MyProfile{}, err
	}
	snappedLat, snappedLon := SnapCoordinate(lat), SnapCoordinate(lon)
	var normalizedCity *string
	if city != nil {
		c, err := NormalizeCity(*city)
		if err != nil {
			return MyProfile{}, err
		}
		normalizedCity = &c
	}
	if err := s.repo.UpdateLocation(ctx, userID, &snappedLat, &snappedLon, normalizedCity); err != nil {
		return MyProfile{}, err
	}
	return s.GetMine(ctx, userID)
}

// ClearLocation forgets the stored position entirely.
func (s *Service) ClearLocation(ctx context.Context, userID string) (MyProfile, error) {
	if err := s.repo.UpdateLocation(ctx, userID, nil, nil, nil); err != nil {
		return MyProfile{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// GetPublic returns target's public profile if viewer is allowed to see it.
func (s *Service) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	if viewerID != targetID {
		ok, err := s.repo.CanView(ctx, viewerID, targetID)
		if err != nil {
			return PublicProfile{}, err
		}
		if !ok {
			return PublicProfile{}, ErrNotVisible
		}
	}
	cards, err := s.LoadCards(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	if len(cards) == 0 {
		return PublicProfile{}, ErrNotVisible
	}
	return cards[0], nil
}

// LoadCards builds public profiles in the order of userIDs. Callers are
// responsible for authorization (discovery and match lists already filter).
func (s *Service) LoadCards(ctx context.Context, viewerID string, userIDs []string) ([]PublicProfile, error) {
	cards, photos, interests, err := s.repo.LoadCards(ctx, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	photosByUser := make(map[string][]PhotoRow, len(userIDs))
	for _, p := range photos {
		photosByUser[p.UserID] = append(photosByUser[p.UserID], p)
	}
	interestsByUser := make(map[string][]Interest, len(userIDs))
	for _, i := range interests {
		interestsByUser[i.UserID] = append(interestsByUser[i.UserID], i.Interest)
	}
	byID := make(map[string]PublicProfile, len(cards))
	today := s.today()
	for _, c := range cards {
		profile := PublicProfile{
			UserID:    c.UserID,
			FirstName: c.FirstName,
			Age:       AgeOn(c.Birthdate, today),
			Gender:    c.Gender,
			Bio:       c.Bio,
			City:      c.City,
			Interests: interestsByUser[c.UserID],
			Photos:    s.photos(photosByUser[c.UserID]),
		}
		if profile.Interests == nil {
			profile.Interests = []Interest{}
		}
		if c.DistanceKm != nil && !c.HideDistance && c.UserID != viewerID {
			d := ApproximateDistance(*c.DistanceKm)
			profile.DistanceKm = &d
		}
		byID[c.UserID] = profile
	}
	ordered := make([]PublicProfile, 0, len(byID))
	for _, id := range userIDs {
		if p, ok := byID[id]; ok {
			ordered = append(ordered, p)
		}
	}
	return ordered, nil
}

// FirstName implements chat.NameResolver.
func (s *Service) FirstName(ctx context.Context, userID string) string {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil || p.FirstName == nil {
		return ""
	}
	return *p.FirstName
}
