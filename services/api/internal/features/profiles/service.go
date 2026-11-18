package profiles

import (
	"context"
	"errors"
	"strings"
	"time"

	"example.com/api/internal/platform/people"
)

// URLSigner turns a photo id into a short-lived URL.
type URLSigner interface {
	PhotoURL(photoID string) string
}

type Service struct {
	repo   Repository
	signer URLSigner
	now    func() time.Time
}

func NewService(repo Repository, signer URLSigner) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) Me(ctx context.Context, userID string) (MyProfileResponse, error) {
	resp := MyProfileResponse{Photos: []Photo{}, Interests: []Interest{}, Missing: []string{}}

	profile, err := s.repo.GetProfile(ctx, userID)
	switch {
	case err == nil:
		view := s.toView(profile)
		resp.Profile = &view
	case errors.Is(err, ErrRepositoryNotFound):
		resp.Missing = append(resp.Missing, "profile")
	default:
		return MyProfileResponse{}, err
	}

	prefs, err := s.repo.GetPreferences(ctx, userID)
	switch {
	case err == nil:
		resp.Preferences = &prefs
	case errors.Is(err, ErrRepositoryNotFound):
		resp.Missing = append(resp.Missing, "preferences")
	default:
		return MyProfileResponse{}, err
	}

	photos, err := s.repo.ListPhotos(ctx, userID)
	if err != nil {
		return MyProfileResponse{}, err
	}
	for _, p := range photos {
		resp.Photos = append(resp.Photos, Photo{ID: p.ID, Position: p.Position, URL: s.signer.PhotoURL(p.ID), Width: p.Width, Height: p.Height})
	}
	if len(photos) == 0 {
		resp.Missing = append(resp.Missing, "photos")
	}

	interests, err := s.repo.ListUserInterests(ctx, userID)
	if err != nil {
		return MyProfileResponse{}, err
	}
	resp.Interests = interests
	resp.IsComplete = len(resp.Missing) == 0
	return resp, nil
}

func (s *Service) toView(p Profile) ProfileView {
	return ProfileView{
		FirstName:         p.FirstName,
		Birthdate:         p.Birthdate.Format("2006-01-02"),
		Age:               people.Age(p.Birthdate, s.now()),
		Gender:            p.Gender,
		Bio:               p.Bio,
		JobTitle:          p.JobTitle,
		City:              p.City,
		HeightCm:          p.HeightCm,
		RelationshipGoal:  p.RelationshipGoal,
		HasLocation:       p.Latitude != nil,
		LocationUpdatedAt: p.LocationUpdatedAt,
		IsDiscoverable:    p.IsDiscoverable,
		ShowDistance:      p.ShowDistance,
		UnderReview:       p.ModerationHidden,
	}
}

func (s *Service) UpsertProfile(ctx context.Context, userID string, req UpsertProfileRequest) error {
	firstName, err := validateFirstName(req.FirstName)
	if err != nil {
		return err
	}
	birthdate, err := validateBirthdate(req.Birthdate, s.now())
	if err != nil {
		return err
	}
	gender := strings.TrimSpace(req.Gender)
	if !contains(Genders, gender) {
		return ErrInvalidGender
	}
	goal := strings.TrimSpace(req.RelationshipGoal)
	if goal == "" {
		goal = "unsure"
	}
	if !contains(RelationshipGoals, goal) {
		return ErrInvalidGoal
	}
	if req.HeightCm != nil && (*req.HeightCm < 120 || *req.HeightCm > 230) {
		return ErrInvalidHeight
	}
	bio, err := cleanText(req.Bio, maxBioRunes, true)
	if err != nil {
		return err
	}
	jobTitle, err := cleanText(req.JobTitle, maxJobTitleRunes, false)
	if err != nil {
		return err
	}
	city, err := cleanText(req.City, maxCityRunes, false)
	if err != nil {
		return err
	}

	existing, err := s.repo.GetProfile(ctx, userID)
	if err == nil && !existing.Birthdate.Equal(birthdate) {
		return ErrBirthdateLocked
	}
	if err != nil && !errors.Is(err, ErrRepositoryNotFound) {
		return err
	}

	return s.repo.UpsertProfile(ctx, Profile{
		UserID:           userID,
		FirstName:        firstName,
		Birthdate:        birthdate,
		Gender:           gender,
		Bio:              bio,
		JobTitle:         jobTitle,
		City:             city,
		HeightCm:         req.HeightCm,
		RelationshipGoal: goal,
	})
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req PreferencesRequest) (Preferences, error) {
	prefs, err := validatePreferences(req)
	if err != nil {
		return Preferences{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, prefs); err != nil {
		return Preferences{}, err
	}
	return prefs, nil
}

func (s *Service) UpdatePrivacy(ctx context.Context, userID string, req PrivacyRequest) error {
	err := s.repo.UpdatePrivacy(ctx, userID, req.IsDiscoverable, req.ShowDistance)
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrProfileRequired
	}
	return err
}

// UpdateLocation rounds coordinates before they ever reach the database.
func (s *Service) UpdateLocation(ctx context.Context, userID string, req LocationRequest) error {
	if err := validateLocation(*req.Latitude, *req.Longitude); err != nil {
		return err
	}
	lat := people.RoundCoordinate(*req.Latitude)
	lng := people.RoundCoordinate(*req.Longitude)
	var city *string
	if strings.TrimSpace(req.City) != "" {
		cleaned, err := cleanText(req.City, maxCityRunes, false)
		if err != nil {
			return err
		}
		city = &cleaned
	}
	err := s.repo.UpdateLocation(ctx, userID, &lat, &lng, city)
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrProfileRequired
	}
	return err
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	err := s.repo.UpdateLocation(ctx, userID, nil, nil, nil)
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrProfileRequired
	}
	return err
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) ReplaceInterests(ctx context.Context, userID string, slugs []string) ([]Interest, error) {
	unique := make([]string, 0, len(slugs))
	seen := map[string]struct{}{}
	for _, slug := range slugs {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		unique = append(unique, slug)
	}
	if len(unique) > MaxInterests {
		return nil, ErrTooManyInterests
	}
	if err := s.repo.ReplaceUserInterests(ctx, userID, unique); err != nil {
		return nil, err
	}
	return s.repo.ListUserInterests(ctx, userID)
}
