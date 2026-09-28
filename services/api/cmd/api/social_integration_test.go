package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	parisLat, parisLon = 48.86, 2.35
)

func TestDiscoveryFilters(t *testing.T) {
	e := newEnv(t)

	viewer := e.person(spec{Name: "Viewer", Gender: "man", Age: 30, Lat: parisLat, Lon: parisLon,
		Prefs: map[string]any{"interestedIn": []string{"woman"}, "minAge": 25, "maxAge": 35, "maxDistanceKm": 50}})

	ok := e.person(spec{Name: "Ok", Gender: "woman", Age: 28, Lat: parisLat, Lon: parisLon + 0.05})
	wrongGender := e.person(spec{Name: "WrongGender", Gender: "man", Age: 28, Lat: parisLat, Lon: parisLon})
	tooOld := e.person(spec{Name: "TooOld", Gender: "woman", Age: 45, Lat: parisLat, Lon: parisLon})
	tooYoung := e.person(spec{Name: "TooYoung", Gender: "woman", Age: 22, Lat: parisLat, Lon: parisLon})
	far := e.person(spec{Name: "Far", Gender: "woman", Age: 28, Lat: 45.76, Lon: 4.84}) // Lyon
	recipGender := e.person(spec{Name: "RecipGender", Gender: "woman", Age: 28, Lat: parisLat, Lon: parisLon,
		Prefs: map[string]any{"interestedIn": []string{"woman"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50}})
	recipAge := e.person(spec{Name: "RecipAge", Gender: "woman", Age: 28, Lat: parisLat, Lon: parisLon,
		Prefs: map[string]any{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 25, "maxDistanceKm": 50}})
	recipDist := e.person(spec{Name: "RecipDist", Gender: "woman", Age: 28, Lat: parisLat, Lon: parisLon + 0.30, // ~22 km
		Prefs: map[string]any{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 10}})
	mid := e.person(spec{Name: "Mid", Gender: "woman", Age: 28, Lat: parisLat, Lon: parisLon + 0.30}) // ~22 km, within both
	hidden := e.person(spec{Name: "Hidden", Gender: "woman", Age: 28, Lat: parisLat, Lon: parisLon, Discoverable: boolPtr(false)})
	noPhoto := e.person(spec{Name: "NoPhoto", Gender: "woman", Age: 28, Lat: parisLat, Lon: parisLon, NoPhoto: true})
	noLoc := e.person(spec{Name: "NoLoc", Gender: "woman", Age: 28, NoLocation: true})
	blockedByViewer := e.person(spec{Name: "BlockedByViewer", Gender: "woman", Age: 28, Lat: parisLat, Lon: parisLon})
	blocksViewer := e.person(spec{Name: "BlocksViewer", Gender: "woman", Age: 28, Lat: parisLat, Lon: parisLon})
	swiped := e.person(spec{Name: "Swiped", Gender: "woman", Age: 28, Lat: parisLat, Lon: parisLon})
	shy := e.person(spec{Name: "Shy", Gender: "woman", Age: 29, Lat: parisLat, Lon: parisLon})
	e.must(shy.call("PUT", "/me/profile", map[string]any{"firstName": "Shy", "birthDate": birthFor(29), "gender": "woman", "bio": "", "interestIds": []int{}, "showDistance": false}), 200)

	e.must(viewer.call("POST", "/blocks", map[string]string{"userId": blockedByViewer.ID}), 204)
	e.must(blocksViewer.call("POST", "/blocks", map[string]string{"userId": viewer.ID}), 204)
	e.must(viewer.swipe(swiped, "pass"), 200)

	got := discoverIDs(t, viewer, "")
	want := []*user{ok, mid, shy}
	wantIDs := make([]string, len(want))
	for i, u := range want {
		wantIDs[i] = u.ID
	}
	sort.Strings(got)
	sort.Strings(wantIDs)
	if strings.Join(got, ",") != strings.Join(wantIDs, ",") {
		names := map[string]string{}
		for _, u := range []*user{ok, wrongGender, tooOld, tooYoung, far, recipGender, recipAge, recipDist, mid, hidden, noPhoto, noLoc, blockedByViewer, blocksViewer, swiped, shy} {
			names[u.ID] = u.Name
		}
		var gn []string
		for _, id := range got {
			gn = append(gn, names[id])
		}
		t.Fatalf("discover mismatch: got %v want [Ok Mid Shy]", gn)
	}

	// privacy of the candidate payload
	res := e.must(viewer.call("GET", "/discover?limit=10", nil), 200)
	for _, raw := range res.json()["items"].([]any) {
		c := raw.(map[string]any)
		for _, forbidden := range []string{"birthDate", "latitude", "longitude", "lat", "lon", "email"} {
			if _, has := c[forbidden]; has {
				t.Errorf("candidate leaks %q: %v", forbidden, c)
			}
		}
		if c["userId"] == shy.ID {
			if c["distanceKm"] != nil {
				t.Errorf("showDistance=false must hide distance: %v", c)
			}
			continue
		}
		d, isNum := c["distanceKm"].(float64)
		if !isNum || d < 5 || int(d)%5 != 0 {
			t.Errorf("distance must be bucketed to 5 km steps, got %v", c["distanceKm"])
		}
		if c["userId"] == mid.ID && d != 25 {
			t.Errorf("~22 km should bucket to 25, got %v", d)
		}
		if c["userId"] == ok.ID && d != 5 {
			t.Errorf("~3.7 km should bucket to 5, got %v", d)
		}
		if len(c["photos"].([]any)) != 1 {
			t.Errorf("candidate photos: %v", c["photos"])
		}
	}

	// limit validation
	for _, q := range []string{"limit=0", "limit=51", "limit=abc"} {
		if r := viewer.call("GET", "/discover?"+q, nil); r.Status != 400 {
			t.Errorf("discover?%s: %d", q, r.Status)
		}
	}
	if got := discoverIDs(t, viewer, "&limit=1"); len(got) != 1 {
		// the helper prepends limit=50; last one wins in Go's Query()? (first wins) so check directly
		r := e.must(viewer.call("GET", "/discover?limit=1", nil), 200).json()
		if len(r["items"].([]any)) != 1 {
			t.Errorf("limit=1 must return one item")
		}
	}

	// swiped profiles disappear
	e.must(viewer.swipe(ok, "like"), 200)
	if contains(discoverIDs(t, viewer, ""), ok.ID) {
		t.Error("liked profile must leave the queue")
	}

	// incomplete viewer
	bare := e.register()
	if r := bare.call("GET", "/discover", nil); r.Status != 409 || r.code() != "profile_incomplete" {
		t.Errorf("incomplete viewer: %d %s", r.Status, r.Body)
	}
	if r := bare.swipe(ok, "like"); r.Status != 409 || r.code() != "profile_incomplete" {
		t.Errorf("incomplete swiper: %d %s", r.Status, r.Body)
	}
}

func boolPtr(b bool) *bool { return &b }

func TestDiscoveryRankingPrefersSharedInterests(t *testing.T) {
	e := newEnv(t)
	viewer := e.person(spec{Gender: "man", Age: 30, Lat: 10, Lon: 10, Interests: []int{1, 2, 3}})
	plain := e.person(spec{Name: "Plain", Gender: "woman", Lat: 10, Lon: 10})
	shared := e.person(spec{Name: "Shared", Gender: "woman", Lat: 10, Lon: 10, Interests: []int{1, 2}})
	// make "plain" the most recently active so only the ranker can put "shared" first
	if _, err := e.pool.Exec(context.Background(), `UPDATE profiles SET last_active_at = NOW() WHERE user_id=$1`, plain.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(context.Background(), `UPDATE profiles SET last_active_at = NOW() - INTERVAL '3 hours' WHERE user_id=$1`, shared.ID); err != nil {
		t.Fatal(err)
	}
	items := e.must(viewer.call("GET", "/discover", nil), 200).json()["items"].([]any)
	if len(items) != 2 || items[0].(map[string]any)["userId"] != shared.ID {
		t.Fatalf("shared-interest candidate should rank first: %v", items)
	}
	if items[0].(map[string]any)["sharedInterestCount"] != float64(2) {
		t.Errorf("sharedInterestCount: %v", items[0])
	}
}

func TestUserProfileEndpoint(t *testing.T) {
	e := newEnv(t)
	v := e.person(spec{Gender: "man", Lat: 10, Lon: 10})
	vis := e.person(spec{Name: "Vis", Gender: "woman", Lat: 10, Lon: 10.2})
	hid := e.person(spec{Name: "Hid", Gender: "woman", Lat: 10, Lon: 10, Discoverable: boolPtr(false)})
	blk := e.person(spec{Name: "Blk", Gender: "woman", Lat: 10, Lon: 10})

	r := e.must(v.call("GET", "/users/"+vis.ID+"/profile", nil), 200).json()
	if r["firstName"] != "Vis" || r["distanceKm"] == nil {
		t.Fatalf("profile: %v", r)
	}
	for _, f := range []string{"birthDate", "latitude", "longitude", "email"} {
		if _, has := r[f]; has {
			t.Errorf("leaks %s", f)
		}
	}
	if r := v.call("GET", "/users/"+hid.ID+"/profile", nil); r.Status != 404 {
		t.Errorf("hidden: %d", r.Status)
	}
	e.matchUsersHidden(v, hid)
	e.must(v.call("GET", "/users/"+hid.ID+"/profile", nil), 200)
	e.must(v.call("POST", "/blocks", map[string]string{"userId": blk.ID}), 204)
	if r := v.call("GET", "/users/"+blk.ID+"/profile", nil); r.Status != 404 {
		t.Errorf("blocked: %d", r.Status)
	}
	if r := blk.call("GET", "/users/"+v.ID+"/profile", nil); r.Status != 404 {
		t.Errorf("blocked (reverse): %d", r.Status)
	}
	if r := v.call("GET", "/users/"+v.ID+"/profile", nil); r.Status != 404 {
		t.Errorf("self: %d", r.Status)
	}
	if r := v.call("GET", "/users/not-a-uuid/profile", nil); r.Status != 404 {
		t.Errorf("bad id: %d", r.Status)
	}
	if r := v.call("GET", "/users/00000000-0000-0000-0000-000000000000/profile", nil); r.Status != 404 {
		t.Errorf("unknown: %d", r.Status)
	}
}

func TestSwipeValidationAndMatchFlow(t *testing.T) {
	e := newEnv(t)
	a := e.person(spec{Name: "Alice", Gender: "woman", Lat: 10, Lon: 10})
	b := e.person(spec{Name: "Bob", Gender: "man", Lat: 10, Lon: 10})
	hidden := e.person(spec{Gender: "man", Lat: 10, Lon: 10, Discoverable: boolPtr(false)})
	blocked := e.person(spec{Gender: "man", Lat: 10, Lon: 10})
	e.must(a.call("POST", "/blocks", map[string]string{"userId": blocked.ID}), 204)

	for name, body := range map[string]map[string]string{
		"bad action":   {"targetUserId": b.ID, "action": "superlike"},
		"empty action": {"targetUserId": b.ID, "action": ""},
		"bad id":       {"targetUserId": "x", "action": "like"},
		"self":         {"targetUserId": a.ID, "action": "like"},
	} {
		if r := a.call("POST", "/swipes", body); r.Status != 400 || r.code() != "invalid_request" {
			t.Errorf("%s: %d %s", name, r.Status, r.Body)
		}
	}
	for name, id := range map[string]string{
		"unknown": "00000000-0000-0000-0000-000000000000", "hidden": hidden.ID, "blocked": blocked.ID,
	} {
		if r := a.call("POST", "/swipes", map[string]string{"targetUserId": id, "action": "like"}); r.Status != 404 {
			t.Errorf("%s: want 404 got %d %s", name, r.Status, r.Body)
		}
	}
	if r := a.call("DELETE", "/swipes/last", nil); r.Status != 501 {
		t.Errorf("undo must not be implemented: %d", r.Status)
	}

	r := e.must(a.swipe(b, "like"), 200).json()
	if r["matched"] != false || r["match"] != nil {
		t.Fatalf("first like: %v", r)
	}
	// repeated swipe: idempotent, and a different action does not overwrite
	e.must(a.swipe(b, "like"), 200)
	e.must(a.swipe(b, "pass"), 200)
	if n := count(t, e.pool, `SELECT count(*) FROM swipes WHERE from_user_id=$1`, a.ID); n != 1 {
		t.Fatalf("swipe must be stored once, got %d", n)
	}

	r = e.must(b.swipe(a, "like"), 200).json()
	if r["matched"] != true {
		t.Fatalf("reciprocal like must match: %v", r)
	}
	m := r["match"].(map[string]any)
	if m["matchId"] == nil || m["conversationId"] == nil || m["unreadCount"] != float64(0) || m["lastMessage"] != nil {
		t.Fatalf("match summary: %v", m)
	}
	mu := m["user"].(map[string]any)
	if mu["userId"] != a.ID || mu["firstName"] != "Alice" || mu["age"] != float64(30) || mu["photo"] == nil {
		t.Fatalf("match.user must describe the other person: %v", mu)
	}
	// repeated swipe after the match returns the stored result and does not duplicate anything
	again := e.must(b.swipe(a, "like"), 200).json()
	if again["matched"] != true || again["match"].(map[string]any)["matchId"] != m["matchId"] {
		t.Fatalf("repeat after match: %v", again)
	}
	back := e.must(a.swipe(b, "like"), 200).json()
	if back["matched"] != true {
		t.Fatalf("original swiper sees the match on repeat: %v", back)
	}
	if n := count(t, e.pool, `SELECT count(*) FROM matches`); n != 1 {
		t.Fatalf("matches: %d", n)
	}
	if n := count(t, e.pool, `SELECT count(*) FROM match_conversations`); n != 1 {
		t.Fatalf("conversations: %d", n)
	}
	// stored ordered
	if n := count(t, e.pool, `SELECT count(*) FROM matches WHERE user_a < user_b`); n != 1 {
		t.Fatalf("pair must be stored ordered")
	}
	// exactly one match notification per user
	for _, u := range []*user{a, b} {
		if n := count(t, e.pool, `SELECT count(*) FROM notifications WHERE user_id=$1 AND type='match'`, u.ID); n != 1 {
			t.Fatalf("match notifications for %s: %d", u.Name, n)
		}
	}

	// a pass never matches
	c := e.person(spec{Gender: "man", Lat: 10, Lon: 10})
	e.must(c.swipe(a, "like"), 200)
	r = e.must(a.swipe(c, "pass"), 200).json()
	if r["matched"] != false {
		t.Fatalf("pass must not match: %v", r)
	}
	// pass then like by the other side never matches either
	d := e.person(spec{Gender: "man", Lat: 10, Lon: 10})
	e.must(d.swipe(a, "pass"), 200)
	if r := e.must(a.swipe(d, "like"), 200).json(); r["matched"] != false {
		t.Fatalf("like after pass must not match: %v", r)
	}
}

func TestConcurrentReciprocalLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	const pairs = 12
	type pair struct{ a, b *user }
	var ps []pair
	for i := 0; i < pairs; i++ {
		ps = append(ps, pair{
			e.person(spec{Gender: "woman", Lat: float64(i), Lon: 100}),
			e.person(spec{Gender: "man", Lat: float64(i), Lon: 100}),
		})
	}
	var wg sync.WaitGroup
	matched := make([][2]bool, pairs)
	start := make(chan struct{})
	for i, p := range ps {
		for side, pair := range [][2]*user{{p.a, p.b}, {p.b, p.a}} {
			wg.Add(1)
			go func(i, side int, from, to *user) {
				defer wg.Done()
				<-start
				r := from.swipe(to, "like")
				if r.Status != 200 {
					t.Errorf("swipe status %d %s", r.Status, r.Body)
					return
				}
				matched[i][side] = r.json()["matched"] == true
			}(i, side, pair[0], pair[1])
		}
	}
	close(start)
	wg.Wait()

	for i, p := range ps {
		n := count(t, e.pool, `SELECT count(*) FROM matches WHERE user_a=LEAST($1::uuid,$2::uuid) AND user_b=GREATEST($1::uuid,$2::uuid)`, p.a.ID, p.b.ID)
		if n != 1 {
			t.Errorf("pair %d: %d matches", i, n)
		}
		if !matched[i][0] && !matched[i][1] {
			t.Errorf("pair %d: nobody was told about the match", i)
		}
		if n := count(t, e.pool, `SELECT count(*) FROM notifications WHERE type='match' AND user_id IN ($1,$2)`, p.a.ID, p.b.ID); n != 2 {
			t.Errorf("pair %d: %d match notifications, want 2", i, n)
		}
	}
	if n := count(t, e.pool, `SELECT count(*) FROM match_conversations`); n != pairs {
		t.Errorf("conversations: %d want %d", n, pairs)
	}
}

func TestMatchesListPaginationAndUnmatch(t *testing.T) {
	e := newEnv(t)
	me := e.person(spec{Name: "Me", Gender: "woman", Lat: 10, Lon: 10})
	var others []*user
	convs := map[string]string{}
	matchIDs := map[string]string{}
	for i := 0; i < 5; i++ {
		o := e.person(spec{Name: fmt.Sprintf("M%d", i), Gender: "man", Lat: 10, Lon: 10})
		mid, cid := e.matchUsers(me, o)
		others = append(others, o)
		convs[o.ID], matchIDs[o.ID] = cid, mid
		time.Sleep(5 * time.Millisecond)
	}
	// activity: message in the oldest match pushes it to the top
	e.must(others[0].call("POST", "/conversations/"+convs[others[0].ID]+"/messages", map[string]string{"body": "hello  "}), 201)

	// blocked pair is excluded
	e.must(me.call("POST", "/blocks", map[string]string{"userId": others[4].ID}), 204)

	var seen []string
	cursor := ""
	pages := 0
	for {
		path := "/matches?limit=2"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		r := e.must(me.call("GET", path, nil), 200).json()
		for _, raw := range r["items"].([]any) {
			m := raw.(map[string]any)
			seen = append(seen, m["user"].(map[string]any)["userId"].(string))
			if m["user"].(map[string]any)["userId"] == others[0].ID {
				lm := m["lastMessage"].(map[string]any)
				if lm["body"] != "hello" || lm["senderId"] != others[0].ID || m["unreadCount"] != float64(1) {
					t.Errorf("summary of active match: %v", m)
				}
			}
		}
		pages++
		if r["nextCursor"] == nil {
			break
		}
		cursor = r["nextCursor"].(string)
		if pages > 5 {
			t.Fatal("pagination does not terminate")
		}
	}
	want := []string{others[0].ID, others[3].ID, others[2].ID, others[1].ID}
	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Fatalf("order by latest activity, excluding blocked: got %v want %v", seen, want)
	}
	if pages != 2 {
		t.Errorf("expected 2 pages of 2, got %d", pages)
	}
	if r := me.call("GET", "/matches?cursor=garbage", nil); r.Status != 400 {
		t.Errorf("bad cursor: %d", r.Status)
	}
	if r := me.call("GET", "/matches?limit=51", nil); r.Status != 400 {
		t.Errorf("bad limit: %d", r.Status)
	}

	// unmatch: only a participant may; removes conversation + messages for both
	stranger := e.person(spec{Gender: "man", Lat: 10, Lon: 10})
	mid := matchIDs[others[0].ID]
	if r := stranger.call("DELETE", "/matches/"+mid, nil); r.Status != 404 {
		t.Errorf("stranger unmatch: %d", r.Status)
	}
	if r := me.call("DELETE", "/matches/not-a-uuid", nil); r.Status != 404 {
		t.Errorf("bad id: %d", r.Status)
	}
	e.must(me.call("DELETE", "/matches/"+mid, nil), 204)
	if r := me.call("DELETE", "/matches/"+mid, nil); r.Status != 404 {
		t.Errorf("unmatch twice: %d", r.Status)
	}
	if n := count(t, e.pool, `SELECT count(*) FROM match_messages WHERE conversation_id=$1`, convs[others[0].ID]); n != 0 {
		t.Errorf("messages must be deleted, %d left", n)
	}
	if r := others[0].call("GET", "/conversations/"+convs[others[0].ID]+"/messages", nil); r.Status != 404 {
		t.Errorf("other side after unmatch: %d", r.Status)
	}
	if n := count(t, e.pool, `SELECT count(*) FROM notifications WHERE data->>'matchId'=$1`, mid); n != 0 {
		t.Errorf("notifications of the match must be deleted: %d", n)
	}
	list := e.must(others[0].call("GET", "/matches", nil), 200).json()["items"].([]any)
	if len(list) != 0 {
		t.Errorf("other side still lists the match")
	}
}

func TestConversationPermissionsAndMessages(t *testing.T) {
	e := newEnv(t)
	a := e.person(spec{Name: "Ann", Gender: "woman", Lat: 10, Lon: 10})
	b := e.person(spec{Name: "Ben", Gender: "man", Lat: 10, Lon: 10})
	c := e.person(spec{Name: "Cid", Gender: "man", Lat: 10, Lon: 10})
	_, conv := e.matchUsers(a, b)
	base := "/conversations/" + conv

	// a third user gets nothing
	for _, tc := range []struct{ method, path string }{
		{"GET", base + "/messages"}, {"POST", base + "/messages"}, {"POST", base + "/read"},
	} {
		var body any
		if tc.method == "POST" && strings.HasSuffix(tc.path, "/messages") {
			body = map[string]string{"body": "let me in"}
		}
		r := c.call(tc.method, tc.path, body)
		if r.Status != 404 && r.Status != 403 {
			t.Errorf("third user %s %s: got %d %s", tc.method, tc.path, r.Status, r.Body)
		}
	}
	if n := count(t, e.pool, `SELECT count(*) FROM match_messages`); n != 0 {
		t.Fatalf("third user must not be able to write: %d", n)
	}
	if r := a.call("GET", "/conversations/not-a-uuid/messages", nil); r.Status != 404 {
		t.Errorf("bad conv id: %d", r.Status)
	}
	if r := a.call("GET", "/conversations/00000000-0000-0000-0000-000000000000/messages", nil); r.Status != 404 {
		t.Errorf("unknown conv: %d", r.Status)
	}

	// validation
	for name, body := range map[string]string{"empty": "", "spaces": "   \n ", "too long": strings.Repeat("é", 2001), "nul": "a\u0000b"} {
		if r := a.call("POST", base+"/messages", map[string]string{"body": body}); r.Status != 400 || r.code() != "invalid_request" {
			t.Errorf("%s: %d %s", name, r.Status, r.Body)
		}
	}
	e.must(a.call("POST", base+"/messages", map[string]string{"body": strings.Repeat("é", 2000)}), 201)

	// sending, ordering, pagination
	var ids []string
	for i := 0; i < 5; i++ {
		sender := a
		if i%2 == 1 {
			sender = b
		}
		r := e.must(sender.call("POST", base+"/messages", map[string]string{"body": fmt.Sprintf("  msg %d  ", i)}), 201).json()
		if r["body"] != fmt.Sprintf("msg %d", i) || r["senderId"] != sender.ID || r["conversationId"] != conv || r["readAt"] != nil || r["id"] == nil || r["createdAt"] == nil {
			t.Fatalf("message shape: %v", r)
		}
		ids = append(ids, r["id"].(string))
		time.Sleep(3 * time.Millisecond)
	}
	page1 := e.must(b.call("GET", base+"/messages?limit=4", nil), 200).json()
	items := page1["items"].([]any)
	if len(items) != 4 || items[0].(map[string]any)["id"] != ids[4] || page1["nextCursor"] == nil {
		t.Fatalf("page1: %v", page1)
	}
	page2 := e.must(b.call("GET", base+"/messages?limit=4&before="+page1["nextCursor"].(string), nil), 200).json()
	items2 := page2["items"].([]any)
	if len(items2) != 2 || items2[0].(map[string]any)["id"] != ids[0] && items2[0].(map[string]any)["body"] == "" {
		t.Fatalf("page2: %v", page2)
	}
	if page2["nextCursor"] != nil {
		t.Fatalf("last page must have null nextCursor")
	}
	if r := b.call("GET", base+"/messages?before=zzz", nil); r.Status != 400 {
		t.Errorf("bad before: %d", r.Status)
	}
	if r := b.call("GET", base+"/messages?limit=0", nil); r.Status != 400 {
		t.Errorf("bad limit: %d", r.Status)
	}

	// unread counts + read state
	sum := func(u *user) map[string]any {
		return e.must(u.call("GET", "/matches", nil), 200).json()["items"].([]any)[0].(map[string]any)
	}
	if sum(b)["unreadCount"] != float64(4) { // 1 long + msg0, msg2, msg4 from a
		t.Fatalf("b unread: %v", sum(b)["unreadCount"])
	}
	if sum(a)["unreadCount"] != float64(2) {
		t.Fatalf("a unread: %v", sum(a)["unreadCount"])
	}
	e.must(b.call("POST", base+"/read", nil), 204)
	if sum(b)["unreadCount"] != float64(0) || sum(a)["unreadCount"] != float64(2) {
		t.Fatalf("read must only affect the reader's incoming messages")
	}
	msgs := e.must(a.call("GET", base+"/messages?limit=10", nil), 200).json()["items"].([]any)
	for _, raw := range msgs {
		m := raw.(map[string]any)
		if m["senderId"] == a.ID && m["readAt"] == nil {
			t.Errorf("a's messages should be read after b read: %v", m)
		}
		if m["senderId"] == b.ID && m["readAt"] != nil {
			t.Errorf("b's messages are still unread by a: %v", m)
		}
	}

	// blocking: both directions get 403 blocked on send, history hidden
	e.must(a.call("POST", "/blocks", map[string]string{"userId": b.ID}), 204)
	for _, u := range []*user{a, b} {
		r := u.call("POST", base+"/messages", map[string]string{"body": "hi"})
		if r.Status != 403 || r.code() != "blocked" {
			t.Errorf("send while blocked (%s): %d %s", u.Name, r.Status, r.Body)
		}
		if r := u.call("GET", base+"/messages", nil); r.Status != 404 {
			t.Errorf("history while blocked (%s): %d", u.Name, r.Status)
		}
		if l := e.must(u.call("GET", "/matches", nil), 200).json()["items"].([]any); len(l) != 0 {
			t.Errorf("blocked pair must not appear in matches (%s)", u.Name)
		}
	}
	e.must(a.call("DELETE", "/blocks/"+b.ID, nil), 204)
	e.must(b.call("POST", base+"/messages", map[string]string{"body": "back again"}), 201)
}

func TestNotifications(t *testing.T) {
	e := newEnv(t)
	a := e.person(spec{Name: "Ann", Gender: "woman", Lat: 10, Lon: 10})
	b := e.person(spec{Name: "Ben", Gender: "man", Lat: 10, Lon: 10})
	match, conv := e.matchUsers(a, b)

	list := e.must(a.call("GET", "/notifications", nil), 200).json()
	if list["unreadCount"] != float64(1) || len(list["items"].([]any)) != 1 || list["nextCursor"] != nil {
		t.Fatalf("match notification: %v", list)
	}
	n := list["items"].([]any)[0].(map[string]any)
	data := n["data"].(map[string]any)
	if n["type"] != "match" || n["readAt"] != nil || data["matchId"] != match || data["conversationId"] != conv || data["userId"] != b.ID || n["title"] == "" {
		t.Fatalf("notification shape: %v", n)
	}

	// messages coalesce into one unread notification per conversation
	for i := 0; i < 3; i++ {
		e.must(b.call("POST", "/conversations/"+conv+"/messages", map[string]string{"body": fmt.Sprintf("hey %d", i)}), 201)
	}
	list = e.must(a.call("GET", "/notifications", nil), 200).json()
	if list["unreadCount"] != float64(2) || len(list["items"].([]any)) != 2 {
		t.Fatalf("expected match + one coalesced message notification: %v", list)
	}
	first := list["items"].([]any)[0].(map[string]any)
	if first["type"] != "message" || first["title"] != "Ben" || first["body"] != "hey 2" {
		t.Fatalf("message notification: %v", first)
	}
	// the sender gets none for their own messages
	if l := e.must(b.call("GET", "/notifications", nil), 200).json(); l["unreadCount"] != float64(1) {
		t.Fatalf("sender must only have the match notification: %v", l)
	}

	// pagination
	p1 := e.must(a.call("GET", "/notifications?limit=1", nil), 200).json()
	if len(p1["items"].([]any)) != 1 || p1["nextCursor"] == nil {
		t.Fatalf("page 1: %v", p1)
	}
	p2 := e.must(a.call("GET", "/notifications?limit=1&cursor="+p1["nextCursor"].(string), nil), 200).json()
	if len(p2["items"].([]any)) != 1 || p2["nextCursor"] != nil || p2["items"].([]any)[0].(map[string]any)["id"] == p1["items"].([]any)[0].(map[string]any)["id"] {
		t.Fatalf("page 2: %v", p2)
	}
	if r := a.call("GET", "/notifications?cursor=xx", nil); r.Status != 400 {
		t.Errorf("bad cursor: %d", r.Status)
	}

	// mark read: own only
	id := n["id"].(string)
	if r := b.call("POST", "/notifications/"+id+"/read", nil); r.Status != 404 {
		t.Errorf("reading someone else's notification: %d", r.Status)
	}
	if r := a.call("POST", "/notifications/not-a-uuid/read", nil); r.Status != 404 {
		t.Errorf("bad id: %d", r.Status)
	}
	e.must(a.call("POST", "/notifications/"+id+"/read", nil), 204)
	e.must(a.call("POST", "/notifications/"+id+"/read", nil), 204) // idempotent
	if l := e.must(a.call("GET", "/notifications", nil), 200).json(); l["unreadCount"] != float64(1) {
		t.Fatalf("after read: %v", l)
	}
	// opening the conversation reads its message notifications
	e.must(a.call("POST", "/conversations/"+conv+"/read", nil), 204)
	if l := e.must(a.call("GET", "/notifications", nil), 200).json(); l["unreadCount"] != float64(0) {
		t.Fatalf("conversation read should clear message notification: %v", l)
	}
	// a new message after reading creates a fresh notification
	e.must(b.call("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "again"}), 201)
	if l := e.must(a.call("GET", "/notifications", nil), 200).json(); l["unreadCount"] != float64(1) {
		t.Fatalf("fresh message notification: %v", l)
	}
	e.must(a.call("POST", "/notifications/read-all", nil), 204)
	if l := e.must(a.call("GET", "/notifications", nil), 200).json(); l["unreadCount"] != float64(0) {
		t.Fatalf("read-all: %v", l)
	}
	if l := e.must(b.call("GET", "/notifications", nil), 200).json(); l["unreadCount"] != float64(1) {
		t.Fatalf("read-all must not touch other users: %v", l)
	}
}

func TestBlocksAndReports(t *testing.T) {
	e := newEnv(t)
	a := e.person(spec{Name: "Ann", Gender: "woman", Lat: 10, Lon: 10})
	b := e.person(spec{Name: "Ben", Gender: "man", Lat: 10, Lon: 10})

	if r := a.call("POST", "/blocks", map[string]string{"userId": a.ID}); r.Status != 400 {
		t.Errorf("block self: %d", r.Status)
	}
	if r := a.call("POST", "/blocks", map[string]string{"userId": "nope"}); r.Status != 400 {
		t.Errorf("block bad id: %d", r.Status)
	}
	if r := a.call("POST", "/blocks", map[string]string{"userId": "00000000-0000-0000-0000-000000000000"}); r.Status != 404 {
		t.Errorf("block unknown: %d", r.Status)
	}
	e.must(a.call("POST", "/blocks", map[string]string{"userId": b.ID}), 204)
	e.must(a.call("POST", "/blocks", map[string]string{"userId": b.ID}), 204) // idempotent
	if n := count(t, e.pool, `SELECT count(*) FROM blocks`); n != 1 {
		t.Fatalf("blocks: %d", n)
	}
	list := e.must(a.call("GET", "/blocks", nil), 200).json()["items"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["userId"] != b.ID || list[0].(map[string]any)["firstName"] != "Ben" || list[0].(map[string]any)["blockedAt"] == nil {
		t.Fatalf("blocks list: %v", list)
	}
	if l := e.must(b.call("GET", "/blocks", nil), 200).json()["items"].([]any); len(l) != 0 {
		t.Fatalf("the blocked user must not see who blocked them")
	}
	// hidden in discover both ways and unswipeable
	if contains(discoverIDs(t, a, ""), b.ID) || contains(discoverIDs(t, b, ""), a.ID) {
		t.Error("blocked pair must be hidden in discover")
	}
	if r := b.swipe(a, "like"); r.Status != 404 {
		t.Errorf("swipe on blocker: %d", r.Status)
	}
	e.must(a.call("DELETE", "/blocks/"+b.ID, nil), 204)
	e.must(a.call("DELETE", "/blocks/"+b.ID, nil), 204)
	if !contains(discoverIDs(t, a, ""), b.ID) {
		t.Error("unblocked user should be back in discover")
	}

	// reports
	for name, body := range map[string]map[string]string{
		"self":         {"userId": a.ID, "reason": "spam"},
		"bad reason":   {"userId": b.ID, "reason": "dislike"},
		"empty reason": {"userId": b.ID},
		"long details": {"userId": b.ID, "reason": "spam", "details": strings.Repeat("x", 1001)},
		"bad id":       {"userId": "zzz", "reason": "spam"},
	} {
		if r := a.call("POST", "/reports", body); r.Status != 400 || r.code() != "invalid_request" {
			t.Errorf("report %s: %d %s", name, r.Status, r.Body)
		}
	}
	if r := a.call("POST", "/reports", map[string]string{"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"}); r.Status != 404 {
		t.Errorf("report unknown: %d", r.Status)
	}
	for _, reason := range []string{"spam", "fake_profile", "harassment", "inappropriate_content", "underage", "other"} {
		r := e.must(a.call("POST", "/reports", map[string]string{"userId": b.ID, "reason": reason, "details": "because"}), 201).json()
		if r["id"] == nil {
			t.Fatalf("report id: %v", r)
		}
	}
	if n := count(t, e.pool, `SELECT count(*) FROM reports WHERE reporter_id=$1 AND reported_id=$2`, a.ID, b.ID); n != 6 {
		t.Fatalf("reports stored: %d", n)
	}
}

func TestDeleteAccountCascades(t *testing.T) {
	e := newEnv(t)
	a := e.person(spec{Name: "Ann", Gender: "woman", Lat: 10, Lon: 10, Interests: []int{1, 2}})
	b := e.person(spec{Name: "Ben", Gender: "man", Lat: 10, Lon: 10})
	c := e.person(spec{Name: "Cid", Gender: "man", Lat: 10, Lon: 10})
	_, conv := e.matchUsers(a, b)
	e.must(b.call("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "hi"}), 201)
	e.must(a.call("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "yo"}), 201)
	e.must(a.swipe(c, "pass"), 200)
	e.must(c.call("POST", "/blocks", map[string]string{"userId": a.ID}), 204)
	e.must(a.call("POST", "/reports", map[string]string{"userId": c.ID, "reason": "spam"}), 201)
	e.must(c.call("POST", "/reports", map[string]string{"userId": a.ID, "reason": "spam"}), 201)
	e.must(e.call("POST", "/auth/forgot-password", "", map[string]string{"email": a.Email}), 204)

	// wrong / missing password
	if r := a.call("DELETE", "/me", map[string]string{"password": "wrong-wrong-1"}); r.Status != 403 {
		t.Fatalf("wrong password: %d %s", r.Status, r.Body)
	}
	if r := a.call("DELETE", "/me", map[string]string{}); r.Status != 403 {
		t.Fatalf("missing password: %d", r.Status)
	}
	e.must(a.call("GET", "/me", nil), 200)

	e.must(a.call("DELETE", "/me", map[string]string{"password": testPassword}), 204)

	if r := a.call("GET", "/me", nil); r.Status != 401 {
		t.Fatalf("token of deleted user: %d", r.Status)
	}
	if r := e.call("POST", "/auth/login", "", map[string]string{"email": a.Email, "password": testPassword}); r.Status != 401 {
		t.Fatalf("login of deleted user: %d", r.Status)
	}
	for _, q := range []string{
		`SELECT count(*) FROM users WHERE id=$1`,
		`SELECT count(*) FROM profiles WHERE user_id=$1`,
		`SELECT count(*) FROM preferences WHERE user_id=$1`,
		`SELECT count(*) FROM user_interests WHERE user_id=$1`,
		`SELECT count(*) FROM photos WHERE user_id=$1`,
		`SELECT count(*) FROM swipes WHERE from_user_id=$1 OR to_user_id=$1`,
		`SELECT count(*) FROM matches WHERE user_a=$1 OR user_b=$1`,
		`SELECT count(*) FROM match_messages WHERE sender_id=$1`,
		`SELECT count(*) FROM blocks WHERE blocker_id=$1 OR blocked_id=$1`,
		`SELECT count(*) FROM reports WHERE reporter_id=$1 OR reported_id=$1`,
		`SELECT count(*) FROM notifications WHERE user_id=$1 OR data->>'userId'=$1::text`,
		`SELECT count(*) FROM sessions WHERE user_id=$1`,
		`SELECT count(*) FROM password_resets WHERE user_id=$1`,
	} {
		if n := count(t, e.pool, q, a.ID); n != 0 {
			t.Errorf("%d rows left for %q", n, q)
		}
	}
	if n := count(t, e.pool, `SELECT count(*) FROM match_conversations`); n != 0 {
		t.Errorf("conversation of the deleted match must be gone: %d", n)
	}
	if n := count(t, e.pool, `SELECT count(*) FROM match_messages`); n != 0 {
		t.Errorf("messages must be gone: %d", n)
	}
	// others are untouched
	e.must(b.call("GET", "/me", nil), 200)
	if l := e.must(b.call("GET", "/matches", nil), 200).json()["items"].([]any); len(l) != 0 {
		t.Errorf("b must not see the deleted match")
	}
}

func TestRateLimits(t *testing.T) {
	limits := hugeLimits()
	limits.Auth = newTestLimiter(3, time.Minute)
	limits.Swipes = newTestLimiter(2, time.Minute)
	limits.Message = newTestLimiter(2, time.Minute)
	limits.Reports = newTestLimiter(2, time.Hour)
	e := newEnvWith(t, limits)

	// per-IP on /auth/*
	for i := 0; i < 3; i++ {
		if r := e.call("POST", "/auth/login", "", map[string]string{"email": "x@test.invalid", "password": "whatever-1"}); r.Status != 401 {
			t.Fatalf("attempt %d: %d", i, r.Status)
		}
	}
	r := e.call("POST", "/auth/login", "", map[string]string{"email": "x@test.invalid", "password": "whatever-1"})
	if r.Status != 429 || r.code() != "rate_limited" || r.Header.Get("Retry-After") == "" {
		t.Fatalf("auth limiter: %d %s %v", r.Status, r.Body, r.Header)
	}
	if r := e.call("POST", "/auth/forgot-password", "", map[string]string{"email": "x@test.invalid"}); r.Status != 429 {
		t.Fatalf("limiter covers every /auth route: %d", r.Status)
	}
	// non-auth routes are not affected by the auth limiter
	e.must(e.call("GET", "/health", "", nil), 200)

	// per-user on swipes / messages / reports; another user is unaffected
	limits.Auth = newTestLimiter(1000, time.Minute)
	e2 := newEnvWith(t, limits)
	a := e2.person(spec{Gender: "woman", Lat: 10, Lon: 10})
	b := e2.person(spec{Gender: "man", Lat: 10, Lon: 10})
	c := e2.person(spec{Gender: "man", Lat: 10, Lon: 10})
	d := e2.person(spec{Gender: "man", Lat: 10, Lon: 10})
	e2.must(a.swipe(b, "like"), 200)
	e2.must(a.swipe(c, "like"), 200)
	if r := a.swipe(d, "like"); r.Status != 429 || r.Header.Get("Retry-After") == "" {
		t.Fatalf("swipe limiter: %d %s", r.Status, r.Body)
	}
	e2.must(b.swipe(a, "like"), 200)

	_, conv := e2.matchUsersLimited(b, a) // b<->a already liked each other
	for i := 0; i < 2; i++ {
		e2.must(b.call("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "x"}), 201)
	}
	if r := b.call("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "x"}); r.Status != 429 {
		t.Fatalf("message limiter: %d", r.Status)
	}
	e2.must(a.call("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "x"}), 201)

	for i := 0; i < 2; i++ {
		e2.must(a.call("POST", "/reports", map[string]string{"userId": d.ID, "reason": "spam"}), 201)
	}
	if r := a.call("POST", "/reports", map[string]string{"userId": d.ID, "reason": "spam"}); r.Status != 429 {
		t.Fatalf("report limiter: %d", r.Status)
	}
}

// matchUsersLimited returns ids of an existing match without more swipes.
func (e *env) matchUsersLimited(a, b *user) (string, string) {
	e.t.Helper()
	var mid, cid string
	if err := e.pool.QueryRow(context.Background(), `
		SELECT m.id, c.id FROM matches m JOIN match_conversations c ON c.match_id=m.id
		WHERE m.user_a=LEAST($1::uuid,$2::uuid) AND m.user_b=GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&mid, &cid); err != nil {
		e.t.Fatal(err)
	}
	return mid, cid
}

func TestRequestSizeLimits(t *testing.T) {
	e := newEnv(t)
	u := e.register()
	huge := `{"firstName":"` + strings.Repeat("a", 2<<20) + `"}`
	r := e.raw("PUT", "/me/profile", u.Access, "application/json", strings.NewReader(huge))
	if r.Status != 413 || r.code() != "payload_too_large" {
		t.Fatalf("oversized JSON: %d %s", r.Status, r.Body)
	}
	// chunked body without Content-Length is limited too
	r = e.raw("POST", "/auth/login", "", "application/json", strings.NewReader(`{"email":"`+strings.Repeat("a", 2<<20)+`"}`))
	if r.Status != 413 {
		t.Fatalf("oversized login body: %d", r.Status)
	}
}
