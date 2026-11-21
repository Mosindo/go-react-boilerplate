package profiles

import (
	"context"
	"time"

	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/media"
)

type Service struct {
	repo   Repository
	signer *media.Signer
	now    func() time.Time
}

func NewService(repo Repository, signer *media.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) Me(ctx context.Context, userID string) (MeProfileResponse, error) {
	email, err := s.repo.GetEmail(ctx, userID)
	if err != nil {
		return MeProfileResponse{}, err
	}
	profile, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return MeProfileResponse{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return MeProfileResponse{}, err
	}
	photos, err := s.repo.ListOwnPhotos(ctx, userID)
	if err != nil {
		return MeProfileResponse{}, err
	}
	interests, err := s.repo.ListUserInterests(ctx, userID)
	if err != nil {
		return MeProfileResponse{}, err
	}

	missing := Missing(profile, len(photos))
	resp := MeProfileResponse{
		UserID:     userID,
		Email:      email,
		Photos:     s.PhotoResponses(photos),
		Interests:  interestResponses(interests),
		IsComplete: len(missing) == 0,
		Missing:    missing,
	}
	if profile != nil {
		resp.Profile = &OwnProfileResponse{
			FirstName:        profile.FirstName,
			Birthdate:        profile.Birthdate.Format("2006-01-02"),
			Age:              Age(profile.Birthdate, s.now()),
			Gender:           profile.Gender,
			Bio:              profile.Bio,
			City:             profile.City,
			JobTitle:         profile.JobTitle,
			HeightCm:         profile.HeightCm,
			RelationshipGoal: profile.RelationshipGoal,
			HasLocation:      profile.HasLocation,
			IsVisible:        profile.IsVisible,
			ShowDistance:     profile.ShowDistance,
		}
	}
	if prefs != nil {
		resp.Preferences = &PreferencesResponse{
			InterestedIn:  prefs.InterestedIn,
			MinAge:        prefs.MinAge,
			MaxAge:        prefs.MaxAge,
			MaxDistanceKm: prefs.MaxDistanceKm,
		}
	}
	return resp, nil
}

func (s *Service) UpsertProfile(ctx context.Context, userID string, req UpsertProfileRequest) error {
	input, err := ValidateProfileInput(req, s.now())
	if err != nil {
		return err
	}
	return s.repo.UpsertProfile(ctx, userID, input)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) error {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return err
	}
	return s.repo.UpdatePreferences(ctx, userID, prefs)
}

func (s *Service) UpdateInterests(ctx context.Context, userID string, ids []int) error {
	normalized, err := NormalizeInterestIDs(ids)
	if err != nil {
		return err
	}
	profile, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return err
	}
	if profile == nil {
		return ErrProfileNotFound
	}
	return s.repo.ReplaceInterests(ctx, userID, normalized)
}

// UpdateLocation stores a coarsened location; exact coordinates are never persisted.
func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lng float64) error {
	if !geo.Valid(lat, lng) {
		return ErrInvalidLocation
	}
	cLat, cLng := geo.Coarsen(lat, lng)
	return s.repo.SetLocation(ctx, userID, &cLat, &cLng)
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return s.repo.SetLocation(ctx, userID, nil, nil)
}

func (s *Service) UpdateSettings(ctx context.Context, userID string, req UpdateSettingsRequest) error {
	return s.repo.UpdateSettings(ctx, userID, req.IsVisible, req.ShowDistance)
}

func (s *Service) ListInterests(ctx context.Context) ([]InterestResponse, error) {
	interests, err := s.repo.ListInterests(ctx)
	if err != nil {
		return nil, err
	}
	return interestResponses(interests), nil
}

// View returns a public profile if the viewer is allowed to see it.
func (s *Service) View(ctx context.Context, viewerID, targetID string) (CardResponse, error) {
	allowed, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return CardResponse{}, err
	}
	if !allowed {
		return CardResponse{}, ErrProfileNotFound
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return CardResponse{}, err
	}
	if len(cards) == 0 {
		return CardResponse{}, ErrProfileNotFound
	}
	return cards[0], nil
}

// Cards returns public cards in the order of userIDs. Callers are responsible
// for authorization (discovery, matches, chat already filter blocked users).
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) ([]CardResponse, error) {
	records, err := s.repo.LoadCards(ctx, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	out := make([]CardResponse, 0, len(records))
	for _, rec := range records {
		out = append(out, s.cardResponse(rec))
	}
	return out, nil
}

// Summaries returns compact cards keyed by user id.
func (s *Service) Summaries(ctx context.Context, viewerID string, userIDs []string) (map[string]SummaryResponse, error) {
	cards, err := s.Cards(ctx, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]SummaryResponse, len(cards))
	for _, c := range cards {
		out[c.UserID] = ToSummary(c)
	}
	return out, nil
}

// TouchActive records activity for ranking ("recently active" signal).
func (s *Service) TouchActive(ctx context.Context, userID string) {
	_ = s.repo.TouchActive(ctx, userID)
}

func ToSummary(c CardResponse) SummaryResponse {
	summary := SummaryResponse{ID: c.UserID, FirstName: c.FirstName, Age: c.Age, IsDemo: c.IsDemo}
	if len(c.Photos) > 0 {
		url := c.Photos[0].ThumbURL
		summary.PhotoURL = &url
	}
	return summary
}

// DeletedSummary is shown when the other member no longer exists.
func DeletedSummary(userID string) SummaryResponse {
	return SummaryResponse{ID: userID, FirstName: "Membre"}
}

func (s *Service) cardResponse(rec CardRecord) CardResponse {
	var distance *int
	if rec.DistanceKm != nil {
		d := geo.DisplayDistanceKm(*rec.DistanceKm)
		distance = &d
	}
	return CardResponse{
		UserID:           rec.UserID,
		FirstName:        rec.FirstName,
		Age:              rec.Age,
		Gender:           rec.Gender,
		Bio:              rec.Bio,
		City:             rec.City,
		JobTitle:         rec.JobTitle,
		HeightCm:         rec.HeightCm,
		RelationshipGoal: rec.RelationshipGoal,
		DistanceKm:       distance,
		IsDemo:           rec.IsDemo,
		Photos:           s.PhotoResponses(rec.Photos),
		Interests:        interestResponses(rec.Interests),
	}
}

func (s *Service) PhotoResponses(photos []Photo) []PhotoResponse {
	out := make([]PhotoResponse, 0, len(photos))
	for _, p := range photos {
		out = append(out, PhotoResponse{
			ID:       p.ID,
			Position: p.Position,
			URL:      s.signer.URL(p.ID, media.VariantFull),
			ThumbURL: s.signer.URL(p.ID, media.VariantThumb),
			Width:    p.Width,
			Height:   p.Height,
		})
	}
	return out
}

func interestResponses(interests []Interest) []InterestResponse {
	out := make([]InterestResponse, 0, len(interests))
	for _, i := range interests {
		out = append(out, InterestResponse{ID: i.ID, Slug: i.Slug, Label: i.Label})
	}
	return out
}
