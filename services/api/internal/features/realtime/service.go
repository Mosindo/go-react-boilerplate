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
func (s *Service) Redeem(ctx context.Context, token string) (platformrealtime.Ticket, error) {
	ticket, ok := s.tickets.Redeem(token)
	if !ok {
		return platformrealtime.Ticket{}, ErrInvalidTicket
	}
	active, err := s.sessions.SessionActive(ctx, ticket.SessionID, ticket.UserID)
	if err != nil {
		return platformrealtime.Ticket{}, err
	}
	if !active {
		return platformrealtime.Ticket{}, ErrInvalidTicket
	}
	return ticket, nil
}

func (s *Service) Serve(ticket platformrealtime.Ticket, conn *websocket.Conn) {
	s.hub.Serve(ticket.UserID, ticket.SessionID, conn)
}
