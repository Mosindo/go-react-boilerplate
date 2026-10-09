package profiles

import (
	"context"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
)

// DBTX is satisfied by *pgxpool.Pool and pgx.Tx so helpers work inside transactions.
type DBTX interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// SummarySelect lists the columns scanned by ScanSummary. It expects the
// profiles table aliased as p.
const SummarySelect = `p.user_id::text, p.first_name, date_part('year', age(p.birth_date))::int,
	(SELECT '/photos/' || ph.id::text || '/file' FROM photos ph WHERE ph.user_id = p.user_id AND ph.position = 0)`

func ScanSummary(row pgx.Row, s *ProfileSummary) error {
	return row.Scan(&s.ID, &s.FirstName, &s.Age, &s.MainPhotoURL)
}

// distanceBuckets are the only distances ever exposed, so that repeated
// look-ups from different spots cannot triangulate a precise position.
var distanceBuckets = []int{1, 2, 5, 10, 20, 50, 100, 200, 500}

// BucketDistance rounds a distance up to the next coarse bucket.
func BucketDistance(km float64) int {
	for _, b := range distanceBuckets {
		if km <= float64(b) {
			return b
		}
	}
	return 1000
}

// RoundCoordinate keeps ~1 km precision (0.01 degree), the most that is stored.
func RoundCoordinate(v float64) float64 {
	return math.Round(v*100) / 100
}

// AgeOn returns the age in full years of someone born on birth at the given instant.
func AgeOn(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// PublicSelect lists the columns scanned by ScanPublic. The query must alias the
// target profile as p and the viewer's profile as v (used for the distance).
const PublicSelect = `p.user_id::text, p.first_name, date_part('year', age(p.birth_date))::int, p.gender, p.bio, p.city,
	CASE WHEN p.show_distance THEN geo_distance_km(v.latitude, v.longitude, p.latitude, p.longitude) END`

// ScanPublic scans the columns produced by PublicSelect.
func ScanPublic(row pgx.Row, p *PublicProfile) error {
	var distance *float64
	var gender string
	if err := row.Scan(&p.ID, &p.FirstName, &p.Age, &gender, &p.Bio, &p.City, &distance); err != nil {
		return err
	}
	p.Gender = Gender(gender)
	p.DistanceKm = nil
	if distance != nil {
		b := BucketDistance(*distance)
		p.DistanceKm = &b
	}
	p.Interests = []Interest{}
	p.Photos = []Photo{}
	return nil
}

// Hydrate loads interests and photos for all profiles with two queries (no N+1).
func Hydrate(ctx context.Context, db DBTX, list []*PublicProfile) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, len(list))
	byID := make(map[string]*PublicProfile, len(list))
	for i, p := range list {
		ids[i] = p.ID
		byID[p.ID] = p
	}

	rows, err := db.Query(ctx, `
		SELECT ui.user_id::text, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY i.label
	`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var userID string
		var in Interest
		if err := rows.Scan(&userID, &in.Slug, &in.Label); err != nil {
			rows.Close()
			return err
		}
		byID[userID].Interests = append(byID[userID].Interests, in)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	rows, err = db.Query(ctx, `
		SELECT user_id::text, id::text, position
		FROM photos WHERE user_id = ANY($1::uuid[])
		ORDER BY user_id, position
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		var ph Photo
		if err := rows.Scan(&userID, &ph.ID, &ph.Position); err != nil {
			return err
		}
		ph.URL = PhotoURL(ph.ID)
		byID[userID].Photos = append(byID[userID].Photos, ph)
	}
	return rows.Err()
}
