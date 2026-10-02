package discovery

// scoreExpr orders eligible candidates. It is the single place to evolve the
// recommendation algorithm: it can reference the aliases `me` (viewer), `c`
// (candidate profile) and `u` (candidate user row).
//
// Current signals: shared interests, candidates who already liked the viewer,
// and recent activity. A per-pair hash breaks ties so that two viewers do not
// see the same ordering.
const scoreExpr = `
  2 * (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON a.interest_id = b.interest_id
        WHERE a.user_id = me.user_id AND b.user_id = c.user_id)
  + CASE WHEN EXISTS (SELECT 1 FROM swipes l WHERE l.swiper_id = c.user_id AND l.target_id = me.user_id AND l.action = 'like') THEN 3 ELSE 0 END
  + CASE WHEN u.last_active_at > NOW() - INTERVAL '7 days' THEN 1 ELSE 0 END`
