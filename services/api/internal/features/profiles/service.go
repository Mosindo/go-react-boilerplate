package profiles

import (
	"context"
	"errors"

	"example.com/api/internal/platform/urlsign"
)

type Service struct {
	repo   Repository
	signer *urlsign.Signer
}

func NewService(repo Repository, signer *urlsign.Signer) *Service {
	return &Service{repo: repo, signer: signer}
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) photos(p *storedProfile) []Photo {
	out := make([]Photo, 0, len(p.Photos))
	for _, ph := range p.Photos {
		out = append(out, Photo{ID: ph.ID, URL: s.signer.PhotoURL(ph.ID), Position: ph.Position})
	}
	return out
}

func (s *Service) toMine(p *storedProfile) MyProfile {
	hasLoc := p.Lat != nil && p.Lng != nil
	return MyProfile{
		PublicProfile: PublicProfile{
			UserID: p.UserID, FirstName: p.FirstName, Age: p.Age, Gender: p.Gender,
			Bio: p.Bio, City: p.City, DistanceKm: nil, Interests: p.Interests, Photos: s.photos(p),
		},
		HasLocation:  hasLoc,
		ShowDistance: p.ShowDistance,
		ShowAge:      p.ShowAge,
		Discoverable: p.Discoverable,
		IsComplete:   hasLoc && len(p.Photos) > 0,
	}
}

func (s *Service) Own(ctx context.Context, userID string) (MyProfile, error) {
	p, err := s.repo.Own(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}
	return s.toMine(p), nil
}

func (s *Service) Save(ctx context.Context, userID string, req ProfileRequest) (MyProfile, error) {
	in, err := ValidateProfile(req)
	if err != nil {
		return MyProfile{}, err
	}
	if err := s.repo.Upsert(ctx, userID, in); err != nil {
		if errors.Is(err, ErrUnknownInterest) {
			return MyProfile{}, invalid("unknown interest")
		}
		return MyProfile{}, err
	}
	return s.Own(ctx, userID)
}

func (s *Service) SetLocation(ctx context.Context, userID string, req LocationRequest) error {
	lat, lng, err := ValidateLocation(req)
	if err != nil {
		return err
	}
	return s.repo.SetLocation(ctx, userID, lat, lng)
}

func (s *Service) Preferences(ctx context.Context, userID string) (Preferences, error) {
	return s.repo.GetPreferences(ctx, userID)
}

func (s *Service) SavePreferences(ctx context.Context, userID string, req PreferencesRequest) (Preferences, error) {
	p, err := ValidatePreferences(req)
	if err != nil {
		return Preferences{}, err
	}
	return s.repo.SetPreferences(ctx, userID, p)
}

// Public returns another user's profile if the viewer may see it (ErrNotFound otherwise).
func (s *Service) Public(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	p, err := s.repo.Visible(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	pub := PublicProfile{
		UserID: p.UserID, FirstName: p.FirstName, Gender: p.Gender, Bio: p.Bio, City: p.City,
		Interests: p.Interests, Photos: s.photos(p),
	}
	self := viewerID == targetID
	if self || p.ShowAge {
		pub.Age = p.Age
	}
	if !self && p.ShowDistance && p.Lat != nil && p.Lng != nil && p.ViewerLat != nil && p.ViewerLng != nil {
		km := BucketKm(DistanceKm(*p.ViewerLat, *p.ViewerLng, *p.Lat, *p.Lng))
		pub.DistanceKm = &km
	}
	return pub, nil
}
