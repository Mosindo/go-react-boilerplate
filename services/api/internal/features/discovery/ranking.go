package discovery

// rankingScoreSQL orders candidate profiles. It is the single place to evolve the
// recommendation algorithm: it can reference any column of the `cand` CTE
// (shared interests, distance, activity, "likes me") without touching the rest of the
// query. The random term keeps repeated fetches varied and avoids starving new profiles.
const rankingScoreSQL = `
	shared * 3.0
	+ CASE WHEN likes_me THEN 4.0 ELSE 0.0 END
	+ CASE WHEN last_active_at > NOW() - INTERVAL '7 days' THEN 2.0 ELSE 0.0 END
	+ COALESCE(3.0 * (1.0 - LEAST(dist, 100.0) / 100.0), 0.0)
	+ random() * 1.5`

// approximateDistance converts an exact kilometre distance to the coarse value shown
// to other users: 1 km granularity under 10 km, then 5 km steps. The exact figure never
// leaves the server.
func approximateDistance(km float64) int {
	if km < 1 {
		return 1
	}
	if km < 10 {
		return int(km + 0.999)
	}
	return int((km+2.5)/5) * 5
}
