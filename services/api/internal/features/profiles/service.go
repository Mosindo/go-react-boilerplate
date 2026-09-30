package profiles

import (
	"context"
	"errors"
	"time"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/media"
)

var (
	ErrProfileNotFound   = apperr.NotFound("profile not found")
	ErrBirthdateLocked   = apperr.Conflict("birthdate_locked", "birthdate cannot be changed once set")
	ErrTooManyInterests  = apperr.Validation("select at most 10 interests")
	ErrUnknownInterest   = apperr.Validation("unknown interest")
	ErrInvalidCoordinate = apperr.Validation("invalid coordinates")
)

type Service struct {
	repo   Repository
	signer *media.Signer
	now    func() time.Time
}

func NewService(repo Repository, signer *media.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	p, err := s.repo.GetOwn(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return OwnProfile{}, ErrProfileNotFound
		}
		return OwnProfile{}, err
	}
	photoIDs, err := s.repo.PhotoIDsByUser(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	interests, err := s.repo.InterestsByUser(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}

	own := OwnProfile{
		UserID:           p.UserID,
		FirstName:        p.FirstName,
		Gender:           p.Gender,
		Bio:              p.Bio,
		JobTitle:         p.JobTitle,
		RelationshipGoal: p.RelationshipGoal,
		City:             p.City,
		HasLocation:      p.HasLocation,
		Discoverable:     p.Discoverable,
		ShowDistance:     p.ShowDistance,
		Photos:           s.photos(photoIDs[userID]),
		Interests:        nonNilInterests(interests[userID]),
		Preferences:      p.Preferences,
		Completeness:     ComputeCompleteness(p, len(photoIDs[userID])),
	}
	if own.Preferences.InterestedIn == nil {
		own.Preferences.InterestedIn = []string{}
	}
	if p.Birthdate != nil {
		formatted := p.Birthdate.Format(birthdateLayout)
		age := AgeOn(*p.Birthdate, s.now())
		own.Birthdate = &formatted
		own.Age = &age
	}
	return own, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (OwnProfile, error) {
	p, err := s.repo.GetOwn(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return OwnProfile{}, ErrProfileNotFound
		}
		return OwnProfile{}, err
	}

	if req.FirstName != nil {
		if p.FirstName, err = ValidateFirstName(*req.FirstName); err != nil {
			return OwnProfile{}, err
		}
	}
	if req.Birthdate != nil {
		birthdate, err := ParseBirthdate(*req.Birthdate, s.now())
		if err != nil {
			return OwnProfile{}, err
		}
		// Birthdate is locked once set to prevent age manipulation.
		if p.Birthdate != nil && !p.Birthdate.Equal(birthdate) {
			return OwnProfile{}, ErrBirthdateLocked
		}
		p.Birthdate = &birthdate
	}
	if req.Gender != nil {
		gender, err := ValidateGender(*req.Gender)
		if err != nil {
			return OwnProfile{}, err
		}
		p.Gender = &gender
	}
	if req.Bio != nil {
		if p.Bio, err = CleanText(*req.Bio, 500, "bio", true); err != nil {
			return OwnProfile{}, err
		}
	}
	if req.JobTitle != nil {
		if p.JobTitle, err = CleanText(*req.JobTitle, 60, "jobTitle", false); err != nil {
			return OwnProfile{}, err
		}
	}
	if req.RelationshipGoal != nil {
		if p.RelationshipGoal, err = ValidateRelationshipGoal(*req.RelationshipGoal); err != nil {
			return OwnProfile{}, err
		}
	}
	if req.City != nil {
		if p.City, err = CleanText(*req.City, 80, "city", false); err != nil {
			return OwnProfile{}, err
		}
	}
	if req.Discoverable != nil {
		p.Discoverable = *req.Discoverable
	}
	if req.ShowDistance != nil {
		p.ShowDistance = *req.ShowDistance
	}

	if err := s.repo.SaveProfile(ctx, p); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (OwnProfile, error) {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.SavePreferences(ctx, userID, prefs); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) UpdateInterests(ctx context.Context, userID string, ids []int) (OwnProfile, error) {
	unique := make([]int, 0, len(ids))
	seen := map[int]bool{}
	for _, id := range ids {
		if id <= 0 || id > 32767 {
			return OwnProfile{}, ErrUnknownInterest
		}
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	if len(unique) > MaxInterests {
		return OwnProfile{}, ErrTooManyInterests
	}
	if len(unique) > 0 {
		count, err := s.repo.CountExistingInterests(ctx, unique)
		if err != nil {
			return OwnProfile{}, err
		}
		if count != len(unique) {
			return OwnProfile{}, ErrUnknownInterest
		}
	}
	if err := s.repo.SetInterests(ctx, userID, unique); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

// UpdateLocation stores a coarsened position (~1 km grid). The exact
// coordinates sent by the device are discarded immediately.
func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lon float64, city *string) (OwnProfile, error) {
	if !geo.ValidLatitude(lat) || !geo.ValidLongitude(lon) {
		return OwnProfile{}, ErrInvalidCoordinate
	}
	var cleanCity *string
	if city != nil {
		value, err := CleanText(*city, 80, "city", false)
		if err != nil {
			return OwnProfile{}, err
		}
		cleanCity = &value
	}
	if err := s.repo.SetLocation(ctx, userID, geo.CoarsenCoordinate(lat), geo.CoarsenCoordinate(lon), cleanCity); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) ClearLocation(ctx context.Context, userID string) (OwnProfile, error) {
	if err := s.repo.ClearLocation(ctx, userID); err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// GetPublic returns another member's profile if the viewer may see it.
func (s *Service) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	if viewerID == targetID {
		return PublicProfile{}, ErrProfileNotFound
	}
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if !ok {
		return PublicProfile{}, ErrProfileNotFound
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	if len(cards) == 0 {
		return PublicProfile{}, ErrProfileNotFound
	}
	return cards[0], nil
}

// Cards builds public profiles for the given users in the given order using
// a constant number of queries (no N+1). Callers are responsible for having
// checked the viewer is allowed to see these users.
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) ([]PublicProfile, error) {
	if len(userIDs) == 0 {
		return []PublicProfile{}, nil
	}
	rows, err := s.repo.CardRows(ctx, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	photoIDs, err := s.repo.PhotoIDsByUser(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	interestLookup := append([]string{viewerID}, userIDs...)
	interests, err := s.repo.InterestsByUser(ctx, interestLookup)
	if err != nil {
		return nil, err
	}
	viewerInterests := map[int]bool{}
	for _, i := range interests[viewerID] {
		viewerInterests[i.ID] = true
	}

	byID := make(map[string]cardRow, len(rows))
	for _, row := range rows {
		byID[row.UserID] = row
	}

	now := s.now()
	cards := make([]PublicProfile, 0, len(userIDs))
	for _, id := range userIDs {
		row, ok := byID[id]
		if !ok || row.Birthdate == nil || row.Gender == nil {
			continue
		}
		card := PublicProfile{
			UserID:           row.UserID,
			FirstName:        row.FirstName,
			Age:              AgeOn(*row.Birthdate, now),
			Gender:           *row.Gender,
			Bio:              row.Bio,
			JobTitle:         row.JobTitle,
			RelationshipGoal: row.RelationshipGoal,
			City:             row.City,
			Photos:           s.photos(photoIDs[id]),
			Interests:        nonNilInterests(interests[id]),
		}
		if row.ShowDistance && row.DistanceKm != nil {
			approx := geo.ApproximateDistanceKm(*row.DistanceKm)
			card.DistanceKm = &approx
		}
		for _, i := range card.Interests {
			if viewerInterests[i.ID] {
				card.SharedInterests++
			}
		}
		cards = append(cards, card)
	}
	return cards, nil
}

// Summaries returns lightweight identities (first name, age, main photo).
func (s *Service) Summaries(ctx context.Context, viewerID string, userIDs []string) (map[string]Summary, error) {
	result := make(map[string]Summary, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := s.repo.CardRows(ctx, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	photoIDs, err := s.repo.PhotoIDsByUser(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	now := s.now()
	for _, row := range rows {
		summary := Summary{UserID: row.UserID, FirstName: row.FirstName}
		if row.Birthdate != nil {
			age := AgeOn(*row.Birthdate, now)
			summary.Age = &age
		}
		if ids := photoIDs[row.UserID]; len(ids) > 0 {
			photo := Photo{ID: ids[0], URL: s.signer.PhotoURL(ids[0])}
			summary.Photo = &photo
		}
		result[row.UserID] = summary
	}
	return result, nil
}

func (s *Service) photos(ids []string) []Photo {
	photos := make([]Photo, 0, len(ids))
	for _, id := range ids {
		photos = append(photos, Photo{ID: id, URL: s.signer.PhotoURL(id)})
	}
	return photos
}

func nonNilInterests(items []Interest) []Interest {
	if items == nil {
		return []Interest{}
	}
	return items
}
