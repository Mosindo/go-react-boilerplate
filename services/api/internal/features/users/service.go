package users

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/signedurl"
)

type Service struct {
	repo   *Repository
	signer *signedurl.Signer
	now    func() time.Time
}

func NewService(repo *Repository, signer *signedurl.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) { return s.repo.Interests(ctx) }

func (s *Service) photos(refs []photoRef) []Photo {
	now := s.now()
	out := make([]Photo, 0, len(refs))
	for i, ref := range refs {
		out = append(out, Photo{ID: ref.ID, Position: ref.Position, IsPrimary: i == 0, URL: s.signer.PhotoURL(ref.ID, now)})
	}
	return out
}

func (s *Service) Own(ctx context.Context, userID string) (OwnProfile, error) {
	p, err := s.repo.GetStored(ctx, userID)
	if err != nil {
		if errors.Is(err, errRepoNotFound) {
			return OwnProfile{}, ErrProfileNotFound
		}
		return OwnProfile{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	interests, photos, err := s.repo.InterestMaps(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	out := OwnProfile{
		UserID: p.UserID, FirstName: p.FirstName, Age: p.Age, BirthDate: p.BirthDate.Format("2006-01-02"),
		Gender: p.Gender, Bio: p.Bio, City: p.City, Interests: interests[userID], Photos: s.photos(photos[userID]),
		Discoverable: p.Discoverable, ShowDistance: p.ShowDistance, HasLocation: p.HasLocation, Preferences: prefs,
	}
	if out.Interests == nil {
		out.Interests = []Interest{}
	}
	return out, nil
}

func (s *Service) SaveProfile(ctx context.Context, userID string, in ProfileInput) (OwnProfile, error) {
	clean, birth, err := ValidateProfile(in, s.now())
	if err != nil {
		return OwnProfile{}, err
	}
	existing, err := s.repo.GetStored(ctx, userID)
	switch {
	case err == nil:
		if !existing.BirthDate.Equal(birth) {
			return OwnProfile{}, ErrBirthDateLocked
		}
	case !errors.Is(err, errRepoNotFound):
		return OwnProfile{}, err
	}
	if err := s.repo.Upsert(ctx, userID, clean, birth.Format("2006-01-02")); err != nil {
		switch fkConstraint(err) {
		case "user_interests_interest_id_fkey":
			return OwnProfile{}, invalid("unknown interest")
		case "profiles_user_id_fkey", "user_interests_user_id_fkey", "preferences_user_id_fkey":
			return OwnProfile{}, ErrUnauthorized
		}
		return OwnProfile{}, err
	}
	return s.Own(ctx, userID)
}

func (s *Service) SetLocation(ctx context.Context, userID string, lat, lng float64) error {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 || lat != lat || lng != lng {
		return invalid("latitude/longitude out of range")
	}
	rl, rg := RoundCoordinate(lat), RoundCoordinate(lng)
	return mapNotFound(s.repo.SetLocation(ctx, userID, &rl, &rg))
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return mapNotFound(s.repo.SetLocation(ctx, userID, nil, nil))
}

func (s *Service) Preferences(ctx context.Context, userID string) (Preferences, error) {
	p, err := s.repo.GetPreferences(ctx, userID)
	return p, mapNotFound(err)
}

func (s *Service) SavePreferences(ctx context.Context, userID string, p Preferences) (Preferences, error) {
	clean, err := ValidatePreferences(p)
	if err != nil {
		return p, err
	}
	if err := s.repo.SetPreferences(ctx, userID, clean); err != nil {
		return p, mapNotFound(err)
	}
	return clean, nil
}

func (s *Service) Privacy(ctx context.Context, userID string) (map[string]bool, error) {
	p, err := s.repo.GetStored(ctx, userID)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return map[string]bool{"discoverable": p.Discoverable, "showDistance": p.ShowDistance}, nil
}

func (s *Service) SavePrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error {
	return mapNotFound(s.repo.SetPrivacy(ctx, userID, discoverable, showDistance))
}

// PublicProfiles hydrates cards for ids (order preserved) with approximate distance, interests and signed photo URLs.
// Callers are responsible for having filtered ids by visibility.
func (s *Service) PublicProfiles(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error) {
	rows, err := s.repo.PublicRows(ctx, viewerID, ids)
	if err != nil || len(rows) == 0 {
		return []PublicProfile{}, err
	}
	found := make([]string, len(rows))
	for i, row := range rows {
		found[i] = row.UserID
	}
	interests, photos, err := s.repo.InterestMaps(ctx, found)
	if err != nil {
		return nil, err
	}
	out := make([]PublicProfile, 0, len(rows))
	for _, row := range rows {
		p := row.PublicProfile
		p.Interests = interests[p.UserID]
		if p.Interests == nil {
			p.Interests = []Interest{}
		}
		p.Photos = s.photos(photos[p.UserID])
		if row.distance != nil {
			km := ApproxDistanceKm(*row.distance)
			p.DistanceKm = &km
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *Service) Profile(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if !ok {
		return PublicProfile{}, ErrNotFound
	}
	list, err := s.PublicProfiles(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	if len(list) == 0 {
		return PublicProfile{}, ErrNotFound
	}
	return list[0], nil
}

func mapNotFound(err error) error {
	if errors.Is(err, errRepoNotFound) {
		return ErrProfileNotFound
	}
	return err
}

// CanView reports whether viewer may see target's profile.
func (s *Service) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	return s.repo.CanView(ctx, viewerID, targetID)
}
