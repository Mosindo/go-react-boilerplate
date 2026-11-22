package profiles

import "fmt"

// Shared SQL predicates. Other features (discovery, matching, chat) reuse them
// so visibility rules are defined in exactly one place.

// SQLProfileComplete is true when the profile aliased p (with preferences
// aliased pr) can be shown to other members.
func SQLProfileComplete(p, pr string) string {
	return fmt.Sprintf(`(%[1]s.first_name IS NOT NULL AND %[1]s.birthdate IS NOT NULL AND %[1]s.gender IS NOT NULL
		AND %[1]s.latitude IS NOT NULL AND cardinality(%[2]s.interested_in) > 0
		AND %[1]s.moderation_status = 'active'
		AND EXISTS (SELECT 1 FROM photos vis_ph WHERE vis_ph.user_id = %[1]s.user_id))`, p, pr)
}

// SQLBlockedBetween is true when either user blocked the other.
func SQLBlockedBetween(a, b string) string {
	return fmt.Sprintf(`EXISTS (SELECT 1 FROM blocks blk
		WHERE (blk.blocker_id = %[1]s AND blk.blocked_id = %[2]s)
		   OR (blk.blocker_id = %[2]s AND blk.blocked_id = %[1]s))`, a, b)
}

// SQLMatched is true when both users share a match.
func SQLMatched(a, b string) string {
	return fmt.Sprintf(`EXISTS (SELECT 1 FROM matches mt
		WHERE mt.user_a_id = LEAST(%[1]s, %[2]s) AND mt.user_b_id = GREATEST(%[1]s, %[2]s))`, a, b)
}
