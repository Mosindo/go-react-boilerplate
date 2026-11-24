// Package realtime exposes the WebSocket endpoint. Clients first obtain a
// one-minute ticket with their access token (POST /realtime/ticket), then
// connect to /realtime?ticket=... — so access tokens never appear in URLs.
package realtime

import (
	"context"
	"time"

	apperr "example.com/api/internal/platform/errors"
	platformrealtime "example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/tokens"
)

const TicketTTL = time.Minute

var ErrInvalidTicket = apperr.Unauthorized("invalid_ticket", "invalid or expired realtime ticket")

type SessionChecker interface {
	IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error)
}

type Service struct {
	tokens   *tokens.Manager
	sessions SessionChecker
	hub      *platformrealtime.Hub
}

func NewService(manager *tokens.Manager, sessions SessionChecker, hub *platformrealtime.Hub) *Service {
	return &Service{tokens: manager, sessions: sessions, hub: hub}
}

func (s *Service) IssueTicket(userID, sessionID string) (TicketResponse, error) {
	ticket, err := s.tokens.Sign(userID, sessionID, tokens.AudienceRealtime, TicketTTL)
	if err != nil {
		return TicketResponse{}, err
	}
	return TicketResponse{Ticket: ticket, ExpiresIn: int(TicketTTL.Seconds())}, nil
}

func (s *Service) Authenticate(ctx context.Context, ticket string) (tokens.Claims, error) {
	claims, err := s.tokens.Parse(ticket, tokens.AudienceRealtime)
	if err != nil {
		return tokens.Claims{}, ErrInvalidTicket
	}
	active, err := s.SessionActive(ctx, claims.UserID, claims.SessionID)
	if err != nil {
		return tokens.Claims{}, err
	}
	if !active {
		return tokens.Claims{}, ErrInvalidTicket
	}
	return claims, nil
}

// SessionActive lets long-lived sockets re-check their session.
func (s *Service) SessionActive(ctx context.Context, userID, sessionID string) (bool, error) {
	return s.sessions.IsSessionActive(ctx, userID, sessionID)
}

func (s *Service) Hub() *platformrealtime.Hub {
	return s.hub
}
