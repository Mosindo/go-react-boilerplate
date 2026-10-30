package profiles

import (
	"context"
	"time"
)

// PhotoURLs builds the signed URL clients use to display a photo.
type PhotoURLs interface {
	PhotoPath(photoID string) string
}

type Service struct {
	repo  Repository
	urls  PhotoURLs
	clock func() time.Time
}

func NewService(repo Repository, urls PhotoURLs) *Service {
	return &Service{repo: repo, urls: urls, clock: time.Now}
}

func (s *Service) PhotoRefs(ids []string) []PhotoRef {
	refs := make([]PhotoRef, 0, len(ids))
	for _, id := range ids {
		refs = append(refs, PhotoRef{ID: id, URL: s.urls.PhotoPath(id)})
	}
	return refs
}

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	stored, err := s.repo.GetOwn(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	photos, err := s.repo.PhotoIDs(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	interests, err := s.repo.Interests(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}

	missing := []string{}
	if len(photos[userID]) == 0 {
		missing = append(missing, "photos")
	}
	if len(stored.Preferences.InterestedIn) == 0 {
		missing = append(missing, "preferences")
	}
	if !stored.HasLocation {
		missing = append(missing, "location")
	}

	list := interests[userID]
	if list == nil {
		list = []Interest{}
	}
	return OwnProfile{
		UserID:         stored.UserID,
		DisplayName:    stored.DisplayName,
		BirthDate:      stored.BirthDate.Format("2006-01-02"),
		Age:            AgeOn(stored.BirthDate, s.clock()),
		Gender:         stored.Gender,
		Bio:            stored.Bio,
		City:           stored.City,
		HasLocation:    stored.HasLocation,
		IsDiscoverable: stored.IsDiscoverable,
		ShowDistance:   stored.ShowDistance,
		Interests:      list,
		Photos:         s.PhotoRefs(photos[userID]),
		Preferences:    stored.Preferences,
		IsComplete:     len(missing) == 0,
		Missing:        missing,
	}, nil
}

func (s *Service) Save(ctx context.Context, userID string, req SaveProfileRequest) (OwnProfile, error) {
	input, err := validateProfile(req, s.clock())
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.Save(ctx, userID, input); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	return s.repo.GetPreferences(ctx, userID)
}

func (s *Service) SavePreferences(ctx context.Context, userID string, req SavePreferencesRequest) (Preferences, error) {
	prefs, err := validatePreferences(req)
	if err != nil {
		return Preferences{}, err
	}
	if err := s.repo.SavePreferences(ctx, userID, prefs); err != nil {
		return Preferences{}, err
	}
	return prefs, nil
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterestCatalog(ctx)
}

// GetPublic returns another user's profile as the viewer is allowed to see it.
func (s *Service) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	stored, err := s.repo.GetPublic(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	photos, err := s.repo.PhotoIDs(ctx, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	interests, err := s.repo.Interests(ctx, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	return s.ToPublic(stored, photos[targetID], interests[targetID]), nil
}

// ToPublic assembles the outward-facing representation: age instead of birth
// date, and a coarse distance instead of coordinates.
func (s *Service) ToPublic(stored StoredPublic, photoIDs []string, interests []Interest) PublicProfile {
	var distance *int
	if stored.DistanceKm != nil {
		d := ApproxDistanceKm(*stored.DistanceKm)
		distance = &d
	}
	if interests == nil {
		interests = []Interest{}
	}
	return PublicProfile{
		UserID:      stored.UserID,
		DisplayName: stored.DisplayName,
		Age:         AgeOn(stored.BirthDate, s.clock()),
		Gender:      stored.Gender,
		Bio:         stored.Bio,
		City:        stored.City,
		DistanceKm:  distance,
		Interests:   interests,
		Photos:      s.PhotoRefs(photoIDs),
	}
}
