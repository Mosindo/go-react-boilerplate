package profiles

import "fmt"

// EligibleSQL is the single definition of "a profile that can be shown to and
// interacted with by others": complete, discoverable, with at least one photo.
// alias is the SQL alias of the profiles table.
func EligibleSQL(alias string) string {
	return fmt.Sprintf(`(%[1]s.discoverable
		AND %[1]s.first_name <> ''
		AND %[1]s.birthdate IS NOT NULL
		AND %[1]s.gender IS NOT NULL
		AND EXISTS (SELECT 1 FROM photos eph WHERE eph.user_id = %[1]s.user_id)
		AND EXISTS (SELECT 1 FROM preferences epf WHERE epf.user_id = %[1]s.user_id AND cardinality(epf.interested_in) > 0))`, alias)
}

// NotBlockedSQL is true when neither user has blocked the other. Arguments
// are SQL expressions (column references or placeholders).
func NotBlockedSQL(a, b string) string {
	return fmt.Sprintf(`NOT EXISTS (
		SELECT 1 FROM blocks nb
		WHERE (nb.blocker_id = %[1]s AND nb.blocked_id = %[2]s)
		   OR (nb.blocker_id = %[2]s AND nb.blocked_id = %[1]s))`, a, b)
}

// DistanceSQL computes the haversine distance in km between two profiles.
func DistanceSQL(a, b string) string {
	return fmt.Sprintf(`(2 * 6371 * asin(least(1, sqrt(
		power(sin(radians(%[2]s.latitude - %[1]s.latitude) / 2), 2) +
		cos(radians(%[1]s.latitude)) * cos(radians(%[2]s.latitude)) *
		power(sin(radians(%[2]s.longitude - %[1]s.longitude) / 2), 2)))))`, a, b)
}
