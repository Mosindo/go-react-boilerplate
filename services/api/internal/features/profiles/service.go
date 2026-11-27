package profiles

import (
	"context"
	"errors"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/httpx"
)

// PhotoURLSigner produces signed, expiring media URLs.
type PhotoURLSigner interface {
	PhotoURL(photoID string) string
}

type Service struct {
	repo   Repository
	signer PhotoURLSigner
	now    func() time.Time
}

func NewService(repo Repository, signer PhotoURLSigner) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

var multiNewlines = regexp.MustCompile(`\n{3,}`)

// AgeAt returns the age in full years at a given instant.
func AgeAt(birthdate, at time.Time) int {
	by, bm, bd := birthdate.Date()
	ay, am, ad := at.UTC().Date()
	age := ay - by
	if am < bm || (am == bm && ad < bd) {
		age--
	}
	return age
}

// SnapCoordinate rounds a coordinate to the privacy grid.
func SnapCoordinate(v float64) float64 {
	return math.Round(v/LocationGridDegrees) * LocationGridDegrees
}

// DistanceKm is the haversine distance between two points.
func DistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// DisplayDistance turns an exact distance into the coarse value shown to
// users: whole kilometres, never below 1 and rounded to 5 km beyond 20 km.
func DisplayDistance(km float64) int {
	switch {
	case km < 1.5:
		return 1
	case km <= 20:
		return int(math.Round(km))
	default:
		return int(math.Round(km/5) * 5)
	}
}

func cleanText(raw string) string {
	var b strings.Builder
	for _, r := range strings.ReplaceAll(raw, "\r\n", "\n") {
		if r == '\n' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(multiNewlines.ReplaceAllString(b.String(), "\n\n"))
}

func normalizeFirstName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	count := utf8.RuneCountInString(name)
	if count < 1 || count > MaxFirstNameRunes {
		return "", httpx.Invalid("invalid_first_name", "Le prénom doit contenir entre 1 et 40 caractères.")
	}
	for _, r := range name {
		if !(unicode.IsLetter(r) || r == ' ' || r == '-' || r == '\'' || r == '’' || unicode.Is(unicode.Mn, r)) {
			return "", httpx.Invalid("invalid_first_name", "Le prénom ne peut contenir que des lettres, espaces, tirets et apostrophes.")
		}
	}
	return name, nil
}

func (s *Service) parseBirthdate(raw string) (time.Time, error) {
	birthdate, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, httpx.Invalid("invalid_birthdate", "Date de naissance invalide (format AAAA-MM-JJ).")
	}
	age := AgeAt(birthdate, s.now())
	if age < MinAge {
		return time.Time{}, httpx.Invalid("underage", "Lueur est réservée aux personnes majeures (18 ans et plus).")
	}
	if age > 120 {
		return time.Time{}, httpx.Invalid("invalid_birthdate", "Date de naissance invalide.")
	}
	return birthdate, nil
}

func validGender(g string) bool {
	return slices.Contains(AllowedGenders, g)
}

// UpsertProfile validates and saves the core profile fields.
func (s *Service) UpsertProfile(ctx context.Context, userID string, req UpsertProfileRequest) error {
	firstName, err := normalizeFirstName(req.FirstName)
	if err != nil {
		return err
	}
	birthdate, err := s.parseBirthdate(req.Birthdate)
	if err != nil {
		return err
	}
	gender := strings.TrimSpace(req.Gender)
	if !validGender(gender) {
		return httpx.Invalid("invalid_gender", "Genre invalide.")
	}
	bio := cleanText(req.Bio)
	if utf8.RuneCountInString(bio) > MaxBioRunes {
		return httpx.Invalid("bio_too_long", "La bio ne peut pas dépasser 500 caractères.")
	}
	city := strings.Join(strings.Fields(cleanText(req.City)), " ")
	if utf8.RuneCountInString(city) > MaxCityRunes {
		return httpx.Invalid("city_too_long", "Le nom de ville ne peut pas dépasser 80 caractères.")
	}

	return s.repo.UpsertProfile(ctx, Profile{
		UserID:    userID,
		FirstName: firstName,
		Birthdate: birthdate,
		Gender:    gender,
		Bio:       bio,
		City:      city,
	})
}

// ValidatePreferences normalises and checks discovery preferences.
func ValidatePreferences(req PreferencesRequest) (Preferences, error) {
	seen := map[string]bool{}
	interested := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		g = strings.TrimSpace(g)
		if !validGender(g) {
			return Preferences{}, httpx.Invalid("invalid_interested_in", "Préférence de genre invalide.")
		}
		if !seen[g] {
			seen[g] = true
			interested = append(interested, g)
		}
	}
	if len(interested) == 0 {
		return Preferences{}, httpx.Invalid("invalid_interested_in", "Choisis au moins un genre.")
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, httpx.Invalid("invalid_age_range", "Tranche d'âge invalide (18 à 99 ans).")
	}
	if req.MaxDistanceKm < MinDistanceKm || req.MaxDistanceKm > MaxDistanceKm {
		return Preferences{}, httpx.Invalid("invalid_distance", "Distance invalide (1 à 500 km).")
	}
	return Preferences{InterestedIn: interested, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req PreferencesRequest) error {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return err
	}
	return s.repo.UpdatePreferences(ctx, userID, prefs)
}

func (s *Service) ReplaceInterests(ctx context.Context, userID string, ids []int) error {
	if _, err := s.repo.GetProfile(ctx, userID); err != nil {
		return err
	}
	unique := make([]int, 0, len(ids))
	seen := map[int]bool{}
	for _, id := range ids {
		if id <= 0 || id > math.MaxInt16 {
			return httpx.Invalid("invalid_interests", "Centre d'intérêt inconnu.")
		}
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	if len(unique) > MaxInterests {
		return httpx.Invalid("too_many_interests", "Tu peux choisir jusqu'à 10 centres d'intérêt.")
	}
	if len(unique) > 0 {
		count, err := s.repo.CountInterests(ctx, unique)
		if err != nil {
			return err
		}
		if count != len(unique) {
			return httpx.Invalid("invalid_interests", "Centre d'intérêt inconnu.")
		}
	}
	return s.repo.ReplaceUserInterests(ctx, userID, unique)
}

func (s *Service) SetLocation(ctx context.Context, userID string, lat, lng float64) error {
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) ||
		lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return httpx.Invalid("invalid_location", "Coordonnées invalides.")
	}
	snappedLat, snappedLng := SnapCoordinate(lat), SnapCoordinate(lng)
	return s.repo.SetLocation(ctx, userID, &snappedLat, &snappedLng)
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return s.repo.SetLocation(ctx, userID, nil, nil)
}

func (s *Service) SetPrivacy(ctx context.Context, userID string, visible, showDistance bool) error {
	return s.repo.SetPrivacy(ctx, userID, visible, showDistance)
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) TouchActivity(ctx context.Context, userID string) {
	_ = s.repo.TouchActivity(ctx, userID)
}

// MyProfile assembles the owner view, including completion status.
func (s *Service) MyProfile(ctx context.Context, userID string) (MyProfileResponse, error) {
	resp := MyProfileResponse{
		UserID:      userID,
		Missing:     []string{},
		Photos:      []PhotoResponse{},
		Interests:   []Interest{},
		Preferences: PreferencesResponse{InterestedIn: []string{}, MinAge: MinAge, MaxAge: MaxAge, MaxDistanceKm: 50},
		IsVisible:   true,
	}

	profile, err := s.repo.GetProfile(ctx, userID)
	switch {
	case errors.Is(err, ErrProfileNotFound):
		resp.Missing = append(resp.Missing, "profile", "photos")
		return resp, nil
	case err != nil:
		return MyProfileResponse{}, err
	}

	resp.Exists = true
	resp.FirstName = profile.FirstName
	resp.Birthdate = profile.Birthdate.Format("2006-01-02")
	resp.Age = AgeAt(profile.Birthdate, s.now())
	resp.Gender = profile.Gender
	resp.Bio = profile.Bio
	resp.City = profile.City
	resp.HasLocation = profile.HasLocation
	resp.IsVisible = profile.IsVisible
	resp.ShowDistance = profile.ShowDistance
	resp.IsDemo = profile.IsDemo

	ids := []string{userID}
	photos, err := s.repo.PhotosByUsers(ctx, ids)
	if err != nil {
		return MyProfileResponse{}, err
	}
	resp.Photos = s.PhotoResponses(photos[userID])
	interests, err := s.repo.InterestsByUsers(ctx, ids)
	if err != nil {
		return MyProfileResponse{}, err
	}
	if list := interests[userID]; list != nil {
		resp.Interests = list
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil && !errors.Is(err, ErrProfileNotFound) {
		return MyProfileResponse{}, err
	}
	if err == nil {
		resp.Preferences = PreferencesResponse(prefs)
	}

	if len(resp.Photos) == 0 {
		resp.Missing = append(resp.Missing, "photos")
	}
	resp.IsComplete = len(resp.Missing) == 0
	return resp, nil
}

// Cards loads public cards for the given users in one round of batched
// queries (no N+1), preserving the input order.
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) ([]Card, error) {
	if len(userIDs) == 0 {
		return []Card{}, nil
	}
	rows, err := s.repo.CardRows(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	lookup := append([]string{viewerID}, userIDs...)
	photos, err := s.repo.PhotosByUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.InterestsByUsers(ctx, lookup)
	if err != nil {
		return nil, err
	}
	viewerLat, viewerLng, err := s.repo.ViewerCoordinates(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	viewerInterests := map[int]bool{}
	for _, i := range interests[viewerID] {
		viewerInterests[i.ID] = true
	}

	byID := make(map[string]Card, len(rows))
	now := s.now()
	for _, row := range rows {
		card := Card{
			UserID:       row.UserID,
			FirstName:    row.FirstName,
			Age:          AgeAt(row.Birthdate, now),
			Gender:       row.Gender,
			Bio:          row.Bio,
			City:         row.City,
			IsDemo:       row.IsDemo,
			Photos:       photos[row.UserID],
			Interests:    interests[row.UserID],
			LastActiveAt: row.LastActiveAt,
		}
		if card.Photos == nil {
			card.Photos = []Photo{}
		}
		if card.Interests == nil {
			card.Interests = []Interest{}
		}
		for _, i := range card.Interests {
			if viewerInterests[i.ID] {
				card.SharedInterests++
			}
		}
		if row.ShowDistance && viewerLat != nil && viewerLng != nil && row.Latitude != nil && row.Longitude != nil {
			d := DisplayDistance(DistanceKm(*viewerLat, *viewerLng, *row.Latitude, *row.Longitude))
			card.DistanceKm = &d
		}
		byID[row.UserID] = card
	}

	out := make([]Card, 0, len(userIDs))
	for _, id := range userIDs {
		if card, ok := byID[id]; ok {
			out = append(out, card)
		}
	}
	return out, nil
}

// PublicProfile returns a card when the viewer is allowed to see it. Hidden
// and blocked profiles are reported as not found.
func (s *Service) PublicProfile(ctx context.Context, organizationID, viewerID, targetID string) (Card, error) {
	ok, err := s.repo.CanView(ctx, organizationID, viewerID, targetID)
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

func (s *Service) PhotoResponses(photos []Photo) []PhotoResponse {
	out := make([]PhotoResponse, 0, len(photos))
	for _, p := range photos {
		out = append(out, PhotoResponse{ID: p.ID, URL: s.signer.PhotoURL(p.ID), Position: p.Position, Width: p.Width, Height: p.Height})
	}
	return out
}

func (s *Service) CardResponse(card Card) CardResponse {
	return CardResponse{
		UserID:          card.UserID,
		FirstName:       card.FirstName,
		Age:             card.Age,
		Gender:          card.Gender,
		Bio:             card.Bio,
		City:            card.City,
		DistanceKm:      card.DistanceKm,
		IsDemo:          card.IsDemo,
		Photos:          s.PhotoResponses(card.Photos),
		Interests:       card.Interests,
		SharedInterests: card.SharedInterests,
	}
}
