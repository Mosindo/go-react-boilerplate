package profiles

import (
	"time"

	"example.com/api/internal/features/photos"
)

const (
	MinAge              = 18
	MaxAge              = 99
	MaxInterestsPerUser = 10
	// MinDistanceKm is also the floor applied to distances when filtering
	// discovery, so a tiny radius cannot be used to pinpoint a member.
	MinDistanceKm = 5
)

var (
	Genders           = []string{"woman", "man", "nonbinary"}
	RelationshipGoals = []string{"long_term", "short_term", "friendship", "unsure"}
)

type Profile struct {
	UserID           string
	FirstName        string
	Birthdate        time.Time
	Gender           string
	Bio              string
	JobTitle         string
	HeightCm         *int
	RelationshipGoal string
	City             string
	Latitude         *float64
	Longitude        *float64
	IsDiscoverable   bool
	ShowDistance     bool
}

type Preferences struct {
	InterestedIn  []string
	MinAge        int
	MaxAge        int
	MaxDistanceKm int
}

type Interest struct {
	ID       int    `json:"id"`
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

// --- Requests ---------------------------------------------------------------

type UpsertProfileRequest struct {
	FirstName        string `json:"firstName" binding:"required,max=80"`
	Birthdate        string `json:"birthdate" binding:"required,datetime=2006-01-02"`
	Gender           string `json:"gender" binding:"required"`
	Bio              string `json:"bio" binding:"max=2000"`
	JobTitle         string `json:"jobTitle" binding:"max=200"`
	HeightCm         *int   `json:"heightCm"`
	RelationshipGoal string `json:"relationshipGoal"`
	City             string `json:"city" binding:"max=200"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      *string  `json:"city" binding:"omitempty,max=200"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,min=1,max=3"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type UpdatePrivacyRequest struct {
	IsDiscoverable *bool `json:"isDiscoverable" binding:"required"`
	ShowDistance   *bool `json:"showDistance" binding:"required"`
}

type UpdateInterestsRequest struct {
	InterestIDs []int `json:"interestIds" binding:"max=10"`
}

// --- Responses --------------------------------------------------------------

type ProfileResponse struct {
	FirstName        string `json:"firstName"`
	Birthdate        string `json:"birthdate"`
	Age              int    `json:"age"`
	Gender           string `json:"gender"`
	Bio              string `json:"bio"`
	JobTitle         string `json:"jobTitle"`
	HeightCm         *int   `json:"heightCm"`
	RelationshipGoal string `json:"relationshipGoal"`
	City             string `json:"city"`
	HasLocation      bool   `json:"hasLocation"`
	IsDiscoverable   bool   `json:"isDiscoverable"`
	ShowDistance     bool   `json:"showDistance"`
}

type PreferencesResponse struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// MyProfileResponse drives onboarding and the profile tab in one call.
type MyProfileResponse struct {
	Profile     *ProfileResponse       `json:"profile"`
	Preferences PreferencesResponse    `json:"preferences"`
	Interests   []Interest             `json:"interests"`
	Photos      []photos.PhotoResponse `json:"photos"`
	IsComplete  bool                   `json:"isComplete"`
	Missing     []string               `json:"missing"`
}

type CardInterest struct {
	Interest
	Shared bool `json:"shared"`
}

// Card is the public view of another member's profile. It never contains
// email, birthdate or coordinates: only age and an approximate distance.
type Card struct {
	UserID           string                 `json:"userId"`
	FirstName        string                 `json:"firstName"`
	Age              int                    `json:"age"`
	Gender           string                 `json:"gender"`
	Bio              string                 `json:"bio"`
	JobTitle         string                 `json:"jobTitle"`
	HeightCm         *int                   `json:"heightCm"`
	RelationshipGoal string                 `json:"relationshipGoal"`
	City             string                 `json:"city"`
	DistanceKm       *int                   `json:"distanceKm"`
	Interests        []CardInterest         `json:"interests"`
	Photos           []photos.PhotoResponse `json:"photos"`
}

type InterestsResponse struct {
	Interests []Interest `json:"interests"`
}
