package profiles

import (
	"context"
	"time"
)

// URLSigner turns a storage key into a short-lived URL.
type URLSigner interface {
	URL(key string) string
}

type Service struct {
	repo   Repository
	signer URLSigner
	now    func() time.Time
}

func NewService(repo Repository, signer URLSigner) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	interests, err := s.repo.InterestsByUsers(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	photos, err := s.repo.PhotosByUsers(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	_ = s.repo.Touch(ctx, userID)

	missing := missingSteps(prefs, len(photos[userID]))
	return OwnProfile{
		UserID:            p.UserID,
		FirstName:         p.FirstName,
		Birthdate:         p.Birthdate.Format("2006-01-02"),
		Age:               AgeOn(p.Birthdate, s.now()),
		Gender:            p.Gender,
		Bio:               p.Bio,
		Occupation:        p.Occupation,
		RelationshipGoal:  p.RelationshipGoal,
		City:              p.City,
		HasLocation:       p.Latitude != nil,
		LocationUpdatedAt: p.LocationUpdatedAt,
		Privacy:           p.Privacy,
		Preferences:       prefs,
		Interests:         nonNilInterests(interests[userID]),
		Photos:            s.signPhotos(photos[userID]),
		IsComplete:        len(missing) == 0,
		Missing:           missing,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}, nil
}

// IsComplete reports whether the user may use discovery (profile, preferences, a photo).
func (s *Service) IsComplete(ctx context.Context, userID string) (bool, error) {
	if _, err := s.repo.GetProfile(ctx, userID); err != nil {
		if err == ErrProfileNotFound {
			return false, nil
		}
		return false, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return false, err
	}
	photos, err := s.repo.PhotosByUsers(ctx, []string{userID})
	if err != nil {
		return false, err
	}
	return len(missingSteps(prefs, len(photos[userID]))) == 0, nil
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (OwnProfile, error) {
	in, err := ValidateProfile(req, s.now())
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.UpsertProfile(ctx, userID, in); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req PreferencesRequest) (OwnProfile, error) {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return OwnProfile{}, err
	}
	if _, err := s.repo.GetProfile(ctx, userID); err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.UpdatePreferences(ctx, userID, prefs); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) SetInterests(ctx context.Context, userID string, ids []int) (OwnProfile, error) {
	clean, err := ValidateInterestIDs(ids)
	if err != nil {
		return OwnProfile{}, err
	}
	if _, err := s.repo.GetProfile(ctx, userID); err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.ReplaceInterests(ctx, userID, clean); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lng float64, city *string) (OwnProfile, error) {
	rlat, rlng, err := ValidateCoordinates(lat, lng)
	if err != nil {
		return OwnProfile{}, err
	}
	if city != nil {
		normalized := ProfileInput{}
		normalized.City = normalizeText(*city)
		if len([]rune(normalized.City)) > maxCityRunes {
			return OwnProfile{}, ErrCityTooLong
		}
		city = &normalized.City
	}
	if err := s.repo.UpdateLocation(ctx, userID, &rlat, &rlng, city); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) ClearLocation(ctx context.Context, userID string) (OwnProfile, error) {
	if err := s.repo.UpdateLocation(ctx, userID, nil, nil, nil); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdatePrivacy(ctx context.Context, userID string, p Privacy) (OwnProfile, error) {
	if err := s.repo.UpdatePrivacy(ctx, userID, p); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// GetPublic returns another member's public profile after authorization.
func (s *Service) GetPublic(ctx context.Context, viewerID, organizationID, targetID string) (PublicProfile, error) {
	if viewerID != targetID {
		ok, err := s.repo.CanView(ctx, viewerID, organizationID, targetID)
		if err != nil {
			return PublicProfile{}, err
		}
		if !ok {
			return PublicProfile{}, ErrProfileNotAccessible
		}
	}
	list, err := s.PublicProfiles(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	if len(list) == 0 {
		return PublicProfile{}, ErrProfileNotAccessible
	}
	return list[0], nil
}

// PublicProfiles builds public cards for ids in order, with 3 batched queries
// (no N+1). Callers are responsible for authorization.
func (s *Service) PublicProfiles(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error) {
	if len(ids) == 0 {
		return []PublicProfile{}, nil
	}
	rows, err := s.repo.PublicRows(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.InterestsByUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.PhotosByUsers(ctx, ids)
	if err != nil {
		return nil, err
	}

	now := s.now()
	out := make([]PublicProfile, 0, len(ids))
	for _, id := range ids {
		row, ok := rows[id]
		if !ok {
			continue
		}
		card := PublicProfile{
			UserID:           row.UserID,
			FirstName:        row.FirstName,
			Gender:           row.Gender,
			Bio:              row.Bio,
			Occupation:       row.Occupation,
			RelationshipGoal: row.RelationshipGoal,
			City:             row.City,
			Interests:        nonNilInterests(interests[id]),
			Photos:           s.signPhotos(photos[id]),
		}
		if row.ShowAge || id == viewerID {
			age := AgeOn(row.Birthdate, now)
			card.Age = &age
		}
		if row.DistanceKm != nil && row.ShowDistance && id != viewerID {
			d := ApproxDistanceKm(*row.DistanceKm)
			card.DistanceKm = &d
		}
		out = append(out, card)
	}
	return out, nil
}

func (s *Service) signPhotos(stored []storedPhoto) []Photo {
	out := make([]Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, Photo{ID: p.ID, URL: s.signer.URL(p.StorageKey), Width: p.Width, Height: p.Height, Position: p.Position})
	}
	return out
}

func nonNilInterests(in []Interest) []Interest {
	if in == nil {
		return []Interest{}
	}
	return in
}
