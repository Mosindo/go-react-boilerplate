package profiles

import "time"

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
}

func NewPhoto(id string, position int) Photo {
	return Photo{
		ID:       id,
		Position: position,
		URL:      "/photos/" + id + "/file",
		ThumbURL: "/photos/" + id + "/file?size=thumb",
	}
}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type OwnProfile struct {
	UserID       string      `json:"userId"`
	DisplayName  string      `json:"displayName"`
	BirthDate    string      `json:"birthDate"`
	Age          int         `json:"age"`
	Gender       string      `json:"gender"`
	Bio          string      `json:"bio"`
	City         string      `json:"city"`
	Interests    []Interest  `json:"interests"`
	Photos       []Photo     `json:"photos"`
	ShowDistance bool        `json:"showDistance"`
	Discoverable bool        `json:"discoverable"`
	HasLocation  bool        `json:"hasLocation"`
	Preferences  Preferences `json:"preferences"`
	Completed    bool        `json:"completed"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
}

type PublicProfile struct {
	UserID          string     `json:"userId"`
	DisplayName     string     `json:"displayName"`
	Age             int        `json:"age"`
	Gender          string     `json:"gender"`
	Bio             string     `json:"bio"`
	City            string     `json:"city"`
	Interests       []Interest `json:"interests"`
	Photos          []Photo    `json:"photos"`
	DistanceKm      *int       `json:"distanceKm"`
	CommonInterests int        `json:"commonInterests"`
}

// SaveProfileRequest is the body of PUT /profile.
type SaveProfileRequest struct {
	DisplayName  string   `json:"displayName"`
	BirthDate    string   `json:"birthDate"`
	Gender       string   `json:"gender"`
	Bio          string   `json:"bio"`
	City         string   `json:"city"`
	Interests    []string `json:"interests"`
	ShowDistance *bool    `json:"showDistance"`
	Discoverable *bool    `json:"discoverable"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

// ProfileInput is the validated, normalised form handed to the repository.
type ProfileInput struct {
	UserID       string
	DisplayName  string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         string
	InterestSlug []string
	ShowDistance bool
	Discoverable bool
}
