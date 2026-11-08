// Package discovery returns the next batch of profiles a user can like or pass.
//
// Eligibility is decided entirely in SQL on the server: orientation and age must match in
// both directions, both users' distance limits must be respected, blocked and already-swiped
// users never come back, and only complete, discoverable profiles are listed.
//
// Ranking is isolated in rankingOrder below so the recommendation algorithm can be replaced
// (e.g. by a scored materialised view or an external service) without touching the API.
package discovery

import (
	"context"
	"net/http"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// rankingOrder: people who already liked the viewer first (highest chance of a match),
// then nearest, then most recently active.
const rankingOrder = `ORDER BY liked_me DESC, dist ASC, u.last_active_at DESC, c.user_id`

const candidatesSQL = `
WITH me AS (
  SELECT p.user_id, p.gender, p.birth_date, p.latitude, p.longitude, pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km
  FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
  WHERE p.user_id = $1
), cand AS (
  SELECT c.user_id,
         geo_distance_km(me.latitude, me.longitude, c.latitude, c.longitude) AS dist,
         EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = c.user_id AND s.target_id = me.user_id AND s.action = 'like') AS liked_me,
         cp.max_distance_km AS their_max
  FROM me
  JOIN profiles c ON c.user_id <> me.user_id AND c.discoverable
       AND c.latitude BETWEEN me.latitude - me.max_distance_km / 111.0 AND me.latitude + me.max_distance_km / 111.0
  JOIN preferences cp ON cp.user_id = c.user_id
  WHERE c.gender = ANY(me.interested_in)
    AND me.gender = ANY(cp.interested_in)
    AND date_part('year', age(c.birth_date)) BETWEEN me.age_min AND me.age_max
    AND date_part('year', age(me.birth_date)) BETWEEN cp.age_min AND cp.age_max
    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
    AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = c.user_id)
    AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
                                              OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
)
SELECT c.user_id FROM cand c JOIN users u ON u.id = c.user_id
WHERE c.dist <= (SELECT max_distance_km FROM me) AND c.dist <= c.their_max
`

type Service struct {
	db    *pgxpool.Pool
	cards profiles.CardLoader
	gate  interface {
		IsComplete(ctx context.Context, userID string) (bool, error)
	}
}

func NewService(db *pgxpool.Pool, cards profiles.CardLoader, gate interface {
	IsComplete(ctx context.Context, userID string) (bool, error)
}) *Service {
	return &Service{db: db, cards: cards, gate: gate}
}

var ErrProfileIncomplete = platformerrors.New(http.StatusForbidden, "complete your profile (gender, preferences, location and at least one photo) to discover people")

func (s *Service) Next(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	ok, err := s.gate.IsComplete(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrProfileIncomplete
	}
	// Opening discovery counts as activity (used by the ranking).
	_, _ = s.db.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, userID)

	rows, err := s.db.Query(ctx, candidatesSQL+` `+rankingOrder+` LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return s.cards.LoadCards(ctx, userID, ids)
}

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) Next(c *gin.Context) {
	cards, err := h.service.Next(c.Request.Context(), c.GetString("userID"), httpx.Limit(c, 10, 20))
	if err != nil {
		httpx.Fail(c, "discovery.next", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"profiles": cards})
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/discover", requireUser, h.Next)
}
