// Package messaging implements chat inside a match. A match is the conversation:
// only its two members can read or write its messages.
package messaging

import "time"

type SendRequest struct {
	Body string `json:"body" binding:"required"`
}

type Message struct {
	ID        string    `json:"id"`
	MatchID   string    `json:"matchId"`
	SenderID  string    `json:"senderId"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type Page struct {
	// Items are ordered newest first.
	Items []Message `json:"items"`
	// NextBefore is the id to pass as ?before= to load older messages; empty when exhausted.
	NextBefore string `json:"nextBefore"`
	// OtherLastReadAt lets the client derive "read" state of its own messages.
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}
