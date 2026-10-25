package db

// Shared SQL fragments. They are plain constants/format helpers: queries stay explicit and readable.

// NotBlocked returns a predicate that is true when neither user blocked the other.
// a and b are SQL expressions (column names or $n placeholders) evaluating to user ids.
func NotBlocked(a, b string) string {
	return `NOT EXISTS (
		SELECT 1 FROM blocks bl
		WHERE (bl.blocker_id = ` + a + ` AND bl.blocked_id = ` + b + `)
		   OR (bl.blocker_id = ` + b + ` AND bl.blocked_id = ` + a + `)
	)`
}

// ProfileComplete returns a predicate that is true when the user has a profile, preferences and at least one photo.
func ProfileComplete(userID string) string {
	return `(EXISTS (SELECT 1 FROM profiles cp WHERE cp.user_id = ` + userID + `)
		AND EXISTS (SELECT 1 FROM preferences cpr WHERE cpr.user_id = ` + userID + `)
		AND EXISTS (SELECT 1 FROM photos cph WHERE cph.user_id = ` + userID + `))`
}

// DistanceKm returns a SQL expression computing the haversine distance in km between two lat/lng pairs
// (NULL when any coordinate is NULL).
func DistanceKm(aLat, aLng, bLat, bLng string) string {
	return `(6371 * 2 * asin(sqrt(LEAST(1.0,
		power(sin(radians(` + bLat + ` - ` + aLat + `) / 2), 2)
		+ cos(radians(` + aLat + `)) * cos(radians(` + bLat + `)) * power(sin(radians(` + bLng + ` - ` + aLng + `) / 2), 2)))))`
}

// ApproxDistance buckets a distance expression to a multiple of 5 km (minimum 5) so exact positions
// cannot be recovered by trilateration.
func ApproxDistance(distanceExpr string) string {
	return `GREATEST(5, CEIL(` + distanceExpr + ` / 5.0) * 5)::int`
}

// Age returns the SQL expression computing a full-years age from a birth_date expression.
func Age(birthDateExpr string) string {
	return `date_part('year', age(` + birthDateExpr + `))::int`
}
