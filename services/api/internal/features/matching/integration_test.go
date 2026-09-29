package matching

import (
	"context"
	"sync"
	"testing"

	"example.com/api/internal/platform/notify"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type recorder struct {
	mu     sync.Mutex
	notes  []notify.New
	events map[string][]realtime.Event
}

func newRecorder() *recorder { return &recorder{events: map[string][]realtime.Event{}} }

func (r *recorder) Notify(_ context.Context, n notify.New) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notes = append(r.notes, n)
	return nil
}

func (r *recorder) Publish(uid string, e realtime.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events[uid] = append(r.events[uid], e)
}

func (r *recorder) count(uid, typ string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events[uid] {
		if e.Type == typ {
			n++
		}
	}
	return n
}

func (r *recorder) notesFor(uid string) []notify.New {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []notify.New
	for _, n := range r.notes {
		if n.UserID == uid {
			out = append(out, n)
		}
	}
	return out
}

type person struct{ id, token string }

func seed(t *testing.T, pool *pgxpool.Pool, discoverable, photo, location bool) person {
	t.Helper()
	id, token := testutil.User(t, pool, "match", "1995-05-05")
	ctx := context.Background()
	var lat, lon any
	if location {
		lat, lon = 10.5, 20.5
	}
	if _, err := pool.Exec(ctx, `INSERT INTO profiles (user_id, first_name, gender, latitude, longitude, discoverable)
		VALUES ($1, 'Alex', 'woman', $2, $3, $4)`, id, lat, lon, discoverable); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO preferences (user_id, interested_in) VALUES ($1, ARRAY['woman','man','non_binary'])`, id); err != nil {
		t.Fatal(err)
	}
	if photo {
		if _, err := pool.Exec(ctx, `INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
			VALUES ($1, $2, 0, 10, 10, 10)`, id, "k-"+id); err != nil {
			t.Fatal(err)
		}
	}
	return person{id, token}
}

func std(t *testing.T, pool *pgxpool.Pool) person { return seed(t, pool, true, true, true) }

func setup(t *testing.T) (*pgxpool.Pool, *gin.Engine, *recorder) {
	pool := testutil.Pool(t)
	rec := newRecorder()
	d := testutil.Deps(t, rec, rec)
	r := testutil.Router(func(r gin.IRouter) { RegisterRoutes(r, pool, d) })
	return pool, r, rec
}

func swipe(t *testing.T, r *gin.Engine, who person, target, action string) testutil.Response {
	t.Helper()
	return testutil.Do(t, r, "POST", "/swipes", map[string]string{"userId": target, "action": action}, who.token)
}

func matchCount(t *testing.T, pool *pgxpool.Pool, a, b string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches
		WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, a, b).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestMutualLikeCreatesMatch(t *testing.T) {
	pool, r, rec := setup(t)
	a, b := std(t, pool), std(t, pool)

	res := swipe(t, r, a, b.id, "like")
	var first SwipeResponse
	res.JSON(t, &first)
	if res.Status != 200 || first.Matched || first.Conversation != nil {
		t.Fatalf("one-sided like: %d %s", res.Status, res.Body)
	}
	if matchCount(t, pool, a.id, b.id) != 0 {
		t.Fatal("no match expected yet")
	}

	res = swipe(t, r, b, a.id, "like")
	var second SwipeResponse
	res.JSON(t, &second)
	if res.Status != 200 || !second.Matched || second.Conversation == nil {
		t.Fatalf("mutual like: %d %s", res.Status, res.Body)
	}
	conv := second.Conversation
	if conv.User.UserID != a.id || conv.User.FirstName != "Alex" || conv.LastMessage != nil || conv.UnreadCount != 0 ||
		conv.User.Photo == nil || conv.User.Photo.URL == "" || !conv.UpdatedAt.Equal(conv.MatchedAt) {
		t.Fatalf("bad summary: %+v", conv)
	}
	if matchCount(t, pool, a.id, b.id) != 1 {
		t.Fatal("exactly one match expected")
	}
	var kind string
	var directKey *string
	var participants int
	if err := pool.QueryRow(context.Background(), `SELECT c.kind, c.direct_key, (SELECT COUNT(*) FROM conversation_participants WHERE conversation_id = c.id)
		FROM conversations c WHERE c.id = $1 AND c.match_id = $2`, conv.ID, conv.MatchID).Scan(&kind, &directKey, &participants); err != nil {
		t.Fatal(err)
	}
	if kind != "match" || directKey != nil || participants != 2 {
		t.Fatalf("conversation kind=%s key=%v participants=%d", kind, directKey, participants)
	}
	for _, p := range []person{a, b} {
		notes := rec.notesFor(p.id)
		if len(notes) != 1 || notes[0].Type != notify.TypeMatch || notes[0].Data["conversationId"] != conv.ID {
			t.Errorf("notifications for %s: %+v", p.id, notes)
		}
		if rec.count(p.id, "match.new") != 1 {
			t.Errorf("match.new not published to %s", p.id)
		}
	}

	// Duplicate swipe -> 409, also after matching.
	if res := swipe(t, r, a, b.id, "like"); res.Status != 409 {
		t.Errorf("duplicate status %d", res.Status)
	}
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	pool, r, rec := setup(t)
	for round := 0; round < 5; round++ {
		a, b := std(t, pool), std(t, pool)
		var wg sync.WaitGroup
		results := make([]SwipeResponse, 2)
		statuses := make([]int, 2)
		start := make(chan struct{})
		for i, pair := range [][2]person{{a, b}, {b, a}} {
			wg.Add(1)
			go func(i int, from, to person) {
				defer wg.Done()
				<-start
				res := swipe(t, r, from, to.id, "like")
				statuses[i] = res.Status
				res.JSON(t, &results[i])
			}(i, pair[0], pair[1])
		}
		close(start)
		wg.Wait()
		if statuses[0] != 200 || statuses[1] != 200 {
			t.Fatalf("statuses %v", statuses)
		}
		if matchCount(t, pool, a.id, b.id) != 1 {
			t.Fatalf("round %d: want exactly one match", round)
		}
		matched := 0
		for _, res := range results {
			if res.Matched {
				matched++
			}
		}
		if matched != 1 {
			t.Fatalf("exactly one of the two requests must report the match, got %d", matched)
		}
		var convs int
		_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM conversations c JOIN matches m ON m.id = c.match_id
			WHERE m.user_a = LEAST($1::uuid,$2::uuid) AND m.user_b = GREATEST($1::uuid,$2::uuid)`, a.id, b.id).Scan(&convs)
		if convs != 1 || len(rec.notesFor(a.id)) != 1 || len(rec.notesFor(b.id)) != 1 {
			t.Fatalf("convs=%d notes=%d/%d", convs, len(rec.notesFor(a.id)), len(rec.notesFor(b.id)))
		}
	}
}

func TestPassThenLikeNoMatch(t *testing.T) {
	pool, r, _ := setup(t)
	a, b := std(t, pool), std(t, pool)
	if res := swipe(t, r, a, b.id, "pass"); res.Status != 200 {
		t.Fatalf("pass: %d %s", res.Status, res.Body)
	}
	var out SwipeResponse
	res := swipe(t, r, b, a.id, "like")
	res.JSON(t, &out)
	if res.Status != 200 || out.Matched || matchCount(t, pool, a.id, b.id) != 0 {
		t.Fatalf("pass then like must not match: %d %s", res.Status, res.Body)
	}
	// Changing your mind is not allowed.
	if res := swipe(t, r, a, b.id, "like"); res.Status != 409 {
		t.Errorf("status %d", res.Status)
	}
}

func TestSwipeValidationAndEligibility(t *testing.T) {
	pool, r, _ := setup(t)
	a := std(t, pool)
	hidden := seed(t, pool, false, true, true)
	noPhoto := seed(t, pool, true, false, true)
	noLoc := seed(t, pool, true, true, false)
	blocker := std(t, pool)
	blocked := std(t, pool)
	if _, err := pool.Exec(context.Background(), `INSERT INTO blocks VALUES ($1, $2)`, blocker.id, a.id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO blocks VALUES ($1, $2)`, a.id, blocked.id); err != nil {
		t.Fatal(err)
	}

	if res := swipe(t, r, a, a.id, "like"); res.Status != 400 {
		t.Errorf("self swipe %d", res.Status)
	}
	if res := swipe(t, r, a, hidden.id, "bogus"); res.Status != 400 {
		t.Errorf("bad action %d", res.Status)
	}
	if res := swipe(t, r, a, "not-a-uuid", "like"); res.Status != 400 {
		t.Errorf("bad id %d", res.Status)
	}
	if res := testutil.Do(t, r, "POST", "/swipes", "junk", a.token); res.Status != 400 {
		t.Errorf("bad body %d", res.Status)
	}
	for name, target := range map[string]string{
		"hidden": hidden.id, "noPhoto": noPhoto.id, "noLoc": noLoc.id, "blockedMe": blocker.id, "iBlocked": blocked.id,
		"missing": "11111111-1111-4111-8111-111111111111",
	} {
		if res := swipe(t, r, a, target, "like"); res.Status != 404 {
			t.Errorf("%s: status %d", name, res.Status)
		}
	}
	// A hidden user who already liked me can be liked back and matches.
	if _, err := pool.Exec(context.Background(), `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, 'like')`, hidden.id, a.id); err != nil {
		t.Fatal(err)
	}
	var out SwipeResponse
	res := swipe(t, r, a, hidden.id, "like")
	res.JSON(t, &out)
	if res.Status != 200 || !out.Matched {
		t.Errorf("like back of hidden user: %d %s", res.Status, res.Body)
	}
	// Incomplete viewer gets 422.
	incomplete := seed(t, pool, true, false, true)
	if res := swipe(t, r, incomplete, a.id, "like"); res.Status != 422 {
		t.Errorf("incomplete viewer %d", res.Status)
	}
	if res := testutil.Do(t, r, "POST", "/swipes", map[string]string{"userId": a.id, "action": "like"}, ""); res.Status != 401 {
		t.Errorf("unauth %d", res.Status)
	}
}

func TestUnmatch(t *testing.T) {
	pool, r, rec := setup(t)
	a, b, c := std(t, pool), std(t, pool), std(t, pool)
	swipe(t, r, a, b.id, "like")
	var out SwipeResponse
	swipe(t, r, b, a.id, "like").JSON(t, &out)
	if !out.Matched {
		t.Fatal("expected match")
	}
	mid, cid := out.Conversation.MatchID, out.Conversation.ID
	if _, err := pool.Exec(context.Background(), `INSERT INTO messages (conversation_id, sender_user_id, content) VALUES ($1, $2, 'hi')`, cid, a.id); err != nil {
		t.Fatal(err)
	}

	if res := testutil.Do(t, r, "DELETE", "/matches/"+mid, nil, c.token); res.Status != 404 {
		t.Errorf("stranger unmatch %d", res.Status)
	}
	if res := testutil.Do(t, r, "DELETE", "/matches/garbage", nil, a.token); res.Status != 404 {
		t.Errorf("bad id %d", res.Status)
	}
	if matchCount(t, pool, a.id, b.id) != 1 {
		t.Fatal("stranger must not delete the match")
	}
	if res := testutil.Do(t, r, "DELETE", "/matches/"+mid, nil, a.token); res.Status != 204 {
		t.Fatalf("unmatch %d", res.Status)
	}
	var convs, msgs, swipes int
	_ = pool.QueryRow(context.Background(), `SELECT (SELECT COUNT(*) FROM conversations WHERE id = $1),
		(SELECT COUNT(*) FROM messages WHERE conversation_id = $1),
		(SELECT COUNT(*) FROM swipes WHERE (swiper_id = $2 AND target_id = $3) OR (swiper_id = $3 AND target_id = $2))`, cid, a.id, b.id).Scan(&convs, &msgs, &swipes)
	if convs != 0 || msgs != 0 || swipes != 2 {
		t.Errorf("convs=%d msgs=%d swipes=%d", convs, msgs, swipes)
	}
	if rec.count(a.id, "match.removed") != 1 || rec.count(b.id, "match.removed") != 1 {
		t.Error("match.removed must reach both users")
	}
	if res := testutil.Do(t, r, "DELETE", "/matches/"+mid, nil, a.token); res.Status != 404 {
		t.Errorf("second unmatch %d", res.Status)
	}
	// The pair cannot re-swipe.
	if res := swipe(t, r, a, b.id, "like"); res.Status != 409 {
		t.Errorf("re-swipe %d", res.Status)
	}
}

func TestPairKeySymmetric(t *testing.T) {
	if pairKey("a", "b") != pairKey("b", "a") {
		t.Fatal("pair key must not depend on order")
	}
}
