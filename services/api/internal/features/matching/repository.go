package matching

import (
	"context"
	"errors"
	"strings"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MatchedPair struct {
	MatchID        string
	ConversationID string
	Self           MatchEvent // vu par le swiper: informations sur la cible
	Other          MatchEvent // vu par la cible: informations sur le swiper
}

type Repository interface {
	Feed(ctx context.Context, userID string, limit int) ([]Card, error)
	// Swipe enregistre la decision de maniere atomique. Si c'est un like reciproque,
	// le match, la conversation et les notifications sont crees dans la meme transaction.
	Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, *MatchedPair, error)
	ListMatches(ctx context.Context, userID string, limit int, cur *httpx.Cursor) ([]MatchSummary, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherID string, err error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

// eligibleFrom definit l'ensemble des profils qu'un utilisateur ($1) peut voir/swiper.
// Toutes les regles de compatibilite sont evaluees cote serveur et reciproquement:
// genre recherche, tranche d'age, distance maximale, decouvrabilite, blocages.
// C'est le point d'extension pour une future amelioration de la recommandation.
const eligibleFrom = `
	FROM profiles me
	JOIN preferences mp ON mp.user_id = me.user_id
	JOIN profiles t ON t.user_id <> me.user_id
	JOIN preferences tp ON tp.user_id = t.user_id
	CROSS JOIN LATERAL (
	  SELECT geo_distance_km(me.latitude, me.longitude, t.latitude, t.longitude) AS dist,
	         EXTRACT(YEAR FROM age(t.birth_date))::int AS t_age,
	         EXTRACT(YEAR FROM age(me.birth_date))::int AS me_age
	) x
	WHERE me.user_id = $1
	  AND tp.discoverable
	  AND t.gender = ANY (mp.interested_in)
	  AND me.gender = ANY (tp.interested_in)
	  AND x.t_age BETWEEN mp.min_age AND mp.max_age
	  AND x.me_age BETWEEN tp.min_age AND tp.max_age
	  AND t.latitude  BETWEEN me.latitude  - mp.max_distance_km / 111.0 AND me.latitude  + mp.max_distance_km / 111.0
	  AND x.dist <= LEAST(mp.max_distance_km, tp.max_distance_km)
	  AND NOT EXISTS (SELECT 1 FROM blocks b
	        WHERE (b.blocker_id = me.user_id AND b.blocked_id = t.user_id)
	           OR (b.blocker_id = t.user_id AND b.blocked_id = me.user_id))
`

// Classement: profils qui vous ont deja like (match quasi certain), puis centres
// d'interet partages, puis proximite, puis activite recente.
const feedQuery = `
	SELECT t.user_id, t.first_name, x.t_age, t.gender, t.bio, t.city, x.dist, tp.show_distance,
	       EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = t.user_id AND s.target_id = me.user_id AND s.action = 'like') AS liked_you,
	       (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
	         WHERE a.user_id = me.user_id AND b.user_id = t.user_id) AS shared,
	       COALESCE((SELECT array_agg(i.slug || '|' || i.label ORDER BY i.label)
	                 FROM user_interests ui JOIN interests i ON i.id = ui.interest_id WHERE ui.user_id = t.user_id), '{}') AS interests,
	       COALESCE((SELECT array_agg(ph.id::text ORDER BY ph.position) FROM photos ph WHERE ph.user_id = t.user_id), '{}') AS photo_ids
` + eligibleFrom + `
	  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = t.user_id)
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = t.user_id)
	ORDER BY liked_you DESC, shared DESC, x.dist ASC,
	         (SELECT last_active_at FROM users u WHERE u.id = t.user_id) DESC
	LIMIT $2`

func (r *PGRepository) Feed(ctx context.Context, userID string, limit int) ([]Card, error) {
	rows, err := r.pool.Query(ctx, feedQuery, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cards := []Card{}
	for rows.Next() {
		var c Card
		var shared int64
		var interests []string
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.Age, &c.Gender, &c.Bio, &c.City, &c.distanceExactKm, &c.showDistance,
			&c.LikedYou, &shared, &interests, &c.PhotoIDs); err != nil {
			return nil, err
		}
		c.SharedInterests = int(shared)
		c.Interests = make([]profiles.Interest, 0, len(interests))
		for _, s := range interests {
			if slug, label, ok := strings.Cut(s, "|"); ok {
				c.Interests = append(c.Interests, profiles.Interest{Slug: slug, Label: label})
			}
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

func (r *PGRepository) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, *MatchedPair, error) {
	res := SwipeResult{Action: action}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return res, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Verrou sur la paire: deux likes simultanes se serialisent et un seul match est cree.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext(LEAST($1::text,$2::text) || ':' || GREATEST($1::text,$2::text)))`,
		userID, targetID); err != nil {
		return res, nil, err
	}

	var hasProfile bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)`, userID).Scan(&hasProfile); err != nil {
		return res, nil, err
	}
	if !hasProfile {
		return res, nil, ErrProfileMissing
	}

	// Le serveur ne fait pas confiance au client: la cible doit etre eligible.
	var eligible bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 `+eligibleFrom+` AND t.user_id = $2)`, userID, targetID).Scan(&eligible); err != nil {
		return res, nil, err
	}
	if !eligible {
		return res, nil, ErrNotFound
	}

	tag, err := tx.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
		userID, targetID, action)
	if err != nil {
		return res, nil, err
	}
	if tag.RowsAffected() == 0 {
		return res, nil, ErrAlreadySwiped
	}
	if action != ActionLike {
		return res, nil, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id=$1 AND target_id=$2 AND action='like')`,
		targetID, userID).Scan(&reciprocal); err != nil {
		return res, nil, err
	}
	if !reciprocal {
		return res, nil, tx.Commit(ctx)
	}

	var matchID, convID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid,$2::uuid), GREATEST($1::uuid,$2::uuid))
		ON CONFLICT (user_a, user_b) DO UPDATE SET user_a = matches.user_a
		RETURNING id`, userID, targetID).Scan(&matchID); err != nil {
		return res, nil, err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (match_id) VALUES ($1)
		ON CONFLICT (match_id) DO UPDATE SET match_id = conversations.match_id
		RETURNING id`, matchID).Scan(&convID); err != nil {
		return res, nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1,$2),($1,$3)
		ON CONFLICT DO NOTHING`, convID, userID, targetID); err != nil {
		return res, nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, ref_id) VALUES ($1,'match',$2,$3),($2,'match',$1,$3)`,
		userID, targetID, matchID); err != nil {
		return res, nil, err
	}

	pair := &MatchedPair{MatchID: matchID, ConversationID: convID}
	if pair.Self, err = matchEvent(ctx, tx, matchID, convID, targetID); err != nil {
		return res, nil, err
	}
	if pair.Other, err = matchEvent(ctx, tx, matchID, convID, userID); err != nil {
		return res, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return res, nil, err
	}
	res.Matched, res.MatchID, res.ConversationID = true, matchID, convID
	return res, pair, nil
}

func matchEvent(ctx context.Context, tx pgx.Tx, matchID, convID, userID string) (MatchEvent, error) {
	ev := MatchEvent{MatchID: matchID, ConversationID: convID, UserID: userID}
	var photo *string
	err := tx.QueryRow(ctx, `
		SELECT p.first_name, (SELECT id::text FROM photos WHERE user_id = p.user_id ORDER BY position LIMIT 1)
		FROM profiles p WHERE p.user_id = $1`, userID).Scan(&ev.FirstName, &photo)
	if err != nil {
		return ev, err
	}
	if photo != nil {
		ev.PhotoID = *photo
	}
	return ev, nil
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit int, cur *httpx.Cursor) ([]MatchSummary, error) {
	var curAt *time.Time
	var curID *string
	if cur != nil {
		curAt, curID = &cur.At, &cur.ID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, c.id, m.created_at, p.user_id, p.first_name, EXTRACT(YEAR FROM age(p.birth_date))::int,
		       (SELECT id::text FROM photos WHERE user_id = p.user_id ORDER BY position LIMIT 1),
		       EXISTS (SELECT 1 FROM messages WHERE conversation_id = c.id)
		FROM matches m
		JOIN conversations c ON c.match_id = m.id
		JOIN profiles p ON p.user_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END
		WHERE (m.user_a = $1 OR m.user_b = $1)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id) OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
		  AND ($3::timestamptz IS NULL OR (m.created_at, m.id) < ($3::timestamptz, $4::uuid))
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $2`, userID, limit, curAt, curID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MatchSummary{}
	for rows.Next() {
		var m MatchSummary
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.CreatedAt, &m.UserID, &m.FirstName, &m.Age, &m.PhotoID, &m.HasMessages); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.pool.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $1 AND (user_a = $2 OR user_b = $2)
		RETURNING CASE WHEN user_a = $2 THEN user_b ELSE user_a END::text`, matchID, userID).Scan(&other)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return other, nil
}
