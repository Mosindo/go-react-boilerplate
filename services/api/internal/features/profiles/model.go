package profiles

import "time"

const (
	GenderMan       = "man"
	GenderWoman     = "woman"
	GenderNonBinary = "non_binary"

	MinAge          = 18
	MaxAge          = 99
	MaxInterests    = 10
	MaxBioRunes     = 500
	MaxFirstName    = 40
	MaxCityRunes    = 80
	MaxDistanceKm   = 500
	DefaultDistance = 50
)

var validGenders = map[string]struct{}{GenderMan: {}, GenderWoman: {}, GenderNonBinary: {}}

type Profile struct {
	UserID       string
	FirstName    string
	Birthdate    time.Time
	Gender       string
	Bio          string
	City         string
	HasLocation  bool
	ShowAge      bool
	ShowDistance bool
	Discoverable bool
}

type Preferences struct {
	InterestedIn  []string
	MinAge        int
	MaxAge        int
	MaxDistanceKm int
}

type Interest struct {
	ID    int16
	Slug  string
	Label string
}

type PhotoRef struct {
	ID       string
	Position int
}

// PublicProfile is what another user may see. It deliberately has no email,
// no coordinates and no exact distance.
type PublicProfile struct {
	UserID     string
	FirstName  string
	Age        *int
	Gender     string
	Bio        string
	City       string
	DistanceKm *int
	Interests  []Interest
	Photos     []PhotoRef
}

type Onboarding struct {
	Complete bool
	Missing  []string
}

type SelfView struct {
	Profile     *Profile
	Preferences *Preferences
	Interests   []Interest
	Photos      []PhotoRef
	Onboarding  Onboarding
}

// --- HTTP payloads ---

type UpsertProfileRequest struct {
	FirstName string `json:"firstName" binding:"required,max=200"`
	Birthdate string `json:"birthdate" binding:"required,max=10"`
	Gender    string `json:"gender" binding:"required,max=20"`
	Bio       string `json:"bio" binding:"max=4000"`
}

type PrivacyRequest struct {
	ShowAge      *bool `json:"showAge" binding:"required"`
	ShowDistance *bool `json:"showDistance" binding:"required"`
	Discoverable *bool `json:"discoverable" binding:"required"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,min=1,max=3"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required,min=-90,max=90"`
	Longitude *float64 `json:"longitude" binding:"required,min=-180,max=180"`
	City      string   `json:"city" binding:"max=400"`
}

type InterestsRequest struct {
	InterestIDs []int16 `json:"interestIds" binding:"max=50"`
}

type InterestResponse struct {
	ID     int16  `json:"id"`
	Slug   string `json:"slug"`
	Label  string `json:"label"`
	Shared bool   `json:"shared,omitempty"`
}

type PhotoResponse struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

type ProfileResponse struct {
	FirstName    string `json:"firstName"`
	Birthdate    string `json:"birthdate"`
	Age          int    `json:"age"`
	Gender       string `json:"gender"`
	Bio          string `json:"bio"`
	City         string `json:"city"`
	HasLocation  bool   `json:"hasLocation"`
	ShowAge      bool   `json:"showAge"`
	ShowDistance bool   `json:"showDistance"`
	Discoverable bool   `json:"discoverable"`
}

type PreferencesResponse struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type OnboardingResponse struct {
	Complete bool     `json:"complete"`
	Missing  []string `json:"missing"`
}

type SelfResponse struct {
	Profile     *ProfileResponse     `json:"profile"`
	Preferences *PreferencesResponse `json:"preferences"`
	Interests   []InterestResponse   `json:"interests"`
	Photos      []PhotoResponse      `json:"photos"`
	Onboarding  OnboardingResponse   `json:"onboarding"`
}

type PublicProfileResponse struct {
	UserID     string             `json:"userId"`
	FirstName  string             `json:"firstName"`
	Age        *int               `json:"age,omitempty"`
	Gender     string             `json:"gender"`
	Bio        string             `json:"bio"`
	City       string             `json:"city,omitempty"`
	DistanceKm *int               `json:"distanceKm,omitempty"`
	Interests  []InterestResponse `json:"interests"`
	Photos     []PhotoResponse    `json:"photos"`
}

// PhotoURL is the authenticated content route of a photo.
func PhotoURL(id string) string { return "/photos/" + id + "/content" }

func ToPhotoResponses(refs []PhotoRef) []PhotoResponse {
	out := make([]PhotoResponse, 0, len(refs))
	for _, p := range refs {
		out = append(out, PhotoResponse{ID: p.ID, Position: p.Position, URL: PhotoURL(p.ID)})
	}
	return out
}

func ToInterestResponses(items []Interest, shared map[int16]struct{}) []InterestResponse {
	out := make([]InterestResponse, 0, len(items))
	for _, i := range items {
		_, isShared := shared[i.ID]
		out = append(out, InterestResponse{ID: i.ID, Slug: i.Slug, Label: i.Label, Shared: isShared})
	}
	return out
}

func ToPublicResponse(p PublicProfile, shared map[int16]struct{}) PublicProfileResponse {
	return PublicProfileResponse{
		UserID:     p.UserID,
		FirstName:  p.FirstName,
		Age:        p.Age,
		Gender:     p.Gender,
		Bio:        p.Bio,
		City:       p.City,
		DistanceKm: p.DistanceKm,
		Interests:  ToInterestResponses(p.Interests, shared),
		Photos:     ToPhotoResponses(p.Photos),
	}
}
