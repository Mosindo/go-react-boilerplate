package profiles

import "time"

var (
	Genders           = []string{"woman", "man", "nonbinary"}
	RelationshipGoals = []string{"long_term", "short_term", "friendship", "undecided"}
)

const (
	MinAge          = 18
	MaxAge          = 99
	MaxInterests    = 10
	MinDistanceKm   = 1
	MaxDistanceKm   = 500
	maxFirstName    = 40
	maxBio          = 500
	maxJobTitle     = 60
	maxCity         = 60
	minHeightCm     = 120
	maxHeightCm     = 230
	birthdateLayout = "2006-01-02"
)

type Interest struct {
	ID       int    `json:"id"`
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

// Profile is the owner's private view of their own profile.
type Profile struct {
	UserID           string
	FirstName        string
	Birthdate        time.Time
	Gender           string
	Bio              string
	JobTitle         string
	City             string
	HeightCm         *int
	RelationshipGoal *string
	Latitude         *float64
	Longitude        *float64
	Discoverable     bool
	Interests        []Interest
	Photos           []PhotoRef
	Preferences      *Preferences
}

// Card is what other members can see. It never contains the birthdate,
// e-mail or coordinates: only the age and an approximate distance.
type Card struct {
	UserID           string
	FirstName        string
	Age              int
	Gender           string
	Bio              string
	JobTitle         string
	City             string
	HeightCm         *int
	RelationshipGoal *string
	DistanceKm       *int
	Interests        []CardInterest
	Photos           []PhotoRef
	IsDemo           bool
}

type CardInterest struct {
	ID     int    `json:"id"`
	Label  string `json:"label"`
	Shared bool   `json:"shared"`
}

type UpsertProfileInput struct {
	FirstName        string
	Birthdate        string
	Gender           string
	Bio              string
	JobTitle         string
	City             string
	HeightCm         *int
	RelationshipGoal *string
	InterestIDs      []int
}

// ---- HTTP payloads ----

type UpsertProfileRequest struct {
	FirstName        string  `json:"firstName" binding:"required,max=80"`
	Birthdate        string  `json:"birthdate" binding:"required,len=10"`
	Gender           string  `json:"gender" binding:"required,max=20"`
	Bio              string  `json:"bio" binding:"max=1000"`
	JobTitle         string  `json:"jobTitle" binding:"max=120"`
	City             string  `json:"city" binding:"max=120"`
	HeightCm         *int    `json:"heightCm"`
	RelationshipGoal *string `json:"relationshipGoal"`
	InterestIDs      []int   `json:"interestIds" binding:"max=20"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,min=1,max=3"`
	AgeMin        int      `json:"ageMin" binding:"required"`
	AgeMax        int      `json:"ageMax" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}

type VisibilityRequest struct {
	Discoverable *bool `json:"discoverable" binding:"required"`
}

type ProfileResponse struct {
	UserID           string       `json:"userId"`
	FirstName        string       `json:"firstName"`
	Birthdate        string       `json:"birthdate"`
	Age              int          `json:"age"`
	Gender           string       `json:"gender"`
	Bio              string       `json:"bio"`
	JobTitle         string       `json:"jobTitle"`
	City             string       `json:"city"`
	HeightCm         *int         `json:"heightCm"`
	RelationshipGoal *string      `json:"relationshipGoal"`
	HasLocation      bool         `json:"hasLocation"`
	Discoverable     bool         `json:"discoverable"`
	Interests        []Interest   `json:"interests"`
	Photos           []PhotoRef   `json:"photos"`
	Preferences      *Preferences `json:"preferences"`
}

type CardResponse struct {
	UserID           string         `json:"userId"`
	FirstName        string         `json:"firstName"`
	Age              int            `json:"age"`
	Gender           string         `json:"gender"`
	Bio              string         `json:"bio"`
	JobTitle         string         `json:"jobTitle"`
	City             string         `json:"city"`
	HeightCm         *int           `json:"heightCm"`
	RelationshipGoal *string        `json:"relationshipGoal"`
	DistanceKm       *int           `json:"distanceKm"`
	Interests        []CardInterest `json:"interests"`
	Photos           []PhotoRef     `json:"photos"`
	IsDemo           bool           `json:"isDemo"`
}

func ToCardResponse(card Card) CardResponse {
	interests := card.Interests
	if interests == nil {
		interests = []CardInterest{}
	}
	photos := card.Photos
	if photos == nil {
		photos = []PhotoRef{}
	}
	return CardResponse{
		UserID:           card.UserID,
		FirstName:        card.FirstName,
		Age:              card.Age,
		Gender:           card.Gender,
		Bio:              card.Bio,
		JobTitle:         card.JobTitle,
		City:             card.City,
		HeightCm:         card.HeightCm,
		RelationshipGoal: card.RelationshipGoal,
		DistanceKm:       card.DistanceKm,
		Interests:        interests,
		Photos:           photos,
		IsDemo:           card.IsDemo,
	}
}

// CardSummary is the compact form used in conversation lists.
type CardSummary struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	Age       int     `json:"age"`
	PhotoURL  *string `json:"photoUrl"`
	IsDemo    bool    `json:"isDemo"`
}

func ToCardSummary(card Card) CardSummary {
	summary := CardSummary{UserID: card.UserID, FirstName: card.FirstName, Age: card.Age, IsDemo: card.IsDemo}
	if len(card.Photos) > 0 {
		url := card.Photos[0].URL
		summary.PhotoURL = &url
	}
	return summary
}
