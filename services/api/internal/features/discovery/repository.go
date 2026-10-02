package discovery

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepoIncomplete = errors.New("viewer profile incomplete")

type Repository interface {
	Candidates(ctx context.Context, viewerID string, limit int) ([]string, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

var distance = profiles.HaversineKm("me.lat", "me.lng", "c.latitude", "c.longitude")

// Candidates applies every hard rule server-side: orientation both ways, age both ways, distance
// both ways, hidden profiles, blocks in both directions, and already-processed profiles.
func (r *PGRepository) Candidates(ctx context.Context, viewerID string, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
		  SELECT p.user_id, p.gender, p.latitude AS lat, p.longitude AS lng,
		         EXTRACT(YEAR FROM age(p.birth_date))::int AS age,
		         pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
		  FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
		  WHERE p.user_id = $1 AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		)
		SELECT c.user_id::text
		FROM me
		JOIN profiles c ON c.user_id <> me.user_id
		JOIN preferences cp ON cp.user_id = c.user_id
		JOIN users u ON u.id = c.user_id
		WHERE c.discoverable
		  AND c.gender = ANY(me.interested_in)
		  AND me.gender = ANY(cp.interested_in)
		  AND EXTRACT(YEAR FROM age(c.birth_date)) BETWEEN me.min_age AND me.max_age
		  AND me.age BETWEEN cp.min_age AND cp.max_age
		  AND (me.max_distance_km IS NULL OR c.latitude BETWEEN me.lat - me.max_distance_km / 111.0 AND me.lat + me.max_distance_km / 111.0)
		  AND (me.max_distance_km IS NULL OR `+distance+` <= me.max_distance_km)
		  AND (cp.max_distance_km IS NULL OR `+distance+` <= cp.max_distance_km)
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = c.user_id)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
		           OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
		ORDER BY (`+scoreExpr+`) DESC, md5(me.user_id::text || c.user_id::text)
		LIMIT $2`, viewerID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// CanView: not blocked either way, and the target is visible, matched with the viewer, or already liked the viewer.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p
		  WHERE p.user_id = $2
		    AND (p.discoverable
		         OR EXISTS (SELECT 1 FROM matches m WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid))
		         OR EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $2 AND s.target_id = $1 AND s.action = 'like'))
		    AND NOT EXISTS (SELECT 1 FROM blocks b
		         WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)))`,
		viewerID, targetID).Scan(&ok)
	return ok, err
}
