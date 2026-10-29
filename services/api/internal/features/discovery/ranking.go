package discovery

// rankExpression orders eligible candidates (lower is shown first). It is the
// single place to evolve the recommendation algorithm; every column it uses is
// selected by candidateQuery:
//
//	dist        distance in km between both users
//	max_dist    the smaller of both users' distance preferences
//	shared      number of interests in common
//	liked_me    1 when the candidate already liked the viewer
//	idle_days   days since the candidate was last active
//
// Candidates who already liked the viewer come first (more matches for everyone),
// then nearby profiles with common interests and recent activity. A little
// randomness avoids always showing the same order between refreshes.
const rankExpression = `
	  (CASE WHEN liked_me THEN -1.0 ELSE 0.0 END)
	+ (dist / GREATEST(max_dist, 1))
	- (0.15 * LEAST(shared, 4))
	+ (0.3 * LEAST(idle_days / 14.0, 1.0))
	+ (random() * 0.2)`
