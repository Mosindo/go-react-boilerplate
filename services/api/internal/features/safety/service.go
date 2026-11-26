package safety

import (
	"context"
	"errors"
	"log"
	"strings"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
)

var (
	ErrInvalidTarget = platformerrors.Validation("invalid_target", "Utilisateur invalide.")
	ErrUserNotFound  = platformerrors.NotFound("user_not_found", "Utilisateur introuvable.")
	ErrNotBlocked    = platformerrors.NotFound("block_not_found", "Cet utilisateur n'est pas bloqué.")
	ErrInvalidReason = platformerrors.Validation("invalid_reason", "Motif de signalement invalide.")
	ErrAlreadyReport = platformerrors.Conflict("already_reported", "Vous avez déjà signalé ce profil. Notre équipe l'examine.")
)

// MatchRemover breaks any match between two users (and with it the conversation).
type MatchRemover interface {
	RemoveMatchBetween(ctx context.Context, userID, otherID string) error
}

type Service struct {
	repo    Repository
	matches MatchRemover
}

func NewService(repo Repository, matches MatchRemover) *Service {
	return &Service{repo: repo, matches: matches}
}

func (s *Service) target(userID, raw string) (string, error) {
	target := strings.ToLower(strings.TrimSpace(raw))
	if !httpx.IsUUID(target) || target == userID {
		return "", ErrInvalidTarget
	}
	return target, nil
}

// Block hides both users from each other everywhere and removes any existing match.
func (s *Service) Block(ctx context.Context, userID, rawTarget string) error {
	target, err := s.target(userID, rawTarget)
	if err != nil {
		return err
	}
	if err := s.repo.Block(ctx, userID, target); err != nil {
		if errors.Is(err, errRepoUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return s.matches.RemoveMatchBetween(ctx, userID, target)
}

func (s *Service) Unblock(ctx context.Context, userID, rawTarget string) error {
	target, err := s.target(userID, rawTarget)
	if err != nil {
		return err
	}
	removed, err := s.repo.Unblock(ctx, userID, target)
	if err != nil {
		return err
	}
	if !removed {
		return ErrNotBlocked
	}
	return nil
}

func (s *Service) ListBlocked(ctx context.Context, userID string) ([]BlockedUser, error) {
	list, err := s.repo.ListBlocked(ctx, userID)
	if list == nil {
		list = []BlockedUser{}
	}
	return list, err
}

// Report records a report and always blocks the reported user for the reporter,
// so they never have to see that person again.
func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	target, err := s.target(userID, req.UserID)
	if err != nil {
		return err
	}
	reason := strings.TrimSpace(req.Reason)
	if _, ok := validReasons[reason]; !ok {
		return ErrInvalidReason
	}
	details, err := profiles.NormalizeText(req.Details, maxDetailsRunes, "invalid_details", "Le détail ne peut pas dépasser 1000 caractères.")
	if err != nil {
		return err
	}
	escalated, err := s.repo.Report(ctx, userID, target, reason, details)
	switch {
	case errors.Is(err, errRepoDuplicate):
		return ErrAlreadyReport
	case errors.Is(err, errRepoUserNotFound):
		return ErrUserNotFound
	case err != nil:
		return err
	}
	if escalated {
		log.Printf(`{"event":"moderation_escalated","reported_user_id":%q}`, target)
	}
	return s.Block(ctx, userID, target)
}
