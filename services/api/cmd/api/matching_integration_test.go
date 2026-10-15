package main

import (
	"context"
	"net/http"
	"sync"
	"testing"
)

// cast creates a small cast around one viewer ("Vera", a woman seeking men 25-35 within 50 km).
func TestDiscoveryRespectsMutualPreferences(t *testing.T) {
	e := newEnv(t)
	// Paris area. Everyone below is judged against Vera.
	vera := e.signup(profileSpec{Name: "Vera", Gender: "woman", Seeks: []string{"man"}, Age: 30, MinAge: 25, MaxAge: 35, Distance: 50})

	fit := e.signup(profileSpec{Name: "Fit", Gender: "man", Seeks: []string{"woman"}, Age: 31})
	wrongGender := e.signup(profileSpec{Name: "Wrong", Gender: "woman", Seeks: []string{"woman", "man"}, Age: 30})
	tooYoung := e.signup(profileSpec{Name: "Young", Gender: "man", Seeks: []string{"woman"}, Age: 22})
	tooOld := e.signup(profileSpec{Name: "Old", Gender: "man", Seeks: []string{"woman"}, Age: 45})
	notSeekingHer := e.signup(profileSpec{Name: "Gay", Gender: "man", Seeks: []string{"man"}, Age: 30})
	herAgeRejected := e.signup(profileSpec{Name: "Picky", Gender: "man", Seeks: []string{"woman"}, Age: 30, MinAge: 18, MaxAge: 25})
	tooFar := e.signup(profileSpec{Name: "Far", Gender: "man", Seeks: []string{"woman"}, Age: 30, Lat: 43.2965, Lng: 5.3698}) // Marseille
	theirRadius := e.signup(profileSpec{Name: "Home", Gender: "man", Seeks: []string{"woman"}, Age: 30, Lat: 48.95, Lng: 2.45, Distance: 3})
	noPhoto := e.signup(profileSpec{Name: "Blank", Gender: "man", Seeks: []string{"woman"}, Age: 30, NoPhoto: true})
	hidden := e.signup(profileSpec{Name: "Ghost", Gender: "man", Seeks: []string{"woman"}, Age: 30})
	e.mustStatus(e.do(http.MethodPatch, "/me/privacy", hidden.Token, map[string]any{"isVisible": false}), http.StatusOK, "hide")
	blockedByHer := e.signup(profileSpec{Name: "Blocked", Gender: "man", Seeks: []string{"woman"}, Age: 30})
	e.mustStatus(e.do(http.MethodPost, "/blocks/"+blockedByHer.ID, vera.Token, nil), http.StatusNoContent, "block")
	blockedHer := e.signup(profileSpec{Name: "Blocker", Gender: "man", Seeks: []string{"woman"}, Age: 30})
	e.mustStatus(e.do(http.MethodPost, "/blocks/"+vera.ID, blockedHer.Token, nil), http.StatusNoContent, "block her")

	ids := e.discoverIDs(vera)
	if !contains(ids, fit.ID) {
		t.Fatalf("a compatible person must be proposed, got %v", ids)
	}
	for name, excluded := range map[string]*person{
		"wrong gender": wrongGender, "too young": tooYoung, "too old": tooOld, "not seeking her": notSeekingHer,
		"her age rejects them": herAgeRejected, "too far": tooFar, "outside their radius": theirRadius,
		"no photo": noPhoto, "hidden": hidden, "blocked by her": blockedByHer, "blocked her": blockedHer, "herself": vera,
	} {
		if contains(ids, excluded.ID) {
			t.Errorf("%s must not be proposed", name)
		}
	}

	// Profiles are also protected when requested directly.
	for name, hiddenOne := range map[string]*person{"wrong gender": wrongGender, "blocked": blockedByHer, "far": tooFar, "hidden": hidden} {
		e.mustStatus(e.do(http.MethodGet, "/profiles/"+hiddenOne.ID, vera.Token, nil), http.StatusNotFound, "direct access "+name)
	}
	e.mustStatus(e.do(http.MethodGet, "/profiles/"+fit.ID, vera.Token, nil), http.StatusOK, "direct access to a fit profile")
	e.mustStatus(e.do(http.MethodGet, "/profiles/"+vera.ID, vera.Token, nil), http.StatusNotFound, "own profile via public route")
	e.mustStatus(e.do(http.MethodGet, "/profiles/not-a-uuid", vera.Token, nil), http.StatusNotFound, "malformed id")

	// Swiping is checked server-side with the same rules.
	for name, target := range map[string]*person{"wrong gender": wrongGender, "far": tooFar, "no photo": noPhoto, "blocked": blockedByHer} {
		r := e.swipe(vera, target, "like")
		if r.Status != http.StatusNotFound {
			t.Errorf("liking an ineligible profile (%s) must fail, got %d %s", name, r.Status, r.Body)
		}
	}
	var likes int
	_ = e.pool.QueryRow(contextBackground(), `SELECT COUNT(*) FROM likes WHERE from_user_id = $1`, vera.ID).Scan(&likes)
	if likes != 0 {
		t.Fatalf("rejected swipes must not be stored, found %d", likes)
	}
}

func TestDiscoveryPaginationAndExclude(t *testing.T) {
	e := newEnv(t)
	viewer := e.signup(profileSpec{Name: "Viewer", Gender: "woman", Seeks: []string{"man"}, Lat: 40.0, Lng: 10.0})
	var men []*person
	for _, name := range []string{"A", "B", "C", "D", "E"} {
		men = append(men, e.signup(profileSpec{Name: name, Gender: "man", Seeks: []string{"woman"}, Lat: 40.0, Lng: 10.0}))
	}
	first := e.mustStatus(e.do(http.MethodGet, "/discover?limit=2", viewer.Token, nil), http.StatusOK, "page 1").json(t)["profiles"].([]any)
	if len(first) != 2 {
		t.Fatalf("limit must be honoured, got %d", len(first))
	}
	seen := []string{first[0].(map[string]any)["userId"].(string), first[1].(map[string]any)["userId"].(string)}
	next := e.mustStatus(e.do(http.MethodGet, "/discover?limit=20&exclude="+seen[0]+","+seen[1], viewer.Token, nil), http.StatusOK, "page 2").json(t)["profiles"].([]any)
	for _, raw := range next {
		id := raw.(map[string]any)["userId"].(string)
		if contains(seen, id) {
			t.Fatalf("excluded profile %s came back", id)
		}
	}
	_ = men

	for _, bad := range []string{"limit=0", "limit=abc", "limit=-3", "exclude=zzz"} {
		e.mustStatus(e.do(http.MethodGet, "/discover?"+bad, viewer.Token, nil), http.StatusBadRequest, "bad query "+bad)
	}
	if r := e.mustStatus(e.do(http.MethodGet, "/discover?limit=1000", viewer.Token, nil), http.StatusOK, "huge limit").json(t); len(r["profiles"].([]any)) > 20 {
		t.Fatal("limit must be clamped")
	}
}

func TestLikePassAndMatch(t *testing.T) {
	e := newEnv(t)
	a := e.signup(profileSpec{Name: "Alice", Gender: "woman", Seeks: []string{"man"}, Lat: 41.0, Lng: 11.0})
	b := e.signup(profileSpec{Name: "Bob", Gender: "man", Seeks: []string{"woman"}, Lat: 41.0, Lng: 11.0})
	c := e.signup(profileSpec{Name: "Carl", Gender: "man", Seeks: []string{"woman"}, Lat: 41.0, Lng: 11.0})

	// A likes B: nothing visible yet.
	r := e.mustStatus(e.swipe(a, b, "like"), http.StatusOK, "like").json(t)
	if r["matched"] != false {
		t.Fatalf("a one-sided like is not a match: %v", r)
	}
	// Decisions are final and never duplicated.
	dup := e.swipe(a, b, "like")
	if dup.Status != http.StatusConflict || dup.code(t) != "already_swiped" {
		t.Fatalf("repeated like must conflict, got %d %s", dup.Status, dup.Body)
	}
	if e.swipe(a, b, "pass").Status != http.StatusConflict {
		t.Fatal("changing a decision by swiping again is not allowed")
	}

	// Processed profiles never come back.
	if contains(e.discoverIDs(a), b.ID) {
		t.Fatal("a liked profile must not reappear")
	}
	e.mustStatus(e.swipe(a, c, "pass"), http.StatusOK, "pass")
	if contains(e.discoverIDs(a), c.ID) {
		t.Fatal("a passed profile must not reappear")
	}

	// B sees A first and knows A liked them.
	cards := e.mustStatus(e.do(http.MethodGet, "/discover", b.Token, nil), http.StatusOK, "discover").json(t)["profiles"].([]any)
	top := cards[0].(map[string]any)
	if top["userId"] != a.ID || top["likedMe"] != true {
		t.Fatalf("someone who liked you should come first: %v", top)
	}

	// Validation.
	e.mustStatus(e.do(http.MethodPost, "/swipes", b.Token, map[string]string{"userId": a.ID, "action": "superlike"}), http.StatusBadRequest, "bad action")
	e.mustStatus(e.do(http.MethodPost, "/swipes", b.Token, map[string]string{"userId": "nope", "action": "like"}), http.StatusBadRequest, "bad id")
	e.mustStatus(e.do(http.MethodPost, "/swipes", b.Token, map[string]string{"userId": b.ID, "action": "like"}), http.StatusBadRequest, "self like")

	// B likes A back: match.
	m := e.mustStatus(e.swipe(b, a, "like"), http.StatusOK, "reciprocal like").json(t)
	if m["matched"] != true {
		t.Fatalf("expected a match: %v", m)
	}
	match := m["match"].(map[string]any)
	if match["userId"] != a.ID || match["firstName"] != "Alice" || match["conversationId"] == "" {
		t.Fatalf("unexpected match payload: %v", match)
	}

	var matches, conversations int
	_ = e.pool.QueryRow(contextBackground(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&matches)
	_ = e.pool.QueryRow(contextBackground(), `SELECT COUNT(*) FROM conversations WHERE match_id = $1`, match["id"]).Scan(&conversations)
	if matches != 1 || conversations != 1 {
		t.Fatalf("expected exactly one match and one conversation, got %d and %d", matches, conversations)
	}

	// A passed profile that later likes back does not create a match.
	e.mustStatus(e.swipe(c, a, "like"), http.StatusOK, "carl likes alice")
	var n int
	_ = e.pool.QueryRow(contextBackground(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, c.ID).Scan(&n)
	if n != 0 {
		t.Fatal("a pass followed by a like from the other side must not match")
	}

	// A "pass" never matches even if the other liked.
	d := e.signup(profileSpec{Name: "Dan", Gender: "man", Seeks: []string{"woman"}, Lat: 41.0, Lng: 11.0})
	e.mustStatus(e.swipe(d, a, "like"), http.StatusOK, "dan likes")
	if res := e.mustStatus(e.swipe(a, d, "pass"), http.StatusOK, "alice passes").json(t); res["matched"] == true {
		t.Fatal("a pass can never produce a match")
	}
}

func TestSimultaneousLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	a := e.signup(profileSpec{Name: "Ann", Gender: "woman", Seeks: []string{"man"}, Lat: 42.0, Lng: 12.0})
	b := e.signup(profileSpec{Name: "Ben", Gender: "man", Seeks: []string{"woman"}, Lat: 42.0, Lng: 12.0})

	var wg sync.WaitGroup
	results := make([]bool, 2)
	for i, pair := range [][2]*person{{a, b}, {b, a}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := e.swipe(pair[0], pair[1], "like")
			if r.Status != http.StatusOK {
				t.Errorf("swipe failed: %d %s", r.Status, r.Body)
				return
			}
			results[i] = r.json(t)["matched"] == true
		}()
	}
	wg.Wait()

	matched := 0
	for _, m := range results {
		if m {
			matched++
		}
	}
	if matched != 1 {
		t.Fatalf("exactly one of the two simultaneous likes must report the match, got %d", matched)
	}
	var n int
	_ = e.pool.QueryRow(contextBackground(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("expected one match row, got %d", n)
	}
}

func TestUnmatchRemovesConversationButNotDecisions(t *testing.T) {
	e := newEnv(t)
	a := e.signup(profileSpec{Name: "Una", Gender: "woman", Seeks: []string{"man"}, Lat: 43.0, Lng: 13.0})
	b := e.signup(profileSpec{Name: "Ulf", Gender: "man", Seeks: []string{"woman"}, Lat: 43.0, Lng: 13.0})
	conv := e.match(a, b)
	e.mustStatus(e.do(http.MethodPost, "/chats/"+conv+"/messages", a.Token, map[string]string{"body": "hello"}), http.StatusCreated, "message")

	e.mustStatus(e.do(http.MethodDelete, "/matches/"+b.ID, a.Token, nil), http.StatusNoContent, "unmatch")
	e.mustStatus(e.do(http.MethodDelete, "/matches/"+b.ID, a.Token, nil), http.StatusNotFound, "unmatch twice")
	e.mustStatus(e.do(http.MethodGet, "/chats/"+conv+"/messages", b.Token, nil), http.StatusNotFound, "conversation gone for the other side")
	var msgs int
	_ = e.pool.QueryRow(contextBackground(), `SELECT COUNT(*) FROM messages WHERE conversation_id = $1`, conv).Scan(&msgs)
	if msgs != 0 {
		t.Fatal("messages must be deleted with the match")
	}
	if contains(e.discoverIDs(a), b.ID) || contains(e.discoverIDs(b), a.ID) {
		t.Fatal("unmatched people must not be proposed again")
	}
}

func contextBackground() context.Context { return context.Background() }
