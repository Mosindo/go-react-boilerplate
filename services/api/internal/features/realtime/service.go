// Package realtime exposes the WebSocket entry point. The hub itself lives in
// platform/realtime so every feature can publish events.
package realtime

import (
	"context"
	"errors"

	platformrealtime "example.com/api/internal/platform/realtime"
	"github.com/gorilla/websocket"
)

var ErrInvalidTicket = errors.New("invalid or expired ticket")

type SessionChecker interface {
	SessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

type Service struct {
	hub      *platformrealtime.Hub
	tickets  *platformrealtime.TicketStore
	sessions SessionChecker
}

func NewService(hub *platformrealtime.Hub, tickets *platformrealtime.TicketStore, sessions SessionChecker) *Service {
	return &Service{hub: hub, tickets: tickets, sessions: sessions}
}

func (s *Service) IssueTicket(userID, sessionID string) (string, error) {
	return s.tickets.Issue(userID, sessionID)
}

// Redeem validates the ticket and that its session is still active.
func (s *Service) Redeem(ctx context.Context, token string) (string, error) {
	ticket, ok := s.tickets.Redeem(token)
	if !ok {
		return "", ErrInvalidTicket
	}
	active, err := s.sessions.SessionActive(ctx, ticket.SessionID, ticket.UserID)
	if err != nil {
		return "", err
	}
	if !active {
		return "", ErrInvalidTicket
	}
	return ticket.UserID, nil
}

func (s *Service) Serve(userID string, conn *websocket.Conn) {
	s.hub.Serve(userID, conn)
}
