package profiles

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/api/internal/features/photos"
	apperr "example.com/api/internal/platform/errors"
)

var (
	ErrProfileNotFound = apperr.NotFound("profile_not_found", "profile not found")
	ErrProfileRequired = apperr.Conflict("profile_required", "create your profile first")
)

type PhotoSource interface {
	ListMine(ctx context.Context, userID string) ([]photos.PhotoResponse, error)
	ListForUsers(ctx context.Context, userIDs []string) (map[string][]photos.PhotoResponse, error)
}

type Service struct {
	repo   Repository
	photos PhotoSource
	now    func() time.Time
}

func NewService(repo Repository, photoSource PhotoSource) *Service {
	return &Service{repo: repo, photos: photoSource, now: time.Now}
}

func (s *Service) toProfileResponse(p Profile) *ProfileResponse {
	return &ProfileResponse{
		FirstName:        p.FirstName,
		Birthdate:        p.Birthdate.Format("2006-01-02"),
		Age:              AgeAt(p.Birthdate, s.now()),
		Gender:           p.Gender,
		Bio:              p.Bio,
		JobTitle:         p.JobTitle,
		HeightCm:         p.HeightCm,
		RelationshipGoal: p.RelationshipGoal,
		City:             p.City,
		HasLocation:      p.Latitude != nil,
		IsDiscoverable:   p.IsDiscoverable,
		ShowDistance:     p.ShowDistance,
	}
}

func (s *Service) GetMine(ctx context.Context, userID string) (MyProfileResponse, error) {
	resp := MyProfileResponse{Missing: []string{}}
	profile, err := s.repo.GetProfile(ctx, userID)
	switch {
	case err == nil:
		resp.Profile = s.toProfileResponse(profile)
	case errors.Is(err, ErrRepositoryNotFound):
		resp.Missing = append(resp.Missing, "profile")
	default:
		return MyProfileResponse{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return MyProfileResponse{}, err
	}
	resp.Preferences = PreferencesResponse(prefs)
	if resp.Interests, err = s.repo.UserInterests(ctx, userID); err != nil {
		return MyProfileResponse{}, err
	}
	if resp.Photos, err = s.photos.ListMine(ctx, userID); err != nil {
		return MyProfileResponse{}, err
	}
	if len(resp.Photos) == 0 {
		resp.Missing = append(resp.Missing, "photo")
	}
	if resp.Profile == nil || !resp.Profile.HasLocation {
		resp.Missing = append(resp.Missing, "location")
	}
	resp.IsComplete = len(resp.Missing) == 0
	return resp, nil
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (MyProfileResponse, error) {
	p, err := NormalizeProfile(req, s.now())
	if err != nil {
		return MyProfileResponse{}, err
	}
	p.UserID = userID
	if err := s.repo.UpsertProfile(ctx, p); err != nil {
		return MyProfileResponse{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, req UpdateLocationRequest) (MyProfileResponse, error) {
	lat, lng, err := NormalizeLocation(*req.Latitude, *req.Longitude)
	if err != nil {
		return MyProfileResponse{}, err
	}
	var city *string
	if req.City != nil {
		cleaned := strings.Join(strings.Fields(*req.City), " ")
		if utf8.RuneCountInString(cleaned) > 80 {
			return MyProfileResponse{}, ErrCityTooLong
		}
		city = &cleaned
	}
	if err := s.repo.UpdateLocation(ctx, userID, lat, lng, city); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return MyProfileResponse{}, ErrProfileRequired
		}
		return MyProfileResponse{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) UpdatePrivacy(ctx context.Context, userID string, req UpdatePrivacyRequest) (MyProfileResponse, error) {
	if err := s.repo.UpdatePrivacy(ctx, userID, *req.IsDiscoverable, *req.ShowDistance); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return MyProfileResponse{}, ErrProfileRequired
		}
		return MyProfileResponse{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (PreferencesResponse, error) {
	prefs, err := NormalizePreferences(req)
	if err != nil {
		return PreferencesResponse{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, prefs); err != nil {
		return PreferencesResponse{}, err
	}
	return PreferencesResponse(prefs), nil
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) ReplaceInterests(ctx context.Context, userID string, ids []int) ([]Interest, error) {
	normalized, err := NormalizeInterestIDs(ids)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceInterests(ctx, userID, normalized); err != nil {
		return nil, err
	}
	return s.repo.UserInterests(ctx, userID)
}

// GetCard returns another member's public profile if the viewer may see it.
func (s *Service) GetCard(ctx context.Context, viewerID, targetID string) (Card, error) {
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return Card{}, err
	}
	if !ok {
		return Card{}, ErrProfileNotFound
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return Card{}, err
	}
	if len(cards) == 0 {
		return Card{}, ErrProfileNotFound
	}
	return cards[0], nil
}

// Cards builds public cards for targetIDs (order preserved) using three
// batched queries, whatever the page size.
func (s *Service) Cards(ctx context.Context, viewerID string, targetIDs []string) ([]Card, error) {
	if len(targetIDs) == 0 {
		return []Card{}, nil
	}
	rows, err := s.repo.CardRows(ctx, viewerID, targetIDs)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.CardInterests(ctx, viewerID, targetIDs)
	if err != nil {
		return nil, err
	}
	photoMap, err := s.photos.ListForUsers(ctx, targetIDs)
	if err != nil {
		return nil, err
	}
	order := make(map[string]int, len(targetIDs))
	for i, id := range targetIDs {
		order[id] = i
	}
	sort.SliceStable(rows, func(i, j int) bool { return order[rows[i].UserID] < order[rows[j].UserID] })

	now := s.now()
	cards := make([]Card, 0, len(rows))
	for _, row := range rows {
		card := Card{
			UserID:           row.UserID,
			FirstName:        row.FirstName,
			Age:              AgeAt(row.Birthdate, now),
			Gender:           row.Gender,
			Bio:              row.Bio,
			JobTitle:         row.JobTitle,
			HeightCm:         row.HeightCm,
			RelationshipGoal: row.RelationshipGoal,
			City:             row.City,
			Interests:        interests[row.UserID],
			Photos:           photoMap[row.UserID],
		}
		if card.Interests == nil {
			card.Interests = []CardInterest{}
		}
		if card.Photos == nil {
			card.Photos = []photos.PhotoResponse{}
		}
		if row.ShowDistance && row.DistanceKm != nil && row.UserID != viewerID {
			d := ApproximateDistance(*row.DistanceKm)
			card.DistanceKm = &d
		}
		cards = append(cards, card)
	}
	return cards, nil
}
