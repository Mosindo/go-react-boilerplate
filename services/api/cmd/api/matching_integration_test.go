package main

import (
	"net/http"
	"sync"
	"testing"
)

func swipe(e *env, from user, target, action string) map[string]any {
	e.t.Helper()
	return e.expect(e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": target, "action": action}, from.Token), http.StatusOK).JSON(e.t)
}

func TestDiscoveryFiltersByMutualPreferences(t *testing.T) {
	e := newEnv(t)
	man, woman := e.pair()
	gayMan := e.register("gayman")
	e.onboard(gayMan, "Gabi", profileOpts{gender: "man", interestedIn: []string{"man"}})
	olderWoman := e.register("older")
	e.onboard(olderWoman, "Olga", profileOpts{gender: "woman", interestedIn: []string{"man"}, age: 45})
	notOnboarded := e.register("empty")

	ids := e.discoverIDs(man)
	if !contains(ids, woman.ID) {
		t.Fatal("compatible woman must be discoverable")
	}
	if contains(ids, man.ID) {
		t.Fatal("never discover yourself")
	}
	if contains(ids, gayMan.ID) {
		t.Fatal("man seeking women must not see a man")
	}
	if contains(ids, olderWoman.ID) {
		t.Fatal("age range must be respected")
	}
	if contains(ids, notOnboarded.ID) {
		t.Fatal("profiles without onboarding must be hidden")
	}
	// Mutuality: the woman only wants men, so gayMan (who wants men) is visible to her? No: he is a man seeking men,
	// she is a woman, so he does not want her.
	if contains(e.discoverIDs(woman), gayMan.ID) {
		t.Fatal("mutual preference must be required")
	}

	// A user who is not discoverable disappears.
	e.expect(e.do(http.MethodPut, "/me/preferences", map[string]any{
		"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 30, "maxDistanceKm": 100, "discoverable": false, "showDistance": true,
	}, woman.Token), http.StatusOK)
	if contains(e.discoverIDs(man), woman.ID) {
		t.Fatal("non-discoverable users must be hidden")
	}

	e.expect(e.do(http.MethodGet, "/discover", nil, notOnboarded.Token), http.StatusConflict)
	e.expect(e.do(http.MethodGet, "/discover?limit=0", nil, man.Token), http.StatusBadRequest)
	e.expect(e.do(http.MethodGet, "/discover?limit=abc", nil, man.Token), http.StatusBadRequest)
}

func cardFor(t *testing.T, e *env, viewer user, id string) map[string]any {
	t.Helper()
	r := e.expect(e.do(http.MethodGet, "/discover?limit=20", nil, viewer.Token), http.StatusOK).JSON(t)
	for _, p := range r["profiles"].([]any) {
		if m := p.(map[string]any); m["userId"] == id {
			return m
		}
	}
	return nil
}

func TestDiscoveryDistanceIsApproximate(t *testing.T) {
	e := newEnv(t)
	f := func(v float64) *float64 { return &v }
	viewer := e.register("geo-viewer")
	near := e.register("geo-near")
	mid := e.register("geo-mid")
	far := e.register("geo-far")
	hidden := e.register("geo-hidden")
	e.onboard(viewer, "Vic", profileOpts{gender: "man", interestedIn: []string{"woman"}, lat: f(45.7640), lng: f(4.8357)})
	e.onboard(near, "Nina", profileOpts{gender: "woman", interestedIn: []string{"man"}, lat: f(45.7660), lng: f(4.8790)})   // Villeurbanne ~3 km
	e.onboard(mid, "Mia", profileOpts{gender: "woman", interestedIn: []string{"man"}, lat: f(45.5256), lng: f(4.8744)})     // Vienne ~27 km
	e.onboard(far, "Fay", profileOpts{gender: "woman", interestedIn: []string{"man"}, lat: f(48.8566), lng: f(2.3522)})     // Paris ~390 km
	e.onboard(hidden, "Hana", profileOpts{gender: "woman", interestedIn: []string{"man"}, lat: f(45.7700), lng: f(4.8400)}) // very close, hides distance
	e.expect(e.do(http.MethodPut, "/me/preferences", map[string]any{
		"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 30, "maxDistanceKm": 100, "discoverable": true, "showDistance": false,
	}, hidden.Token), http.StatusOK)

	if c := cardFor(t, e, viewer, near.ID); c == nil || c["distanceKm"].(float64) != 5 {
		t.Fatalf("near card: %v", c)
	}
	if c := cardFor(t, e, viewer, mid.ID); c == nil || c["distanceKm"].(float64) != 30 {
		t.Fatalf("mid card: %v", c)
	}
	if cardFor(t, e, viewer, far.ID) != nil {
		t.Fatal("profiles beyond maxDistance must be excluded")
	}
	c := cardFor(t, e, viewer, hidden.ID)
	if c == nil || c["distanceKm"] != nil {
		t.Fatalf("showDistance=false must hide the distance: %v", c)
	}
	// Cards never leak private fields.
	for _, k := range []string{"email", "birthDate", "latitude", "longitude", "lat", "lng"} {
		if _, ok := c[k]; ok {
			t.Fatalf("card leaks %q", k)
		}
	}
}

func TestDiscoveryRanksSharedInterestsFirst(t *testing.T) {
	e := newEnv(t)
	viewer := e.register("rank-viewer")
	plain := e.register("rank-plain")
	shared := e.register("rank-shared")
	e.onboard(viewer, "Vic", profileOpts{gender: "man", interestedIn: []string{"woman"}, interests: []string{"cinema", "yoga", "wine"}})
	e.onboard(plain, "Pia", profileOpts{gender: "woman", interestedIn: []string{"man"}, interests: []string{"gaming"}})
	e.onboard(shared, "Sia", profileOpts{gender: "woman", interestedIn: []string{"man"}, interests: []string{"cinema", "yoga"}})

	ids := e.discoverIDs(viewer)
	if len(ids) < 2 || ids[0] != shared.ID {
		t.Fatalf("shared-interest profile should rank first, got %v", ids)
	}
	card := cardFor(t, e, viewer, shared.ID)
	sharedCount := 0
	for _, i := range card["interests"].([]any) {
		if i.(map[string]any)["shared"] == true {
			sharedCount++
		}
	}
	if sharedCount != 2 {
		t.Fatalf("expected 2 shared interests flagged, got %d", sharedCount)
	}
}

func TestLikeThenLikeBackCreatesExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()

	r := swipe(e, a, b.ID, "like")
	if r["matched"] != false || r["alreadyProcessed"] != false {
		t.Fatalf("first like must not match: %v", r)
	}
	if contains(e.discoverIDs(a), b.ID) {
		t.Fatal("a swiped profile must never be shown again")
	}
	// Repeating is idempotent.
	r = swipe(e, a, b.ID, "like")
	if r["alreadyProcessed"] != true || r["matched"] != false {
		t.Fatalf("repeated like must be reported as processed: %v", r)
	}
	// A swipe decision is final: switching to pass does nothing.
	if r = swipe(e, a, b.ID, "pass"); r["action"] != "like" || r["alreadyProcessed"] != true {
		t.Fatalf("decision must be final: %v", r)
	}

	// B sees A liked them first (ranking boost) and likes back.
	r = swipe(e, b, a.ID, "like")
	if r["matched"] != true || r["match"] == nil {
		t.Fatalf("mutual like must match: %v", r)
	}
	match := r["match"].(map[string]any)
	if match["user"].(map[string]any)["userId"] != a.ID || match["user"].(map[string]any)["firstName"] != "Adam" {
		t.Fatalf("match summary from the wrong perspective: %v", match)
	}
	if match["user"].(map[string]any)["thumbUrl"] == nil {
		t.Fatal("match summary should carry a thumbnail")
	}
	again := swipe(e, b, a.ID, "like")
	if again["matched"] != true || again["alreadyProcessed"] != true || again["match"].(map[string]any)["id"] != match["id"] {
		t.Fatalf("repeat must return the same match: %v", again)
	}

	var count int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, a.ID, b.ID).Scan(&count)
	if count != 1 {
		t.Fatalf("expected exactly one match row, got %d", count)
	}
	for _, u := range []user{a, b} {
		list := e.expect(e.do(http.MethodGet, "/matches", nil, u.Token), http.StatusOK).JSON(t)["matches"].([]any)
		if len(list) != 1 {
			t.Fatalf("each user must see the match once, got %d", len(list))
		}
	}
}

func TestPassNeverMatches(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	swipe(e, a, b.ID, "pass")
	if r := swipe(e, b, a.ID, "like"); r["matched"] != false {
		t.Fatalf("a pass followed by a like must not match: %v", r)
	}
	if contains(e.discoverIDs(a), b.ID) {
		t.Fatal("passed profiles must not reappear")
	}
}

func TestSwipeValidation(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	e.expect(e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": a.ID, "action": "like"}, a.Token), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": b.ID, "action": "superlike"}, a.Token), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": "123e4567-e89b-12d3-a456-426614174000", "action": "like"}, a.Token), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": "nope", "action": "like"}, a.Token), http.StatusNotFound)
}

func TestConcurrentMutualLikesProduceOneMatch(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 8; round++ {
		a, b := e.pair()
		var wg sync.WaitGroup
		results := make([]map[string]any, 2)
		for i, pair := range [][2]user{{a, b}, {b, a}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i] = swipe(e, pair[0], pair[1].ID, "like")
			}()
		}
		wg.Wait()
		var count int
		_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, a.ID, b.ID).Scan(&count)
		if count != 1 {
			t.Fatalf("round %d: expected one match, got %d (%v)", round, count, results)
		}
		if results[0]["matched"] != true && results[1]["matched"] != true {
			t.Fatalf("round %d: at least one side must learn about the match: %v", round, results)
		}
	}
}

func TestProfileVisibilityRules(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	stranger := e.register("vis-stranger")
	e.onboard(stranger, "Sam", profileOpts{gender: "man", interestedIn: []string{"woman"}})

	card := e.expect(e.do(http.MethodGet, "/profiles/"+b.ID, nil, a.Token), http.StatusOK).JSON(t)
	if card["firstName"] != "Berta" || len(card["photos"].([]any)) != 1 {
		t.Fatalf("unexpected card %v", card)
	}
	e.expect(e.do(http.MethodGet, "/profiles/"+a.ID, nil, a.Token), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/profiles/not-a-uuid", nil, a.Token), http.StatusNotFound)

	// Hidden profiles stay reachable for matches only.
	e.expect(e.do(http.MethodPut, "/me/preferences", map[string]any{
		"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 30, "maxDistanceKm": 100, "discoverable": false, "showDistance": true,
	}, b.Token), http.StatusOK)
	e.expect(e.do(http.MethodGet, "/profiles/"+b.ID, nil, stranger.Token), http.StatusNotFound)
	e.match(a, b)
	e.expect(e.do(http.MethodGet, "/profiles/"+b.ID, nil, a.Token), http.StatusOK)
}
