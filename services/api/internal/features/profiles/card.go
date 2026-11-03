package profiles

import (
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"
)

// DistanceKmExpr is the great-circle distance between viewer alias `v` and
// target alias `p`, in km. Both aliases must expose lat/lng.
const DistanceKmExpr = `(2 * 6371 * asin(LEAST(1, sqrt(
	power(sin(radians(p.lat - v.lat) / 2), 2) +
	cos(radians(v.lat)) * cos(radians(p.lat)) * power(sin(radians(p.lng - v.lng) / 2), 2)))))`

// CardColumns selects a public card for target alias `p` as seen by viewer
// alias `v`. The distance is bucketed up to a multiple of 5 km and hidden
// when the target opted out. Interests and photos are aggregated in-query
// so listing N cards costs one round trip.
const CardColumns = `p.user_id, p.first_name,
	date_part('year', age(p.birth_date))::int, p.gender, p.bio, p.city,
	CASE WHEN p.show_distance AND p.lat IS NOT NULL AND v.lat IS NOT NULL
		THEN GREATEST(5, CEIL(` + DistanceKmExpr + ` / 5.0) * 5)::int END,
	COALESCE((SELECT json_agg(json_build_object('slug', i.slug, 'label', i.label) ORDER BY i.label)
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = p.user_id), '[]'::json),
	COALESCE((SELECT json_agg(ph.id ORDER BY ph.position)
		FROM photos ph WHERE ph.user_id = p.user_id), '[]'::json)`

// ScanCard scans the CardColumns projection, followed by extra destinations.
func ScanCard(row pgx.Row, extra ...any) (Card, error) {
	var (
		card         Card
		interestsRaw []byte
		photosRaw    []byte
	)
	dest := append([]any{&card.ID, &card.FirstName, &card.Age, &card.Gender, &card.Bio, &card.City,
		&card.DistanceKm, &interestsRaw, &photosRaw}, extra...)
	if err := row.Scan(dest...); err != nil {
		return Card{}, err
	}
	card.Interests = []Interest{}
	if err := json.Unmarshal(interestsRaw, &card.Interests); err != nil {
		return Card{}, err
	}
	var photoIDs []string
	if err := json.Unmarshal(photosRaw, &photoIDs); err != nil {
		return Card{}, err
	}
	card.Photos = make([]PhotoRef, 0, len(photoIDs))
	for i, id := range photoIDs {
		card.Photos = append(card.Photos, PhotoRef{ID: id, URL: PhotoURL(id), Position: i})
	}
	return card, nil
}

func PhotoURL(id string) string { return "/photos/" + strings.TrimSpace(id) }

// VisibleToViewer is the SQL predicate (target alias `p`, viewer param $1)
// deciding whether a profile may be shown: no block in either direction and
// the profile is discoverable, matched with, or has liked the viewer.
const VisibleToViewer = `
	NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id) OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
	AND (p.discoverable
		OR EXISTS (SELECT 1 FROM matches m WHERE m.user_a = LEAST($1::uuid, p.user_id) AND m.user_b = GREATEST($1::uuid, p.user_id))
		OR EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = p.user_id AND s.target_id = $1 AND s.action = 'like'))`
