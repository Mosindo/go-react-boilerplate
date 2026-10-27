// Package mailer abstracts outgoing email (account recovery). The default
// implementation only logs, so no SMTP credentials are needed to run locally.
// Production deployments plug a real implementation behind the same interface.
package mailer

import "context"

type Message struct {
	To      string
	Subject string
	Body    string
}

type Mailer interface {
	Send(ctx context.Context, msg Message) error
}
