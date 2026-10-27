package profiles

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/imaging"
	"example.com/api/internal/platform/storage"
)

var (
	ErrNotFound        = errors.New("profile not found")
	ErrPhotoNotFound   = errors.New("photo not found")
	ErrPhotoLimit      = errors.New("photo limit reached")
	ErrProfileRequired = errors.New("create your profile before adding photos")
	ErrInvalidImage    = errors.New("invalid image")
	ErrUnderage        = errors.New("you must be at least 18 years old")
	ErrBadPhotoOrder   = errors.New("photo order must list each of your photos exactly once")
	ErrUnknownInterest = errors.New("unknown interest")
)

// ValidationError carries a user-facing message for a rejected input.
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return ValidationError{Message: fmt.Sprintf(format, args...)}
}

var firstNamePattern = regexp.MustCompile(`^[\p{L}][\p{L}\p{M} '\-]*$`)

type Service struct {
	repo  Repository
	store *storage.Store
	now   func() time.Time
}

func NewService(repo Repository, store *storage.Store) *Service {
	return &Service{repo: repo, store: store, now: time.Now}
}

func (s *Service) Get(ctx context.Context, userID string) (Profile, error) {
	p, err := s.repo.GetProfile(ctx, userID)
	if errors.Is(err, ErrRepoNotFound) {
		return Profile{}, ErrNotFound
	}
	return p, err
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (Profile, error) {
	in, err := s.validateProfile(req)
	if err != nil {
		return Profile{}, err
	}
	if err := s.repo.UpsertProfile(ctx, userID, in); err != nil {
		if errors.Is(err, ErrRepoUnknownInterest) {
			return Profile{}, ErrUnknownInterest
		}
		return Profile{}, err
	}
	return s.Get(ctx, userID)
}

func (s *Service) validateProfile(req UpsertProfileRequest) (ProfileInput, error) {
	name := strings.Join(strings.Fields(req.FirstName), " ")
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > MaxFirstName || !firstNamePattern.MatchString(name) {
		return ProfileInput{}, invalid("first name must be 1-%d letters", MaxFirstName)
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return ProfileInput{}, invalid("birth date must use the YYYY-MM-DD format")
	}
	age := AgeAt(birth, s.now())
	if age < MinAge {
		return ProfileInput{}, ErrUnderage
	}
	if age > MaxAge {
		return ProfileInput{}, invalid("birth date is out of range")
	}
	gender := strings.TrimSpace(req.Gender)
	if _, ok := validGenders[gender]; !ok {
		return ProfileInput{}, invalid("gender must be one of man, woman, non_binary")
	}
	bio := strings.TrimSpace(req.Bio)
	if utf8.RuneCountInString(bio) > MaxBioRunes {
		return ProfileInput{}, invalid("bio must be at most %d characters", MaxBioRunes)
	}
	if hasControlChars(bio, true) {
		return ProfileInput{}, invalid("bio contains invalid characters")
	}
	city := strings.Join(strings.Fields(req.City), " ")
	if utf8.RuneCountInString(city) > MaxCityRunes || hasControlChars(city, false) {
		return ProfileInput{}, invalid("city must be at most %d characters", MaxCityRunes)
	}

	in := ProfileInput{FirstName: name, BirthDate: birth, Gender: gender, Bio: bio, City: city}

	if (req.Latitude == nil) != (req.Longitude == nil) {
		return ProfileInput{}, invalid("latitude and longitude must be provided together")
	}
	if req.Latitude != nil {
		lat, lng := *req.Latitude, *req.Longitude
		if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			return ProfileInput{}, invalid("coordinates are out of range")
		}
		in.Latitude, in.Longitude = ptr(coarse(lat)), ptr(coarse(lng))
	}

	seen := map[string]struct{}{}
	for _, slug := range req.Interests {
		slug = strings.ToLower(strings.TrimSpace(slug))
		if slug == "" {
			continue
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		in.Interests = append(in.Interests, slug)
	}
	if len(in.Interests) > MaxInterests {
		return ProfileInput{}, invalid("choose at most %d interests", MaxInterests)
	}
	return in, nil
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (Profile, error) {
	seen := map[string]struct{}{}
	var interested []string
	for _, g := range req.InterestedIn {
		g = strings.TrimSpace(g)
		if _, ok := validGenders[g]; !ok {
			return Profile{}, invalid("interestedIn must contain man, woman or non_binary")
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			interested = append(interested, g)
		}
	}
	if len(interested) == 0 {
		return Profile{}, invalid("choose at least one gender you are interested in")
	}
	if req.AgeMin < MinAge || req.AgeMax > MaxAge || req.AgeMin > req.AgeMax {
		return Profile{}, invalid("age range must be within %d-%d with min <= max", MinAge, MaxAge)
	}
	if req.MaxDistanceKM < 1 || req.MaxDistanceKM > MaxDistanceKM {
		return Profile{}, invalid("max distance must be between 1 and %d km", MaxDistanceKM)
	}
	err := s.repo.UpdatePreferences(ctx, userID, Preferences{
		InterestedIn: interested, AgeMin: req.AgeMin, AgeMax: req.AgeMax, MaxDistanceKM: req.MaxDistanceKM,
	})
	if errors.Is(err, ErrRepoNotFound) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	return s.Get(ctx, userID)
}

func (s *Service) UpdatePrivacy(ctx context.Context, userID string, req UpdatePrivacyRequest) (Profile, error) {
	if req.Discoverable == nil && req.ShowDistance == nil {
		return Profile{}, invalid("nothing to update")
	}
	err := s.repo.UpdatePrivacy(ctx, userID, req.Discoverable, req.ShowDistance)
	if errors.Is(err, ErrRepoNotFound) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	return s.Get(ctx, userID)
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) AddPhoto(ctx context.Context, userID string, raw []byte) (Photo, error) {
	img, err := imaging.Normalize(raw)
	if err != nil {
		return Photo{}, mapImageError(err)
	}
	id, err := newUUID()
	if err != nil {
		return Photo{}, err
	}
	key := id + ".jpg"
	if err := s.store.Save(key, img.Data); err != nil {
		return Photo{}, err
	}
	stored, err := s.repo.AddPhoto(ctx, userID, id, key, img.Width, img.Height)
	if err != nil {
		_ = s.store.Delete(key)
		switch {
		case errors.Is(err, ErrRepoPhotoLimit):
			return Photo{}, ErrPhotoLimit
		case errors.Is(err, ErrRepoNotFound):
			return Photo{}, ErrProfileRequired
		}
		return Photo{}, err
	}
	return toPhoto(stored), nil
}

func (s *Service) ReplacePhoto(ctx context.Context, userID, photoID string, raw []byte) (Photo, error) {
	img, err := imaging.Normalize(raw)
	if err != nil {
		return Photo{}, mapImageError(err)
	}
	id, err := newUUID()
	if err != nil {
		return Photo{}, err
	}
	key := id + ".jpg"
	if err := s.store.Save(key, img.Data); err != nil {
		return Photo{}, err
	}
	oldKey, err := s.repo.ReplacePhoto(ctx, userID, photoID, key, img.Width, img.Height)
	if err != nil {
		_ = s.store.Delete(key)
		if errors.Is(err, ErrRepoNotFound) {
			return Photo{}, ErrPhotoNotFound
		}
		return Photo{}, err
	}
	_ = s.store.Delete(oldKey)
	return Photo{ID: photoID, URL: PhotoURL(photoID), Width: img.Width, Height: img.Height}, nil
}

func (s *Service) DeletePhoto(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.DeletePhoto(ctx, userID, photoID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return ErrPhotoNotFound
		}
		return err
	}
	_ = s.store.Delete(key)
	return nil
}

func (s *Service) ReorderPhotos(ctx context.Context, userID string, ids []string) (Profile, error) {
	if err := s.repo.ReorderPhotos(ctx, userID, ids); err != nil {
		if errors.Is(err, ErrRepoBadPhotoOrder) {
			return Profile{}, ErrBadPhotoOrder
		}
		return Profile{}, err
	}
	return s.Get(ctx, userID)
}

// OpenPhoto authorizes a read and returns a stream of the stored JPEG.
func (s *Service) OpenPhoto(ctx context.Context, photoID, viewerID string) (StoredPhoto, io.ReadSeekCloser, error) {
	photo, err := s.repo.FindPhotoForViewer(ctx, photoID, viewerID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return StoredPhoto{}, nil, ErrPhotoNotFound
		}
		return StoredPhoto{}, nil, err
	}
	f, err := s.store.Open(photo.StorageKey)
	if err != nil {
		return StoredPhoto{}, nil, ErrPhotoNotFound
	}
	return photo, f, nil
}

// PrepareUserDeletion implements auth.DeletionHook: stored files are removed
// only after the user row (and therefore its photo rows) is deleted.
func (s *Service) PrepareUserDeletion(ctx context.Context, userID string) (func(), error) {
	keys, err := s.repo.PhotoKeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	return func() {
		for _, k := range keys {
			_ = s.store.Delete(k)
		}
	}, nil
}

func toPhoto(p StoredPhoto) Photo {
	return Photo{ID: p.ID, URL: PhotoURL(p.ID), Position: p.Position, Width: p.Width, Height: p.Height}
}

func mapImageError(err error) error {
	switch {
	case errors.Is(err, imaging.ErrUnsupported):
		return invalid("only JPEG and PNG pictures are accepted")
	case errors.Is(err, imaging.ErrTooLarge):
		return invalid("picture dimensions are too large")
	case errors.Is(err, imaging.ErrTooSmall):
		return invalid("picture must be at least %d px on each side", imaging.MinSide)
	default:
		return ErrInvalidImage
	}
}

// AgeAt returns the age in full years at the given instant.
func AgeAt(birth, now time.Time) int {
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return age
}

func coarse(v float64) float64 { return math.Round(v*coordPrecisionF) / coordPrecisionF }

func ptr[T any](v T) *T { return &v }

func hasControlChars(s string, allowNewline bool) bool {
	for _, r := range s {
		if r == '\n' && allowNewline {
			continue
		}
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
