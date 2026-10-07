package discovery

import (
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/media"
)

type PublicProfile struct {
	UserID          string              `json:"userId"`
	FirstName       string              `json:"firstName"`
	Age             int                 `json:"age"`
	Gender          string              `json:"gender"`
	Bio             string              `json:"bio"`
	City            string              `json:"city"`
	DistanceKm      *int                `json:"distanceKm"`
	Interests       []profiles.Interest `json:"interests"`
	SharedInterests []string            `json:"sharedInterests"`
	Photos          []media.Photo       `json:"photos"`
}

type Match struct {
	ID             string               `json:"id"`
	User           profiles.UserSummary `json:"user"`
	ConversationID string               `json:"conversationId"`
	CreatedAt      time.Time            `json:"createdAt"`
}

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required"`
	Action       string `json:"action" binding:"required"`
}

type SwipeResponse struct {
	Matched bool   `json:"matched"`
	Match   *Match `json:"match"`
}
