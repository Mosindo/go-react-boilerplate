package safety

import (
	"context"
	"strings"
	"sync"
	"testing"

	"example.com/api/internal/features/discovery"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pub struct {
	mu     sync.Mutex
	events map[string][]realtime.Event
}

func (p *pub) Publish(uid string, e realtime.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.events == nil {
		p.events = map[string][]realtime.Event{}
	}
	p.events[uid] = append(p.events[uid], e)
}

func (p *pub) count(uid, typ string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := 0
	for _, e := range p.events[uid] {
		if e.Type == typ {
			n++
		}
	}
	return n
}

type person struct{ id, token string }

func seed(t *testing.T, pool *pgxpool.Pool) person {
	t.Helper()
	id, token := testutil.User(t, pool, "safe", "1995-05-05")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO profiles (user_id, first_name, gender, latitude, longitude)
		VALUES ($1, 'Sam', 'woman', -85.5, 170.5)`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO preferences (user_id, interested_in) VALUES ($1, ARRAY['woman','man','non_binary'])`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
		VALUES ($1, $2, 0, 10, 10, 10)`, id, "k-"+id); err != nil {
		t.Fatal(err)
	}
	return person{id, token}
}

func setup(t *testing.T) (*pgxpool.Pool, *gin.Engine, *pub) {
	pool := testutil.Pool(t)
	p := &pub{}
	d := testutil.Deps(t, nil, p)
	r := testutil.Router(func(r gin.IRouter) {
		RegisterRoutes(r, pool, d)
		discovery.RegisterRoutes(r, pool, d)
	})
	return pool, r, p
}

func makeMatch(t *testing.T, pool *pgxpool.Pool, a, b string) (matchID, convID string) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid,$2::uuid), GREATEST($1::uuid,$2::uuid)) RETURNING id`, a, b).Scan(&matchID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO conversations (kind, match_id) VALUES ('match', $1) RETURNING id`, matchID).Scan(&convID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1,$2),($1,$3)`, convID, a, b); err != nil {
		t.Fatal(err)
	}
	return
}

func TestBlockLifecycle(t *testing.T) {
	pool, r, p := setup(t)
	a, b, stranger := seed(t, pool), seed(t, pool), seed(t, pool)
	_, convID := makeMatch(t, pool, a.id, b.id)

	// visible in discovery before the block (stranger + b are unmatched? b is matched so hidden)
	if res := testutil.Do(t, r, "POST", "/blocks", map[string]string{"userId": a.id}, a.token); res.Status != 400 {
		t.Errorf("self block %d", res.Status)
	}
	if res := testutil.Do(t, r, "POST", "/blocks", map[string]string{"userId": "nope"}, a.token); res.Status != 400 {
		t.Errorf("bad id %d", res.Status)
	}
	if res := testutil.Do(t, r, "POST", "/blocks", map[string]string{"userId": "11111111-1111-4111-8111-111111111111"}, a.token); res.Status != 404 {
		t.Errorf("unknown user %d", res.Status)
	}
	for i := 0; i < 2; i++ { // idempotent
		if res := testutil.Do(t, r, "POST", "/blocks", map[string]string{"userId": b.id}, a.token); res.Status != 204 {
			t.Fatalf("block #%d: %d %s", i, res.Status, res.Body)
		}
	}
	var blocks, matches, convs int
	_ = pool.QueryRow(context.Background(), `SELECT (SELECT COUNT(*) FROM blocks WHERE blocker_id=$1 AND blocked_id=$2),
		(SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)),
		(SELECT COUNT(*) FROM conversations WHERE id=$3)`, a.id, b.id, convID).Scan(&blocks, &matches, &convs)
	if blocks != 1 || matches != 0 || convs != 0 {
		t.Errorf("blocks=%d matches=%d convs=%d", blocks, matches, convs)
	}
	if p.count(a.id, "match.removed") != 1 || p.count(b.id, "match.removed") != 1 {
		t.Error("match.removed must reach both users exactly once")
	}

	// Hidden from discovery in both directions, stranger stays visible.
	var disc struct {
		Profiles []struct {
			UserID string `json:"userId"`
		} `json:"profiles"`
	}
	for _, who := range []person{a, b} {
		other := b
		if who == b {
			other = a
		}
		testutil.Do(t, r, "GET", "/discover?limit=20", nil, who.token).JSON(t, &disc)
		seenStranger := false
		for _, pr := range disc.Profiles {
			if pr.UserID == other.id {
				t.Errorf("blocked pair visible in discovery")
			}
			if pr.UserID == stranger.id {
				seenStranger = true
			}
		}
		if !seenStranger {
			t.Errorf("stranger should remain visible")
		}
	}

	var list struct {
		Blocks []BlockedUser `json:"blocks"`
	}
	testutil.Do(t, r, "GET", "/blocks", nil, a.token).JSON(t, &list)
	if len(list.Blocks) != 1 || list.Blocks[0].UserID != b.id || list.Blocks[0].FirstName != "Sam" {
		t.Errorf("list: %+v", list)
	}
	testutil.Do(t, r, "GET", "/blocks", nil, b.token).JSON(t, &list)
	if len(list.Blocks) != 0 {
		t.Errorf("blocked user must not see the block: %+v", list)
	}

	if res := testutil.Do(t, r, "DELETE", "/blocks/"+a.id, nil, b.token); res.Status != 204 {
		t.Errorf("unblock by wrong side %d", res.Status)
	}
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM blocks WHERE blocker_id=$1`, a.id).Scan(&blocks)
	if blocks != 1 {
		t.Error("only the blocker may remove the block")
	}
	if res := testutil.Do(t, r, "DELETE", "/blocks/"+b.id, nil, a.token); res.Status != 204 {
		t.Errorf("unblock %d", res.Status)
	}
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM blocks WHERE blocker_id=$1`, a.id).Scan(&blocks)
	if blocks != 0 {
		t.Error("block should be gone")
	}
	if res := testutil.Do(t, r, "GET", "/blocks", nil, ""); res.Status != 401 {
		t.Errorf("unauth %d", res.Status)
	}
}

func TestReports(t *testing.T) {
	pool, r, _ := setup(t)
	a, b := seed(t, pool), seed(t, pool)
	post := func(body map[string]any) testutil.Response {
		return testutil.Do(t, r, "POST", "/reports", body, a.token)
	}
	if res := post(map[string]any{"userId": b.id, "reason": "nonsense"}); res.Status != 400 {
		t.Errorf("bad reason %d", res.Status)
	}
	if res := post(map[string]any{"userId": b.id, "reason": "spam", "details": strings.Repeat("x", 1001)}); res.Status != 400 {
		t.Errorf("long details %d", res.Status)
	}
	if res := post(map[string]any{"userId": a.id, "reason": "spam"}); res.Status != 400 {
		t.Errorf("self report %d", res.Status)
	}
	if res := post(map[string]any{"userId": "11111111-1111-4111-8111-111111111111", "reason": "spam"}); res.Status != 404 {
		t.Errorf("unknown %d", res.Status)
	}
	if res := post(map[string]any{"reason": "spam"}); res.Status != 400 {
		t.Errorf("missing user %d", res.Status)
	}

	res := post(map[string]any{"userId": b.id, "reason": "scam", "details": "  bad\x00 \x07actor \n"})
	var first struct{ ID string }
	res.JSON(t, &first)
	if res.Status != 201 || first.ID == "" {
		t.Fatalf("create %d %s", res.Status, res.Body)
	}
	var details, status string
	_ = pool.QueryRow(context.Background(), `SELECT details, status FROM reports WHERE id=$1`, first.ID).Scan(&details, &status)
	if details != "bad actor" || status != "open" {
		t.Errorf("stored %q %q", details, status)
	}
	// Second report within 24h returns the same id and inserts nothing.
	var second struct{ ID string }
	res = post(map[string]any{"userId": b.id, "reason": "spam"})
	res.JSON(t, &second)
	var n int
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id=$1`, a.id).Scan(&n)
	if res.Status != 201 || second.ID != first.ID || n != 1 {
		t.Errorf("dedup: %d id=%s n=%d", res.Status, second.ID, n)
	}
	// Older than 24h: a new report is accepted.
	_, _ = pool.Exec(context.Background(), `UPDATE reports SET created_at = NOW() - INTERVAL '25 hours' WHERE id=$1`, first.ID)
	var third struct{ ID string }
	post(map[string]any{"userId": b.id, "reason": "spam"}).JSON(t, &third)
	if third.ID == first.ID {
		t.Error("expected a fresh report after 24h")
	}

	// block:true blocks and removes the match.
	c := seed(t, pool)
	makeMatch(t, pool, a.id, c.id)
	if res := post(map[string]any{"userId": c.id, "reason": "harassment", "block": true}); res.Status != 201 {
		t.Fatalf("report+block %d", res.Status)
	}
	var blocked, matches int
	_ = pool.QueryRow(context.Background(), `SELECT (SELECT COUNT(*) FROM blocks WHERE blocker_id=$1 AND blocked_id=$2),
		(SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid))`, a.id, c.id).Scan(&blocked, &matches)
	if blocked != 1 || matches != 0 {
		t.Errorf("blocked=%d matches=%d", blocked, matches)
	}
}

func TestSanitizeDetails(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"  hello  ", "hello", true},
		{"a\x00b\x1bc", "abc", true},
		{"line1\nline2\ttab", "line1\nline2\ttab", true},
		{"", "", true},
		{strings.Repeat("é", 1000), strings.Repeat("é", 1000), true},
		{strings.Repeat("é", 1001), strings.Repeat("é", 1001), false},
		{"  " + strings.Repeat("x", 1000) + "  ", strings.Repeat("x", 1000), true},
	}
	for _, c := range cases {
		got, ok := SanitizeDetails(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("SanitizeDetails(%q) = %q,%v want %q,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestPairKeySymmetric(t *testing.T) {
	if pairKey("a", "b") != pairKey("b", "a") || pairKey("a", "b") != "pair:a:b" {
		t.Fatal("pair key must be order independent")
	}
}
