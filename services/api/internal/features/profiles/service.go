package profiles

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/media"
)

var ErrForbidden = errors.New("profile not visible")

type Service struct {
	repo   Repository
	signer *media.Signer
	now    func() time.Time
}

func NewService(repo Repository, signer *media.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) toPhotos(stored []StoredPhoto) []Photo {
	photos := make([]Photo, 0, len(stored))
	for _, p := range stored {
		photos = append(photos, Photo{ID: p.ID, URL: s.signer.PhotoPath(p.ID), Position: p.Position, Width: p.Width, Height: p.Height})
	}
	return photos
}

// Mine returns the owner's profile, preferences and photos.
func (s *Service) Mine(ctx context.Context, userID string) (MyProfileResponse, error) {
	stored, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return MyProfileResponse{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return MyProfileResponse{}, err
	}
	photosByUser, err := s.repo.PhotosByUsers(ctx, []string{userID})
	if err != nil {
		return MyProfileResponse{}, err
	}
	resp := MyProfileResponse{Preferences: prefs, Photos: s.toPhotos(photosByUser[userID])}
	if stored != nil {
		interests, err := s.repo.InterestsByUsers(ctx, []string{userID})
		if err != nil {
			return MyProfileResponse{}, err
		}
		p := stored.Profile
		p.Birthdate = stored.birthdate.Format(birthdateLayout)
		p.Age = AgeAt(stored.birthdate, s.now())
		p.Interests = interests[userID]
		if p.Interests == nil {
			p.Interests = []Interest{}
		}
		resp.Profile = &p
	}
	resp.Complete = resp.Profile != nil && len(resp.Photos) > 0
	return resp, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (MyProfileResponse, error) {
	in, err := ValidateProfile(req, s.now())
	if err != nil {
		return MyProfileResponse{}, err
	}
	if err := s.repo.UpsertProfile(ctx, userID, in); err != nil {
		if errors.Is(err, ErrUnknownInterest) {
			return MyProfileResponse{}, invalid("invalid_interests", "centre d'intérêt inconnu")
		}
		return MyProfileResponse{}, err
	}
	return s.Mine(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (Preferences, error) {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return Preferences{}, err
	}
	if err := s.repo.UpdatePreferences(ctx, userID, prefs); err != nil {
		return Preferences{}, err
	}
	return prefs, nil
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, req UpdateLocationRequest) error {
	city, ok := cleanText(req.City, maxCityRunes, false)
	if !ok {
		return invalid("invalid_city", "ville trop longue")
	}
	if (req.Latitude == nil) != (req.Longitude == nil) {
		return invalid("invalid_location", "coordonnées incomplètes")
	}
	if req.Latitude == nil {
		return s.repo.UpdateLocation(ctx, userID, nil, nil, city)
	}
	lat, lng, err := CoarsenLocation(*req.Latitude, *req.Longitude)
	if err != nil {
		return invalid("invalid_location", "coordonnées invalides")
	}
	return s.repo.UpdateLocation(ctx, userID, &lat, &lng, city)
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return s.repo.UpdateLocation(ctx, userID, nil, nil, "")
}

func (s *Service) UpdatePrivacy(ctx context.Context, userID string, req UpdatePrivacyRequest) error {
	return s.repo.UpdatePrivacy(ctx, userID, req.Discoverable, req.ShowDistance)
}

// Cards hydrates public profiles for many users in three queries (no N+1),
// preserving the order of userIDs and skipping users without a profile.
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) ([]PublicProfile, error) {
	if len(userIDs) == 0 {
		return []PublicProfile{}, nil
	}
	if len(userIDs) > maxProfilesPerCall {
		userIDs = userIDs[:maxProfilesPerCall]
	}
	cards, err := s.repo.Cards(ctx, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.PhotosByUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.InterestsByUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := make([]PublicProfile, 0, len(userIDs))
	for _, id := range userIDs {
		card, ok := cards[id]
		if !ok {
			continue
		}
		p := card.PublicProfile
		if card.showDistance && card.rawDistance != nil {
			d := ApproximateDistance(*card.rawDistance)
			p.DistanceKm = &d
		}
		p.Photos = s.toPhotos(photos[id])
		p.Interests = interests[id]
		if p.Interests == nil {
			p.Interests = []Interest{}
		}
		out = append(out, p)
	}
	return out, nil
}

// CardsByID is Cards indexed by user id.
func (s *Service) CardsByID(ctx context.Context, viewerID string, userIDs []string) (map[string]PublicProfile, error) {
	list, err := s.Cards(ctx, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]PublicProfile, len(list))
	for _, p := range list {
		out[p.UserID] = p
	}
	return out, nil
}

func (s *Service) View(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if !ok {
		return PublicProfile{}, ErrForbidden
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	if len(cards) == 0 {
		return PublicProfile{}, ErrForbidden
	}
	return cards[0], nil
}
