package profiles

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/storage"
	"github.com/google/uuid"
)

type Service struct {
	repo  Repository
	store storage.Store
	now   func() time.Time
}

func NewService(repo Repository, store storage.Store) *Service {
	return &Service{repo: repo, store: store, now: time.Now}
}

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	return s.repo.GetOwn(ctx, userID)
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// AgeAt returns the full years elapsed between birth and now.
func AgeAt(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

func cleanText(s string, maxRunes int, allowNewlines bool) (string, bool) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > maxRunes || !utf8.ValidString(s) {
		return "", false
	}
	for _, r := range s {
		if unicode.IsControl(r) && !(allowNewlines && (r == '\n' || r == '\t')) {
			return "", false
		}
	}
	return s, true
}

func (s *Service) validateUpsert(ctx context.Context, req UpsertProfileRequest) (UpsertInput, error) {
	var in UpsertInput
	name, ok := cleanText(req.FirstName, MaxNameRunes, false)
	if !ok || name == "" {
		return in, ErrInvalidInput
	}
	in.FirstName = name

	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return in, ErrInvalidInput
	}
	age := AgeAt(birth, s.now().UTC())
	if age < MinAge {
		return in, ErrUnderage
	}
	if age > 120 {
		return in, ErrInvalidInput
	}
	in.BirthDate = birth

	if _, ok := validGenders[req.Gender]; !ok {
		return in, ErrInvalidInput
	}
	in.Gender = req.Gender

	if in.Bio, ok = cleanText(req.Bio, MaxBioRunes, true); !ok {
		return in, ErrInvalidInput
	}
	if in.City, ok = cleanText(req.City, MaxCityRunes, false); !ok {
		return in, ErrInvalidInput
	}

	if req.Location != nil {
		if req.Location.Lat < -90 || req.Location.Lat > 90 || req.Location.Lng < -180 || req.Location.Lng > 180 {
			return in, ErrInvalidInput
		}
		// Only a coarse (~1 km) position is ever persisted.
		in.Location = &Coordinates{Lat: SnapCoordinate(req.Location.Lat), Lng: SnapCoordinate(req.Location.Lng)}
	}

	seen := make(map[int16]struct{}, len(req.Interests))
	for _, id := range req.Interests {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		in.Interests = append(in.Interests, id)
	}
	if len(in.Interests) > MaxInterests {
		return in, ErrInvalidInput
	}
	if len(in.Interests) > 0 {
		n, err := s.repo.CountInterests(ctx, in.Interests)
		if err != nil {
			return in, err
		}
		if n != len(in.Interests) {
			return in, ErrInvalidInput
		}
	}
	in.IsVisible = req.IsVisible
	return in, nil
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (OwnProfile, error) {
	in, err := s.validateUpsert(ctx, req)
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.Upsert(ctx, userID, in); err != nil {
		return OwnProfile{}, err
	}
	return s.repo.GetOwn(ctx, userID)
}

func ValidatePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	if len(req.InterestedIn) == 0 || len(req.InterestedIn) > len(validGenders) {
		return Preferences{}, ErrInvalidInput
	}
	seen := map[string]struct{}{}
	genders := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		if _, ok := validGenders[g]; !ok {
			return Preferences{}, ErrInvalidInput
		}
		if _, dup := seen[g]; dup {
			continue
		}
		seen[g] = struct{}{}
		genders = append(genders, g)
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, ErrInvalidInput
	}
	if req.MaxDistanceKm != nil && (*req.MaxDistanceKm < 1 || *req.MaxDistanceKm > 500) {
		return Preferences{}, ErrInvalidInput
	}
	return Preferences{
		InterestedIn:  genders,
		MinAge:        req.MinAge,
		MaxAge:        req.MaxAge,
		MaxDistanceKm: req.MaxDistanceKm,
		ShowDistance:  req.ShowDistance,
	}, nil
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (Preferences, error) {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return Preferences{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, prefs); err != nil {
		return Preferences{}, err
	}
	return prefs, nil
}

func (s *Service) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	return s.repo.GetPublic(ctx, viewerID, targetID)
}

func (s *Service) ListPhotos(ctx context.Context, userID string) ([]PhotoRef, error) {
	photos, err := s.repo.ListPhotos(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PhotoRef, 0, len(photos))
	for _, p := range photos {
		out = append(out, PhotoRef{ID: p.ID, Position: p.Position})
	}
	return out, nil
}

// AddPhoto sanitises the upload, stores it and registers it. When position is
// set and occupied the previous photo is replaced.
func (s *Service) AddPhoto(ctx context.Context, userID string, raw []byte, position *int) ([]PhotoRef, error) {
	img, err := ProcessImage(raw)
	if err != nil {
		return nil, err
	}
	key := uuid.NewString() + ".jpg"
	if err := s.store.Put(key, img.Data); err != nil {
		return nil, err
	}
	_, replaced, err := s.repo.AddPhoto(ctx, userID, key, position, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.deleteFile(key)
		return nil, err
	}
	if replaced != nil {
		s.deleteFile(replaced.StorageKey)
	}
	return s.ListPhotos(ctx, userID)
}

func (s *Service) DeletePhoto(ctx context.Context, userID, photoID string) ([]PhotoRef, error) {
	removed, err := s.repo.DeletePhoto(ctx, userID, photoID)
	if err != nil {
		return nil, err
	}
	s.deleteFile(removed.StorageKey)
	return s.ListPhotos(ctx, userID)
}

func (s *Service) ReorderPhotos(ctx context.Context, userID string, ids []string) ([]PhotoRef, error) {
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			return nil, ErrInvalidInput
		}
		if _, dup := seen[id]; dup {
			return nil, ErrInvalidInput
		}
		seen[id] = struct{}{}
	}
	if err := s.repo.ReorderPhotos(ctx, userID, ids); err != nil {
		return nil, err
	}
	return s.ListPhotos(ctx, userID)
}

// OpenPhoto resolves a photo for a viewer, enforcing access rules.
func (s *Service) OpenPhoto(ctx context.Context, viewerID, photoID string) (io.ReadSeekCloser, error) {
	photo, err := s.repo.GetPhotoForViewer(ctx, viewerID, photoID)
	if err != nil {
		return nil, err
	}
	f, err := s.store.Open(photo.StorageKey)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *Service) deleteFile(key string) {
	if err := s.store.Delete(key); err != nil {
		log.Printf(`{"event":"photo_file_delete_failed","key":%q,"error":%q}`, key, err.Error())
	}
}

// PrepareDeletion implements auth.DeletionHook: photo files live outside the
// database, so they are removed once the account rows are gone.
func (s *Service) PrepareDeletion(ctx context.Context, userID string) (func(), error) {
	photos, err := s.repo.ListPhotos(ctx, userID)
	if err != nil {
		return nil, err
	}
	return func() {
		for _, p := range photos {
			s.deleteFile(p.StorageKey)
		}
	}, nil
}
