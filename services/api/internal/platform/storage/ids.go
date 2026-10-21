package storage

// UserSummary is the minimal public identity shown in lists (matches, chats, notifications).
type UserSummary struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	ThumbURL  string `json:"thumbUrl"`
}
