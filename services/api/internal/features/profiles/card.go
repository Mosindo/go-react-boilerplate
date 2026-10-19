package profiles

// CardColumns is the SQL column list (for a profiles row aliased "p") that
// ScanCard understands. Photos and interests are aggregated in the same query
// so listing N profiles never costs N+1 queries.
const CardColumns = `p.user_id, p.first_name, date_part('year', age(p.birth_date))::int, p.bio, COALESCE(p.city, ''),
	COALESCE((SELECT json_agg(json_build_object('id', ph.id, 'position', ph.position) ORDER BY ph.position)
		FROM photos ph WHERE ph.user_id = p.user_id), '[]'::json),
	COALESCE((SELECT json_agg(json_build_object('id', i.id, 'slug', i.slug, 'label', i.label) ORDER BY i.id)
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id WHERE ui.user_id = p.user_id), '[]'::json)`

// Scanner is satisfied by pgx.Row and pgx.Rows.
type Scanner interface {
	Scan(dest ...any) error
}

// ScanCard scans the CardColumns followed by any extra destinations.
func ScanCard(row Scanner, extra ...any) (PublicProfile, error) {
	return ScanCardWithPrefix(row, nil, extra...)
}

// ApproxDistanceKm turns an exact distance into a deliberately coarse one so
// the real position of a user cannot be triangulated: <=1 km collapses to 1,
// anything else is rounded up to the next multiple of 5 km.
func ApproxDistanceKm(exactKm float64) int {
	if exactKm <= 1 {
		return 1
	}
	return int((exactKm+4.999)/5) * 5
}

// SnapCoordinate rounds a coordinate to two decimals (~1.1 km) before storage.
func SnapCoordinate(v float64) float64 {
	if v < 0 {
		return -float64(int(-v*100+0.5)) / 100
	}
	return float64(int(v*100+0.5)) / 100
}

// ScanCardWithPrefix scans extra leading destinations followed by the card
// columns (for queries that SELECT other columns before CardColumns).
func ScanCardWithPrefix(row Scanner, prefix []any, extra ...any) (PublicProfile, error) {
	var p PublicProfile
	var raw cardJSON
	dest := append(append([]any{}, prefix...), &p.UserID, &p.FirstName, &p.Age, &p.Bio, &p.City, &raw.photos, &raw.interests)
	dest = append(dest, extra...)
	if err := row.Scan(dest...); err != nil {
		return PublicProfile{}, err
	}
	if err := raw.decode(&p); err != nil {
		return PublicProfile{}, err
	}
	return p, nil
}
