package discover

// rankingOrder is the only place that decides who is shown first. It is plain SQL on
// the columns exposed by the candidates CTE (liked_me, shared_interests, km,
// last_active_at). To plug in a smarter recommender later, replace this expression
// (or the whole Repository) without touching filtering, hydration or the API.
//
// Current policy, in order:
//  1. people who already liked you (a guaranteed match for you to take),
//  2. most shared interests,
//  3. closest,
//  4. most recently active.
const rankingOrder = `liked_me DESC, shared_interests DESC, km ASC, last_active_at DESC, user_id`
