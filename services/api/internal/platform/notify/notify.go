// Package notify is the seam between features that raise notifications (matching, chat) and the
// notifications feature that persists and pushes them. Features depend on this interface only.
package notify

import "context"

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

type New struct {
	UserID string
	Type   string
	Title  string
	Body   string
	// Data carries routing hints for the client, e.g. {"conversationId": "...", "userId": "..."}.
	Data map[string]any
}

type Notifier interface {
	Notify(ctx context.Context, n New) error
}

type Nop struct{}

func (Nop) Notify(context.Context, New) error { return nil }
