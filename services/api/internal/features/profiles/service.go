package profiles

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrValidation     = errors.New("profiles: validation failed")
	ErrProfileMissing = errors.New("profiles: profile not found")
	ErrForbidden      = errors.New("profiles: not visible")
)

// ValidationError carries a user-facing message and unwraps to ErrValidation.
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }
func (e ValidationError) Unwrap() error { return ErrValidation }

func invalid(msg string) error { return ValidationError{Message: msg} }

type FileStore interface {
	Put(data []byte) (string, error)
	Delete(key string)
}

type Service struct {
	repo   Repository
	files  FileStore
	signer *MediaSigner
	now    func() time.Time
}

func NewService(repo Repository, files FileStore, signer *MediaSigner) *Service {
	return &Service{repo: repo, files: files, signer: signer, now: time.Now}
}

func (s *Service) Signer() *MediaSigner { return s.signer }

func (s *Service) Catalog(ctx context.Context) ([]Interest, error) { return s.repo.Catalog(ctx) }

func (s *Service) photoViews(list []storedPhoto) []PhotoView {
	out := make([]PhotoView, 0, len(list))
	for _, p := range list {
		out = append(out, PhotoView{ID: p.ID, URL: s.signer.URL(p.ID)})
	}
	return out
}

func (s *Service) GetMine(ctx context.Context, userID string) (MyProfile, error) {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return MyProfile{}, ErrProfileMissing
		}
		return MyProfile{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return MyProfile{}, err
	}
	interests, err := s.repo.InterestsFor(ctx, []string{userID})
	if err != nil {
		return MyProfile{}, err
	}
	photos, err := s.repo.PhotosFor(ctx, []string{userID})
	if err != nil {
		return MyProfile{}, err
	}
	out := MyProfile{
		UserID: p.UserID, FirstName: p.FirstName, BirthDate: p.BirthDate.Format("2006-01-02"),
		Age: AgeOn(p.BirthDate, s.now()), Gender: p.Gender, Bio: p.Bio, City: p.City,
		HasLocation: p.HasLocation, ShowDistance: p.ShowDistance, Discoverable: p.Discoverable,
		Interests: interests[userID], Preferences: prefs, Photos: s.photoViews(photos[userID]),
	}
	if out.Interests == nil {
		out.Interests = []Interest{}
	}
	out.Complete = len(out.Photos) > 0
	return out, nil
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (MyProfile, error) {
	in, err := s.validate(req)
	if err != nil {
		return MyProfile{}, err
	}
	// Omitted privacy flags keep their stored value (never silently re-list or re-expose a user).
	if req.ShowDistance == nil || req.Discoverable == nil {
		if cur, err := s.repo.GetProfile(ctx, userID); err == nil {
			if req.ShowDistance == nil {
				in.ShowDistance = cur.ShowDistance
			}
			if req.Discoverable == nil {
				in.Discoverable = cur.Discoverable
			}
		}
	}
	if err := s.repo.UpsertProfile(ctx, userID, in); err != nil {
		if errors.Is(err, ErrUnknownInterest) {
			return MyProfile{}, invalid("unknown interest")
		}
		return MyProfile{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) validate(req UpsertProfileRequest) (ProfileInput, error) {
	name := strings.TrimSpace(req.FirstName)
	if n := utf8.RuneCountInString(name); n < 1 || n > MaxNameRunes || hasControl(name) {
		return ProfileInput{}, invalid("first name must be 1-40 characters")
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return ProfileInput{}, invalid("birth date must be YYYY-MM-DD")
	}
	if age := AgeOn(birth, s.now()); age < MinAge {
		return ProfileInput{}, invalid("you must be at least 18 years old")
	} else if age > MaxAge {
		return ProfileInput{}, invalid("invalid birth date")
	}
	if !validGenders[req.Gender] {
		return ProfileInput{}, invalid("invalid gender")
	}
	bio := strings.TrimSpace(req.Bio)
	if utf8.RuneCountInString(bio) > MaxBioRunes {
		return ProfileInput{}, invalid("bio is limited to 500 characters")
	}
	city := strings.TrimSpace(req.City)
	if utf8.RuneCountInString(city) > MaxCityRunes || hasControl(city) {
		return ProfileInput{}, invalid("city is limited to 80 characters")
	}
	if (req.Latitude == nil) != (req.Longitude == nil) {
		return ProfileInput{}, invalid("latitude and longitude go together")
	}
	var lat, lon *float64
	if req.Latitude != nil {
		if *req.Latitude < -90 || *req.Latitude > 90 || *req.Longitude < -180 || *req.Longitude > 180 {
			return ProfileInput{}, invalid("invalid coordinates")
		}
		la, lo := RoundCoordinate(*req.Latitude), RoundCoordinate(*req.Longitude)
		lat, lon = &la, &lo
	}
	slugs, err := dedupe(req.Interests, MaxInterests, "interests")
	if err != nil {
		return ProfileInput{}, err
	}
	interestedIn, err := dedupe(req.Preferences.InterestedIn, 3, "interestedIn")
	if err != nil || len(interestedIn) == 0 {
		return ProfileInput{}, invalid("choose who you want to meet")
	}
	for _, g := range interestedIn {
		if !validGenders[g] {
			return ProfileInput{}, invalid("invalid preference")
		}
	}
	p := req.Preferences
	if p.AgeMin < MinAge || p.AgeMax > MaxAge || p.AgeMin > p.AgeMax {
		return ProfileInput{}, invalid("age range must be within 18-99")
	}
	if p.MaxDistanceKm < 1 || (p.MaxDistanceKm > 500 && p.MaxDistanceKm != AnyDistance) {
		return ProfileInput{}, invalid("invalid distance")
	}
	return ProfileInput{
		FirstName: name, BirthDate: birth, Gender: req.Gender, Bio: bio, City: city,
		Latitude: lat, Longitude: lon,
		ShowDistance:  req.ShowDistance == nil || *req.ShowDistance,
		Discoverable:  req.Discoverable == nil || *req.Discoverable,
		InterestSlugs: slugs,
		Preferences:   Preferences{InterestedIn: interestedIn, AgeMin: p.AgeMin, AgeMax: p.AgeMax, MaxDistanceKm: p.MaxDistanceKm},
	}, nil
}

func dedupe(in []string, max int, field string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || len(v) > 40 {
			return nil, invalid("invalid " + field)
		}
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	if len(out) > max {
		return nil, invalid("too many " + field)
	}
	return out, nil
}

func hasControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func (s *Service) AddPhoto(ctx context.Context, userID string, raw []byte) (PhotoView, error) {
	img, err := ProcessImage(raw)
	if err != nil {
		return PhotoView{}, invalid(err.Error())
	}
	key, err := s.files.Put(img.Data)
	if err != nil {
		return PhotoView{}, err
	}
	p, err := s.repo.AddPhoto(ctx, userID, key, len(img.Data), img.Width, img.Height)
	if err != nil {
		s.files.Delete(key)
		if errors.Is(err, ErrPhotoLimit) {
			return PhotoView{}, invalid("you can upload up to 6 photos")
		}
		return PhotoView{}, err
	}
	return PhotoView{ID: p.ID, URL: s.signer.URL(p.ID)}, nil
}

func (s *Service) DeletePhoto(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.DeletePhoto(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.files.Delete(key)
	return nil
}

func (s *Service) ReorderPhotos(ctx context.Context, userID string, ids []string) error {
	if err := s.repo.ReorderPhotos(ctx, userID, ids); err != nil {
		if errors.Is(err, ErrBadPhotoSet) {
			return invalid("ids must list all of your photos exactly once")
		}
		return err
	}
	return nil
}

// Cards builds display cards for the given users in the given order, batching all queries.
// Users without a profile are skipped. sharedInterests is computed against the viewer.
func (s *Service) Cards(ctx context.Context, viewerID string, ids []string) ([]Card, error) {
	if len(ids) == 0 {
		return []Card{}, nil
	}
	rows, err := s.repo.Cards(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	all := append([]string{viewerID}, ids...)
	interests, err := s.repo.InterestsFor(ctx, all)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.PhotosFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	mine := map[string]bool{}
	for _, it := range interests[viewerID] {
		mine[it.Slug] = true
	}
	out := make([]Card, 0, len(ids))
	for _, id := range ids {
		row, ok := rows[id]
		if !ok {
			continue
		}
		card := Card{
			UserID: id, FirstName: row.FirstName, Age: row.Age, Gender: row.Gender, Bio: row.Bio, City: row.City,
			Photos: s.photoViews(photos[id]), Interests: interests[id],
		}
		if card.Interests == nil {
			card.Interests = []Interest{}
		}
		if row.DistanceKm != nil {
			d := ApproxDistance(*row.DistanceKm)
			card.DistanceKm = &d
		}
		for _, it := range card.Interests {
			if mine[it.Slug] {
				card.SharedInterests++
			}
		}
		out = append(out, card)
	}
	return out, nil
}

// ViewProfile returns another user's card if the viewer is allowed to see it.
func (s *Service) ViewProfile(ctx context.Context, viewerID, targetID string) (Card, error) {
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return Card{}, err
	}
	if !ok {
		return Card{}, ErrForbidden
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil || len(cards) == 0 {
		if err == nil {
			err = ErrForbidden
		}
		return Card{}, err
	}
	return cards[0], nil
}

// PhotoBytes returns the stored bytes for a photo id.
func (s *Service) PhotoKey(ctx context.Context, photoID string) (string, error) {
	return s.repo.PhotoKey(ctx, photoID)
}
