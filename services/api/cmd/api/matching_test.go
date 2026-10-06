package main

import (
	"net/http"
	"sync"
	"testing"
)

func TestDiscoveryAppliesMutualPreferences(t *testing.T) {
	e := newEnv(t)
	me := e.newPerson(persona{Name: "Me", Gender: "man", InterestedIn: []string{"woman"}, Age: 40, MinAge: 30, MaxAge: 45, MaxDistance: 50})

	ok := e.newPerson(persona{Name: "Okay", Gender: "woman", InterestedIn: []string{"man"}, Age: 35, MinAge: 35, MaxAge: 50})
	wrongGender := e.newPerson(persona{Name: "Gus", Gender: "man", InterestedIn: []string{"man"}, Age: 35})
	theyDontWantMe := e.newPerson(persona{Name: "Pickyone", Gender: "woman", InterestedIn: []string{"woman"}, Age: 35})
	tooYoungForMe := e.newPerson(persona{Name: "Young", Gender: "woman", InterestedIn: []string{"man"}, Age: 22})
	theyWantYounger := e.newPerson(persona{Name: "Youngseeker", Gender: "woman", InterestedIn: []string{"man"}, Age: 35, MinAge: 18, MaxAge: 30})
	far := e.newPerson(persona{Name: "Far", Gender: "woman", InterestedIn: []string{"man"}, Age: 35, Lat: lyonLat, Lng: lyonLng, MaxDistance: 500})
	theirRangeShort := e.newPerson(persona{Name: "Shortrange", Gender: "woman", InterestedIn: []string{"man"}, Age: 35, Lat: 49.30, Lng: 2.35, MaxDistance: 5}) // ~50km north, only wants 5km
	hidden := e.newPerson(persona{Name: "Hidden", Gender: "woman", InterestedIn: []string{"man"}, Age: 35})
	e.expect(e.do(http.MethodPut, "/profile/privacy", hidden.Token, map[string]any{"showAge": true, "showDistance": true, "discoverable": false}), http.StatusOK)

	incomplete := e.register("noprofile") // no onboarding at all
	_ = incomplete

	got := e.discoverIDs(me)
	if !contains(got, ok.ID) {
		t.Fatalf("compatible profile must be shown, got %v", got)
	}
	for name, id := range map[string]string{
		"wrong gender": wrongGender.ID, "they don't want me": theyDontWantMe.ID, "outside my age range": tooYoungForMe.ID,
		"I am outside their age range": theyWantYounger.ID, "too far": far.ID, "outside their distance": theirRangeShort.ID,
		"not discoverable": hidden.ID, "myself": me.ID,
	} {
		if contains(got, id) {
			t.Errorf("%s must not be shown", name)
		}
	}
}

func TestDiscoveryRequiresCompleteProfile(t *testing.T) {
	e := newEnv(t)
	u := e.register("newbie")
	r := e.expect(e.do(http.MethodGet, "/discover", u.Token, nil), http.StatusForbidden)
	if r.json(t)["code"] != "profile_incomplete" {
		t.Fatalf("expected profile_incomplete code: %s", r.Body)
	}
	other := e.newPerson(persona{Name: "Other", Gender: "woman", InterestedIn: []string{"man"}})
	e.expect(e.do(http.MethodPost, "/swipes", u.Token, map[string]string{"userId": other.ID, "action": "like"}), http.StatusForbidden)
}

func TestSwipeRulesAndIdempotence(t *testing.T) {
	e := newEnv(t)
	a := e.newPerson(persona{Name: "Anna", Gender: "woman", InterestedIn: []string{"man"}})
	b := e.newPerson(persona{Name: "Bill", Gender: "man", InterestedIn: []string{"woman"}})
	c := e.newPerson(persona{Name: "Carl", Gender: "man", InterestedIn: []string{"woman"}})

	// Invalid input.
	e.expect(e.do(http.MethodPost, "/swipes", a.Token, map[string]string{"userId": a.ID, "action": "like"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/swipes", a.Token, map[string]string{"userId": b.ID, "action": "superlike"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/swipes", a.Token, map[string]string{"userId": "nope", "action": "like"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/swipes", a.Token, map[string]string{"userId": "00000000-0000-0000-0000-000000000000", "action": "like"}), http.StatusNotFound)

	// Repeated likes never duplicate anything; the first decision wins.
	first := e.swipe(a, b, "like", http.StatusCreated)
	again := e.swipe(a, b, "like", http.StatusOK)
	if first.AlreadyProcessed || !again.AlreadyProcessed {
		t.Fatalf("unexpected idempotence flags: %+v %+v", first, again)
	}
	flip := e.swipe(a, b, "pass", http.StatusOK)
	if flip.Action != "like" || !flip.AlreadyProcessed {
		t.Fatalf("a processed profile cannot be re-decided: %+v", flip)
	}
	if n := countRows(t, e.pool, `SELECT count(*) FROM swipes WHERE from_user_id=$1`, a.ID); n != 1 {
		t.Fatalf("expected 1 swipe row, got %d", n)
	}

	// A pass never matches, even if the other side likes.
	e.swipe(a, c, "pass", http.StatusCreated)
	out := e.swipe(c, a, "like", http.StatusCreated)
	if out.Matched {
		t.Fatal("a pass must not produce a match")
	}

	// Mutual like -> exactly one match, same id from both sides, even on repeats.
	m1 := e.swipe(b, a, "like", http.StatusCreated)
	m2 := e.swipe(b, a, "like", http.StatusOK)
	m3 := e.swipe(a, b, "like", http.StatusOK)
	if !m1.Matched || m1.MatchID == "" || m2.MatchID != m1.MatchID || m3.MatchID != m1.MatchID {
		t.Fatalf("inconsistent matches: %+v %+v %+v", m1, m2, m3)
	}
	if n := countRows(t, e.pool, `SELECT count(*) FROM matches`); n != 1 {
		t.Fatalf("expected a single match row, got %d", n)
	}
	if n := countRows(t, e.pool, `SELECT count(*) FROM conversation_participants`); n != 2 {
		t.Fatalf("expected 2 participants, got %d", n)
	}
}

func TestSimultaneousLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	const pairs = 12
	type pair struct{ a, b user }
	all := make([]pair, pairs)
	for i := range all {
		all[i] = pair{
			a: e.newPerson(persona{Name: "Aa", Gender: "woman", InterestedIn: []string{"man"}}),
			b: e.newPerson(persona{Name: "Bb", Gender: "man", InterestedIn: []string{"woman"}}),
		}
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan swipeOut, pairs*2)
	for _, p := range all {
		for _, dir := range [][2]user{{p.a, p.b}, {p.b, p.a}} {
			wg.Add(1)
			go func(from, to user) {
				defer wg.Done()
				<-start
				r := e.do(http.MethodPost, "/swipes", from.Token, map[string]string{"userId": to.ID, "action": "like"})
				if r.Status != http.StatusCreated {
					t.Errorf("swipe failed: %d %s", r.Status, r.Body)
					return
				}
				var out swipeOut
				r.decode(t, &out)
				results <- out
			}(dir[0], dir[1])
		}
	}
	close(start)
	wg.Wait()
	close(results)

	matched := 0
	for out := range results {
		if out.Matched {
			matched++
		}
	}
	if n := countRows(t, e.pool, `SELECT count(*) FROM matches`); n != pairs {
		t.Fatalf("expected exactly %d matches (one per pair), got %d", pairs, n)
	}
	if matched < pairs {
		t.Fatalf("at least one side of each pair must learn about the match, got %d/%d", matched, pairs)
	}
}

func TestDiscoveryPrefersPeopleWhoLikedYou(t *testing.T) {
	e := newEnv(t)
	me := e.newPerson(persona{Name: "Me", Gender: "man", InterestedIn: []string{"woman"}})
	var fans []user
	for i := 0; i < 4; i++ {
		fans = append(fans, e.newPerson(persona{Name: "Cand", Gender: "woman", InterestedIn: []string{"man"}}))
	}
	// The *oldest-registered* candidate likes me; ranking should still put her first.
	e.swipe(fans[0], me, "like", http.StatusCreated)
	ids := e.discoverIDs(me)
	if len(ids) != 4 || ids[0] != fans[0].ID {
		t.Fatalf("the person who liked me should come first, got %v", ids)
	}
}
