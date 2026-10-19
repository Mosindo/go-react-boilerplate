package server_test

import (
	"fmt"
	"sync"
	"testing"
)

func TestDiscoverRespectsMutualPreferences(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("Adam", man)                                                                                                      // man -> woman, 18-99
	b := e.onboard("Beth", woman)                                                                                                    // woman -> man
	c := e.onboard("Cleo", profileSpec{gender: "woman", birth: "1994-01-01", interestedIn: []string{"woman"}})                       // woman -> woman
	d := e.onboard("Dora", profileSpec{gender: "woman", birth: "1994-01-01", interestedIn: []string{"man"}, minAge: 40, maxAge: 60}) // wants 40-60
	f := e.onboard("Fran", profileSpec{gender: "woman", birth: "1960-01-01", interestedIn: []string{"man"}})                         // old
	young := e.onboard("Yan", profileSpec{gender: "woman", birth: "2001-06-01", interestedIn: []string{"man"}})
	_ = f

	got := a.discover()
	if !contains(got, b.ID) || !contains(got, young.ID) {
		t.Fatalf("Adam should see Beth and Yan, got %+v", got)
	}
	if contains(got, c.ID) {
		t.Fatal("Cleo is not interested in men: she must not appear")
	}
	if contains(got, d.ID) {
		t.Fatal("Dora's age range excludes Adam: she must not appear")
	}
	if contains(got, a.ID) {
		t.Fatal("never show yourself")
	}
	// Adam (man, interested in women 18-99) must not see other men either.
	e2 := e.onboard("Eli", man)
	if contains(a.discover(), e2.ID) {
		t.Fatal("men must not appear to someone interested only in women")
	}
	for _, p := range got {
		if len(p.Photos) == 0 {
			t.Fatal("candidates always have a photo")
		}
	}
}

func TestDiscoverRequiresCompleteProfile(t *testing.T) {
	e := newEnv(t)
	u := e.register("Zed")
	if r := u.do("GET", "/discover", nil); r.Status != 409 || r.code() != "profile_incomplete" {
		t.Fatalf("expected profile_incomplete, got %d %s", r.Status, r.Body)
	}
	if r := u.do("POST", "/swipes", map[string]string{"userId": u.ID, "action": "like"}); r.Status != 409 {
		t.Fatalf("swiping requires a complete profile, got %d", r.Status)
	}
}

func TestDiscoverHidesIncompleteAndHiddenProfiles(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("Adam", man)
	noPhoto := e.register("Nophoto")
	noPhoto.expect(200, "PUT", "/profile", map[string]any{"firstName": "Nophoto", "birthDate": "1994-01-01", "gender": "woman"})
	noPhoto.expect(200, "PUT", "/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 99})
	ghost := e.onboard("Ghost", woman)
	ghost.expect(200, "PUT", "/profile", map[string]any{"firstName": "Ghost", "birthDate": "1994-01-01", "gender": "woman", "isVisible": false})

	got := a.discover()
	if contains(got, noPhoto.ID) {
		t.Fatal("profiles without photos must not be shown")
	}
	if contains(got, ghost.ID) {
		t.Fatal("hidden profiles must not be shown")
	}
	a.expect(404, "POST", "/swipes", map[string]string{"userId": ghost.ID, "action": "like"})
	a.expect(404, "GET", "/profiles/"+ghost.ID, nil)
}

func TestSwipeAndMatchFlow(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("Adam", man)
	b := e.onboard("Beth", woman)

	res := a.swipe(b, "like")
	if res.Matched || res.Duplicate {
		t.Fatalf("one-sided like must not match: %+v", res)
	}
	if contains(a.discover(), b.ID) {
		t.Fatal("a swiped profile must never come back")
	}
	if dup := a.swipe(b, "like"); !dup.Duplicate || dup.Matched {
		t.Fatalf("repeated like is idempotent: %+v", dup)
	}
	if dup := a.swipe(b, "pass"); !dup.Duplicate {
		t.Fatal("first decision stands: a later pass is a no-op")
	}

	// Beth sees Adam in "likes received" and (ranked first) in discover.
	var likes struct {
		Profiles []card `json:"profiles"`
	}
	b.expect(200, "GET", "/likes/received", nil).decode(t, &likes)
	if len(likes.Profiles) != 1 || likes.Profiles[0].UserID != a.ID {
		t.Fatalf("expected Adam in received likes: %+v", likes)
	}

	res = b.swipe(a, "like")
	if !res.Matched || res.Match == nil || res.Match.ConversationID == "" {
		t.Fatalf("mutual like must match: %+v", res)
	}
	if again := b.swipe(a, "like"); !again.Duplicate || !again.Matched {
		t.Fatalf("liking again keeps the match, no new one: %+v", again)
	}

	var n int
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM matches`).Scan(&n)
	if n != 1 {
		t.Fatalf("exactly one match expected, got %d", n)
	}
	b.expect(200, "GET", "/likes/received", nil).decode(t, &likes)
	if len(likes.Profiles) != 0 {
		t.Fatal("received likes must not list profiles already swiped")
	}

	var matches struct {
		Matches []struct {
			ID      string `json:"id"`
			Profile card   `json:"profile"`
		} `json:"matches"`
	}
	a.expect(200, "GET", "/matches", nil).decode(t, &matches)
	if len(matches.Matches) != 1 || matches.Matches[0].Profile.UserID != b.ID {
		t.Fatalf("Adam should be matched with Beth: %+v", matches)
	}

	// Validation.
	a.expect(400, "POST", "/swipes", map[string]string{"userId": b.ID, "action": "superlike"})
	a.expect(400, "POST", "/swipes", map[string]string{"userId": "not-a-uuid", "action": "like"})
	a.expect(404, "POST", "/swipes", map[string]string{"userId": a.ID, "action": "like"})
	a.expect(404, "POST", "/swipes", map[string]string{"userId": "00000000-0000-0000-0000-000000000000", "action": "like"})

	// Unmatch removes the conversation for both and never resurfaces the profile.
	a.expect(204, "DELETE", "/matches/"+matches.Matches[0].ID, nil)
	b.expect(404, "GET", "/conversations/"+res.Match.ConversationID, nil)
	if contains(a.discover(), b.ID) || contains(b.discover(), a.ID) {
		t.Fatal("unmatched users must not reappear in discover")
	}
	b.expect(404, "DELETE", "/matches/"+matches.Matches[0].ID, nil)
}

func TestPassNeverMatches(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("Adam", man)
	b := e.onboard("Beth", woman)
	a.swipe(b, "pass")
	if res := b.swipe(a, "like"); res.Matched {
		t.Fatal("a pass must not turn into a match")
	}
}

func TestConcurrentMutualLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	const pairs = 12
	type pair struct{ a, b *user }
	var ps []pair
	for i := 0; i < pairs; i++ {
		ps = append(ps, pair{e.onboard(fmt.Sprintf("M%d", i), man), e.onboard(fmt.Sprintf("W%d", i), woman)})
	}
	var wg sync.WaitGroup
	matchedReports := make([]int, pairs)
	var mu sync.Mutex
	for i, p := range ps {
		for _, side := range []struct{ from, to *user }{{p.a, p.b}, {p.b, p.a}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				r := side.from.do("POST", "/swipes", map[string]string{"userId": side.to.ID, "action": "like"})
				if r.Status != 200 {
					t.Errorf("swipe: %d %s", r.Status, r.Body)
					return
				}
				var out swipeResp
				r.decode(t, &out)
				if out.Matched && !out.Duplicate {
					mu.Lock()
					matchedReports[i]++
					mu.Unlock()
				}
			}()
		}
	}
	wg.Wait()
	var matches, convs int
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM matches`).Scan(&matches)
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM conversations`).Scan(&convs)
	if matches != pairs || convs != pairs {
		t.Fatalf("expected %d matches/conversations, got %d/%d", pairs, matches, convs)
	}
	for i, n := range matchedReports {
		if n != 1 {
			t.Errorf("pair %d: exactly one request should have created the match, got %d", i, n)
		}
	}
}

func TestDistanceFilteringAndPrivacy(t *testing.T) {
	e := newEnv(t)
	paris := [2]float64{48.8566, 2.3522}
	versailles := [2]float64{48.8049, 2.1204} // ~17 km
	lyon := [2]float64{45.7640, 4.8357}       // ~390 km

	spec := func(g string, interestedIn string, loc [2]float64, maxDist *int) profileSpec {
		return profileSpec{gender: g, birth: "1994-01-01", interestedIn: []string{interestedIn}, lat: ptr(loc[0]), lng: ptr(loc[1]), maxDist: maxDist}
	}
	a := e.onboard("Adam", spec("man", "woman", paris, ptr(50)))
	near := e.onboard("Near", spec("woman", "man", versailles, nil))
	far := e.onboard("Far", spec("woman", "man", lyon, nil))
	picky := e.onboard("Picky", spec("woman", "man", versailles, ptr(5))) // only wants people within 5 km

	got := a.discover()
	if !contains(got, near.ID) {
		t.Fatal("Near is within Adam's 50 km")
	}
	if contains(got, far.ID) {
		t.Fatal("Far is beyond Adam's max distance")
	}
	if contains(got, picky.ID) {
		t.Fatal("Picky only accepts people within 5 km")
	}
	for _, c := range got {
		if c.UserID == near.ID {
			if c.DistanceKm == nil || *c.DistanceKm%5 != 0 || *c.DistanceKm < 15 || *c.DistanceKm > 25 {
				t.Fatalf("distance must be approximate (multiple of 5), got %v", c.DistanceKm)
			}
		}
	}

	// Opting out of distance display hides it for others.
	near.expect(200, "PUT", "/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 99, "showDistance": false})
	for _, c := range a.discover() {
		if c.UserID == near.ID && c.DistanceKm != nil {
			t.Fatal("distance must be hidden when the user disabled it")
		}
	}
	r := a.expect(200, "GET", "/profiles/"+near.ID, nil)
	if got := string(r.Body); contains2(got, "lat") || contains2(got, "lng") || contains2(got, "birth") || contains2(got, "email") {
		t.Fatalf("public profile leaks private fields: %s", got)
	}
}

func contains2(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
