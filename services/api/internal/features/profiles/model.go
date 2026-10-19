package profiles

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	GenderMan       = "man"
	GenderWoman     = "woman"
	GenderNonBinary = "non_binary"

	MinAge       = 18
	MaxAge       = 99
	MaxBioRunes  = 500
	MaxNameRunes = 40
	MaxCityRunes = 80
	MaxInterests = 10
	MaxPhotos    = 6
)

var (
	ErrProfileNotFound   = errors.New("profile not found")
	ErrProfileIncomplete = errors.New("profile incomplete")
	ErrInvalidInput      = errors.New("invalid profile input")
	ErrUnderage          = errors.New("must be at least 18 years old")
	ErrPhotoNotFound     = errors.New("photo not found")
	ErrPhotoLimit        = errors.New("photo limit reached")
	ErrInvalidImage      = errors.New("invalid image")
	ErrImageTooLarge     = errors.New("image too large")
	ErrForbidden         = errors.New("forbidden")
)

var validGenders = map[string]struct{}{GenderMan: {}, GenderWoman: {}, GenderNonBinary: {}}

type Interest struct {
	ID    int16  `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

// PublicProfile is the only shape in which another user's profile is ever
// exposed: no email, birth date or coordinates.
type PublicProfile struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Bio        string     `json:"bio"`
	City       string     `json:"city,omitempty"`
	DistanceKm *int       `json:"distanceKm,omitempty"`
	Interests  []Interest `json:"interests"`
	Photos     []PhotoRef `json:"photos"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
	ShowDistance  bool     `json:"showDistance"`
}

// OwnProfile is what a user sees about themselves.
type OwnProfile struct {
	PublicProfile
	BirthDate   string       `json:"birthDate"`
	Gender      string       `json:"gender"`
	IsVisible   bool         `json:"isVisible"`
	HasLocation bool         `json:"hasLocation"`
	Preferences *Preferences `json:"preferences"`
}

type Coordinates struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type UpsertProfileRequest struct {
	FirstName string       `json:"firstName"`
	BirthDate string       `json:"birthDate"`
	Gender    string       `json:"gender"`
	Bio       string       `json:"bio"`
	City      string       `json:"city"`
	Location  *Coordinates `json:"location"`
	Interests []int16      `json:"interests"`
	IsVisible *bool        `json:"isVisible"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
	ShowDistance  bool     `json:"showDistance"`
}

type ReorderPhotosRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,max=9"`
}

// UpsertInput is the validated, normalised form of UpsertProfileRequest.
type UpsertInput struct {
	FirstName string
	BirthDate time.Time
	Gender    string
	Bio       string
	City      string
	Location  *Coordinates
	Interests []int16
	IsVisible *bool
}

// StoredPhoto is a photo row including its storage key.
type StoredPhoto struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
}

type cardJSON struct {
	photos    []byte
	interests []byte
}

func (c cardJSON) decode(p *PublicProfile) error {
	p.Photos = []PhotoRef{}
	p.Interests = []Interest{}
	if err := json.Unmarshal(c.photos, &p.Photos); err != nil {
		return err
	}
	return json.Unmarshal(c.interests, &p.Interests)
}
