package notifications

import "time"

type Notification struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"` // match | message
	ActorID   *string    `json:"actorId"`
	ActorName string     `json:"actorName"`
	RefID     *string    `json:"refId"` // match id ou conversation id
	ReadAt    *time.Time `json:"readAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

type MarkReadRequest struct {
	IDs []string `json:"ids" binding:"max=100"` // vide = tout marquer comme lu
}
