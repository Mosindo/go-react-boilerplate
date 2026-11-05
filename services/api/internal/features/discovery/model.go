package discovery

// Candidate is the public view of a profile. It never contains coordinates,
// the birth date or the e-mail address.
type Candidate struct {
	ID              string     `json:"id"`
	FirstName       string     `json:"firstName"`
	Age             int        `json:"age"`
	Gender          string     `json:"gender"`
	Bio             string     `json:"bio"`
	City            string     `json:"city"`
	DistanceKm      *int       `json:"distanceKm"`
	Interests       []string   `json:"interests"`
	SharedInterests int        `json:"sharedInterests"`
	Photos          []PhotoRef `json:"photos"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
	Action string `json:"action" binding:"required,oneof=like pass"`
}

type SwipeResult struct {
	Matched        bool   `json:"matched"`
	AlreadySwiped  bool   `json:"alreadySwiped"`
	MatchID        string `json:"matchId,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
}

// MatchInfo describes a freshly created match for event fan-out.
type MatchInfo struct {
	MatchID        string
	ConversationID string
	UserA, UserB   string
	NameA, NameB   string
}

// Events lets other modules (notifications, realtime) react to match changes
// without discovery depending on them.
type Events interface {
	MatchCreated(info MatchInfo)
	MatchEnded(matchID, conversationID, byUserID, otherUserID string)
}
