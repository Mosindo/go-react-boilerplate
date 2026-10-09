package main

import (
	"net/http"
	"sync"
	"testing"
)

func TestDiscoveryHonoursMutualPreferences(t *testing.T) {
	e := newEnv(t)
	ar := newArea()

	me := e.register("")
	e.onboard(me, ar, func() persona {
		p := ar.persona("Me", "man", 30, "woman")
		p.MinAge, p.MaxAge, p.MaxDistance = 25, 35, 20
		return p
	}())

	ok := e.register("")
	e.onboard(ok, ar, ar.persona("Ok", "woman", 28, "man"))

	wrongGender := e.register("")
	e.onboard(wrongGender, ar, ar.persona("Man", "man", 28, "man", "woman"))

	tooOld := e.register("")
	e.onboard(tooOld, ar, ar.persona("Old", "woman", 50, "man"))

	tooYoungForThem := e.register("")
	pt := ar.persona("Picky", "woman", 28, "man")
	pt.MinAge, pt.MaxAge = 18, 29 // I am 30: not within her range
	e.onboard(tooYoungForThem, ar, pt)

	notInterestedInMe := e.register("")
	e.onboard(notInterestedInMe, ar, ar.persona("Women", "woman", 28, "woman"))

	farAway := e.register("")
	far := ar.persona("Far", "woman", 28, "man")
	far.Dlat = 1.0 // ~111 km north, beyond my 20 km
	far.MaxDistance = 500
	e.onboard(farAway, ar, far)

	farForHer := e.register("")
	farHer := ar.persona("Homebody", "woman", 28, "man")
	farHer.Dlat = 0.1 // ~11 km: inside my range, outside hers
	farHer.MaxDistance = 5
	e.onboard(farForHer, ar, farHer)

	hidden := e.register("")
	hp := ar.persona("Hidden", "woman", 28, "man")
	hp.Discoverable = false
	e.onboard(hidden, ar, hp)

	noPhoto := e.register("")
	np := ar.persona("NoPhoto", "woman", 28, "man")
	np.Photos = 0
	e.onboard(noPhoto, ar, np)

	ids := e.discoverIDs(me)
	if len(ids) != 1 || ids[0] != ok.ID {
		t.Fatalf("expected only the compatible profile, got %v (ok=%s)", ids, ok.ID)
	}

	// Cards are complete and private data is absent.
	r := e.expect(e.do("GET", "/discover?limit=5", nil, me.Token), http.StatusOK)
	card := r.m(t)["profiles"].([]any)[0].(map[string]any)
	if card["firstName"] != "Ok" || len(card["photos"].([]any)) != 1 || len(card["interests"].([]any)) != 2 {
		t.Fatalf("incomplete card %v", card)
	}
	for _, k := range []string{"email", "birthDate", "latitude", "longitude"} {
		if _, leaked := card[k]; leaked {
			t.Fatalf("discover leaks %s", k)
		}
	}

	e.expect(e.do("GET", "/discover?limit=0", nil, me.Token), http.StatusBadRequest)
	e.expect(e.do("GET", "/discover?limit=21", nil, me.Token), http.StatusBadRequest)
	e.expect(e.do("GET", "/discover", nil, ""), http.StatusUnauthorized)
}

func TestLikeMatchFlowWithoutDuplicates(t *testing.T) {
	e := newEnv(t)
	a, b, _ := e.pair()

	if ids := e.discoverIDs(a); len(ids) != 1 || ids[0] != b.ID {
		t.Fatalf("a should discover b, got %v", ids)
	}

	// A likes B: no match yet, B is no longer proposed to A, but A is proposed to B (first, as a liker).
	res := e.swipe(a, b.ID, "like")
	if res["matched"] != false || res["alreadySwiped"] != false || res["match"] != nil {
		t.Fatalf("unexpected first swipe result %v", res)
	}
	if ids := e.discoverIDs(a); len(ids) != 0 {
		t.Fatalf("a profile already processed must not reappear: %v", ids)
	}
	if likes := e.expect(e.do("GET", "/likes/received", nil, b.Token), http.StatusOK).m(t)["profiles"].([]any); len(likes) != 1 {
		t.Fatalf("b should see a's like (free feature), got %v", likes)
	}

	// Repeating the like is idempotent.
	again := e.swipe(a, b.ID, "like")
	if again["alreadySwiped"] != true || again["matched"] != false {
		t.Fatalf("repeat like: %v", again)
	}
	// Changing your mind via a second swipe does not rewrite history.
	if flip := e.swipe(a, b.ID, "pass"); flip["alreadySwiped"] != true {
		t.Fatalf("second swipe must be ignored: %v", flip)
	}

	// B likes A back: match with a conversation, for both.
	m := e.swipe(b, a.ID, "like")
	if m["matched"] != true || m["alreadySwiped"] != false {
		t.Fatalf("expected a match: %v", m)
	}
	ref := m["match"].(map[string]any)
	if ref["id"] == "" || ref["conversationId"] == "" {
		t.Fatalf("match without ids: %v", ref)
	}
	if likes := e.expect(e.do("GET", "/likes/received", nil, b.Token), http.StatusOK).m(t)["profiles"].([]any); len(likes) != 0 {
		t.Fatalf("processed likes must leave the inbox: %v", likes)
	}

	// Repeating after a match returns the same match and never creates another.
	rep := e.swipe(b, a.ID, "like")
	if rep["matched"] != true || rep["alreadySwiped"] != true || rep["match"].(map[string]any)["id"] != ref["id"] {
		t.Fatalf("repeat after match: %v", rep)
	}
	var count int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, a.ID, b.ID).Scan(&count)
	if count != 1 {
		t.Fatalf("expected exactly one match row, got %d", count)
	}

	for _, u := range []*user{a, b} {
		list := e.expect(e.do("GET", "/matches", nil, u.Token), http.StatusOK).m(t)["matches"].([]any)
		if len(list) != 1 {
			t.Fatalf("expected one match in the list, got %v", list)
		}
	}

	// Both sides received an in-app notification.
	for _, u := range []*user{a, b} {
		n := e.expect(e.do("GET", "/notifications", nil, u.Token), http.StatusOK).m(t)
		if n["unreadCount"].(float64) != 1 || n["notifications"].([]any)[0].(map[string]any)["type"] != "match" {
			t.Fatalf("match notification missing: %v", n)
		}
	}
}

func TestPassNeverMatches(t *testing.T) {
	e := newEnv(t)
	a, b, _ := e.pair()
	e.swipe(a, b.ID, "pass")
	e.swipe(b, a.ID, "like")
	if ids := e.discoverIDs(a); len(ids) != 0 {
		t.Fatalf("passed profile reappeared: %v", ids)
	}
	if likes := e.expect(e.do("GET", "/likes/received", nil, a.Token), http.StatusOK).m(t)["profiles"].([]any); len(likes) != 0 {
		t.Fatalf("a passed on b, b's like must not be offered: %v", likes)
	}
	if list := e.expect(e.do("GET", "/matches", nil, a.Token), http.StatusOK).m(t)["matches"].([]any); len(list) != 0 {
		t.Fatalf("a pass must never create a match: %v", list)
	}
}

func TestSwipeValidation(t *testing.T) {
	e := newEnv(t)
	a, b, _ := e.pair()
	incomplete := e.register("")

	e.expect(e.do("POST", "/swipes", map[string]any{"userId": a.ID, "action": "like"}, a.Token), http.StatusBadRequest)
	e.expect(e.do("POST", "/swipes", map[string]any{"userId": b.ID, "action": "superlike"}, a.Token), http.StatusBadRequest)
	e.expect(e.do("POST", "/swipes", map[string]any{"userId": "nope", "action": "like"}, a.Token), http.StatusNotFound)
	e.expect(e.do("POST", "/swipes", map[string]any{"userId": "00000000-0000-0000-0000-000000000000", "action": "like"}, a.Token), http.StatusNotFound)
	e.expect(e.do("POST", "/swipes", map[string]any{"userId": incomplete.ID, "action": "like"}, a.Token), http.StatusNotFound) // no profile
	e.expect(e.do("POST", "/swipes", map[string]any{"userId": b.ID, "action": "like"}, incomplete.Token), http.StatusForbidden)
	e.expect(e.do("POST", "/swipes", map[string]any{"userId": b.ID, "action": "like"}, ""), http.StatusUnauthorized)
}

func TestSimultaneousLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 8; round++ {
		a, b, _ := e.pair()
		var wg sync.WaitGroup
		results := make([]map[string]any, 2)
		for i, pair := range [][2]*user{{a, b}, {b, a}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i] = e.expect(e.do("POST", "/swipes", map[string]any{"userId": pair[1].ID, "action": "like"}, pair[0].Token), http.StatusOK).m(t)
			}()
		}
		wg.Wait()
		matched := 0
		for _, r := range results {
			if r["matched"] == true {
				matched++
			}
		}
		if matched == 0 {
			t.Fatalf("round %d: mutual likes produced no match: %v", round, results)
		}
		var rows, convs int
		_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, a.ID, b.ID).Scan(&rows)
		_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM conversations c JOIN matches m ON m.id = c.match_id WHERE m.user_a IN ($1,$2) AND m.user_b IN ($1,$2)`, a.ID, b.ID).Scan(&convs)
		if rows != 1 || convs != 1 {
			t.Fatalf("round %d: %d matches, %d conversations", round, rows, convs)
		}
	}
}

func TestUnmatch(t *testing.T) {
	e := newEnv(t)
	a, b, convID := e.matchPair()
	stranger := e.register("")
	list := e.expect(e.do("GET", "/matches", nil, a.Token), http.StatusOK).m(t)["matches"].([]any)
	matchID := list[0].(map[string]any)["id"].(string)

	e.expect(e.do("DELETE", "/matches/"+matchID, nil, stranger.Token), http.StatusNotFound)
	e.expect(e.do("DELETE", "/matches/"+matchID, nil, a.Token), http.StatusNoContent)
	e.expect(e.do("DELETE", "/matches/"+matchID, nil, a.Token), http.StatusNotFound)

	for _, u := range []*user{a, b} {
		if l := e.expect(e.do("GET", "/matches", nil, u.Token), http.StatusOK).m(t)["matches"].([]any); len(l) != 0 {
			t.Fatalf("match still listed: %v", l)
		}
		e.expect(e.do("GET", "/conversations/"+convID+"/messages", nil, u.Token), http.StatusNotFound)
		e.expect(e.do("POST", "/conversations/"+convID+"/messages", map[string]any{"body": "hi"}, u.Token), http.StatusNotFound)
		if ids := e.discoverIDs(u); len(ids) != 0 {
			t.Fatalf("unmatched users must never be proposed again: %v", ids)
		}
	}
	// Liking again cannot revive the match.
	e.swipe(a, b.ID, "like")
	if l := e.expect(e.do("GET", "/matches", nil, a.Token), http.StatusOK).m(t)["matches"].([]any); len(l) != 0 {
		t.Fatalf("unmatched pair was revived: %v", l)
	}
}

func TestLikesAndSwipesPerUserAreUnlimited(t *testing.T) {
	// The product has no daily like cap: 30 consecutive swipes by one user all succeed.
	e := newEnv(t)
	ar := newArea()
	me := e.register("")
	e.onboard(me, ar, ar.persona("Me", "man", 30, "woman"))
	for i := 0; i < 30; i++ {
		other := e.register("")
		e.onboard(other, ar, func() persona { p := ar.persona("N", "woman", 25, "man"); p.Photos = 1; return p }())
		e.swipe(me, other.ID, "like")
	}
}
