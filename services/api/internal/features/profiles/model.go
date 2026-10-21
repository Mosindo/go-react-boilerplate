package profiles

// Photo is a photo as exposed to clients: signed, expiring URLs only.
type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
	Position int    `json:"position"`
}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

// OwnProfile is the full profile returned to its owner.
type OwnProfile struct {
	UserID     string   `json:"userId"`
	FirstName  string   `json:"firstName"`
	Age        int      `json:"age"`
	Gender     string   `json:"gender"`
	Bio        string   `json:"bio"`
	City       string   `json:"city"`
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	Occupation string   `json:"occupation"`
	HeightCM   *int     `json:"heightCm"`
	LookingFor string   `json:"lookingFor"`
	Interests  []string `json:"interests"`
	Photos     []Photo  `json:"photos"`
}

// PublicProfile is what other members get to see. It never carries coordinates.
type PublicProfile struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	Occupation string     `json:"occupation"`
	HeightCM   *int       `json:"heightCm"`
	LookingFor string     `json:"lookingFor"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
}

// ProfileInput is the payload to create or update the caller's profile.
type ProfileInput struct {
	FirstName     string   `json:"firstName"`
	Gender        string   `json:"gender"`
	Bio           string   `json:"bio"`
	City          string   `json:"city"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	ClearLocation bool     `json:"clearLocation"`
	Occupation    string   `json:"occupation"`
	HeightCM      *int     `json:"heightCm"`
	LookingFor    string   `json:"lookingFor"`
	Interests     []string `json:"interests"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
	Discoverable  bool     `json:"discoverable"`
	ShowDistance  bool     `json:"showDistance"`
}

var (
	Genders     = []string{"man", "woman", "non_binary"}
	LookingFors = []string{"long_term", "short_term", "friendship", "open", "unsure"}
)
