// Package realtime exposes the WebSocket endpoint. Business events are
// produced by other features through platform/realtime.Publisher. This feature
// has no repository: it only authenticates and attaches connections.
package realtime

import (
	"example.com/api/internal/platform/tokens"
)

type Service struct {
	tokens *tokens.Manager
}

func NewService(tm *tokens.Manager) *Service {
	return &Service{tokens: tm}
}

// IssueTicket returns a 60-second token that authorizes one WebSocket upgrade.
// Tickets keep long-lived access tokens out of URLs.
func (s *Service) IssueTicket(userID, sessionID string) (string, error) {
	return s.tokens.SignRealtimeTicket(userID, sessionID)
}

func (s *Service) ParseTicket(raw string) (tokens.Claims, error) {
	return s.tokens.Parse(raw, tokens.TypeRealtime)
}
