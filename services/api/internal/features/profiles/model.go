package profiles

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type Privacy struct {
	ShowDistance bool `json:"showDistance"`
	Discoverable bool `json:"discoverable"`
}

// MyProfile is the private view of the caller's own profile.
type MyProfile struct {
	UserID      string       `json:"userId"`
	FirstName   string       `json:"firstName"`
	BirthDate   string       `json:"birthDate"`
	Age         int          `json:"age"`
	Gender      string       `json:"gender"`
	Bio         string       `json:"bio"`
	City        string       `json:"city"`
	HasLocation bool         `json:"hasLocation"`
	Interests   []Interest   `json:"interests"`
	Photos      []PhotoRef   `json:"photos"`
	Preferences *Preferences `json:"preferences"`
	Privacy     Privacy      `json:"privacy"`
	Complete    bool         `json:"complete"`
	Missing     []string     `json:"missing"`
}

// Card is the public view of someone else's profile. It never carries
// coordinates, only an approximate distance bucket.
type Card struct {
	ID         string     `json:"id"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []PhotoRef `json:"photos"`
}

type UpsertProfileRequest struct {
	FirstName string   `json:"firstName"`
	BirthDate string   `json:"birthDate"`
	Gender    string   `json:"gender"`
	Bio       string   `json:"bio"`
	City      string   `json:"city"`
	Interests []string `json:"interests"`
}

type LocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type PrivacyRequest struct {
	ShowDistance *bool `json:"showDistance"`
	Discoverable *bool `json:"discoverable"`
}
